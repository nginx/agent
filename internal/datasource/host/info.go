// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package host

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/shirou/gopsutil/host"
	"golang.org/x/sync/singleflight"
)

const (
	dockerEnvLocation      = "/.dockerenv"
	containerEnvLocation   = "/run/.containerenv"
	k8sServiceAcctLocation = "/var/run/secrets/kubernetes.io/serviceaccount"

	selfCgroupLocation = "/proc/self/cgroup"
	mountInfoLocation  = "/proc/self/mountinfo"
	osReleaseLocation  = "/etc/os-release"

	k8sKind    = "kubepods"
	docker     = "docker"
	conatinerd = "containerd"

	lengthOfContainerID = 64

	versionID = "VERSION_ID"
	version   = "VERSION"
	codeName  = "VERSION_CODENAME"
	id        = "ID"
	name      = "NAME"

	IsContainerKey    = "IsContainer"
	GetContainerIDKey = "GetContainerID"
	GetSystemUUIDKey  = "GetSystemUUIDKey"
)

var (
	singleflightGroup = &singleflight.Group{}

	basePattern       = regexp.MustCompile("/([a-f0-9]{64})$")
	colonPattern      = regexp.MustCompile(":([a-f0-9]{64})$")
	scopePattern      = regexp.MustCompile(`/.+-(.+?).scope$`)
	containersPattern = regexp.MustCompile("containers/([a-f0-9]{64})")
	containerdPattern = regexp.MustCompile("sandboxes/([a-f0-9]{64})")
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . InfoInterface

type (
	InfoInterface interface {
		IsContainer() bool
		GetContainerInfo() *v1.Resource_ContainerInfo
		GetHostInfo(ctx context.Context) *v1.Resource_HostInfo
	}

	Info struct {
		// containerSpecificFiles are files that are only created in containers.
		// We this to determine if an instance is running in a container or not
		containerSpecificFiles []string
		selfCgroupLocation     string
		mountInfoLocation      string
	}
)

func NewInfo() *Info {
	return &Info{
		containerSpecificFiles: []string{
			dockerEnvLocation,
			containerEnvLocation,
			k8sServiceAcctLocation,
		},
		selfCgroupLocation: selfCgroupLocation,
		mountInfoLocation:  mountInfoLocation,
	}
}

func (i *Info) IsContainer() bool {
	res, err, _ := singleflightGroup.Do(IsContainerKey, func() (interface{}, error) {
		for _, filename := range i.containerSpecificFiles {
			if _, err := os.Stat(filename); err == nil {
				return true, nil
			}
		}

		return containsContainerReference(i.selfCgroupLocation), nil
	})

	if err != nil {
		slog.Warn("Unable to determine if resource is a container or not", "error", err)
		return false
	}

	if result, ok := res.(bool); ok {
		return result
	}

	return false
}

func (i *Info) GetContainerInfo() *v1.Resource_ContainerInfo {
	return &v1.Resource_ContainerInfo{
		ContainerInfo: &v1.ContainerInfo{
			ContainerId: i.getContainerID(),
			Image:       "",
		},
	}
}

func (*Info) GetHostInfo(ctx context.Context) *v1.Resource_HostInfo {
	hostname, err := os.Hostname()
	if err != nil {
		slog.WarnContext(ctx, "Unable to get hostname", "error", err)
	}

	return &v1.Resource_HostInfo{
		HostInfo: &v1.HostInfo{
			Id:          getHostID(ctx),
			Hostname:    hostname,
			ReleaseInfo: getReleaseInfo(ctx, osReleaseLocation),
		},
	}
}

func containsContainerReference(cgroupFile string) bool {
	data, err := os.ReadFile(cgroupFile)
	if err != nil {
		slog.Error("Unable to check if cgroup file contains a container reference", "error", err)
		return false
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, k8sKind) || strings.Contains(line, docker) || strings.Contains(line, conatinerd) {
			return true
		}
	}

	return false
}

func (i *Info) getContainerID() string {
	res, err, _ := singleflightGroup.Do(GetContainerIDKey, func() (interface{}, error) {
		containerID, err := getContainerIDFromMountInfo(i.mountInfoLocation)
		return uuid.NewMD5(uuid.NameSpaceDNS, []byte(containerID)).String(), err
	})

	if err != nil {
		slog.Error("Could not get container ID", "error", err)
		return ""
	}

	if result, ok := res.(string); ok {
		return result
	}

	return ""
}

// getContainerID returns the container ID of the current running environment.
// Supports cgroup v1 and v2. Reading "/proc/1/cpuset" would only work for cgroups v1
// mountInfo is the path: "/proc/self/mountinfo"
func getContainerIDFromMountInfo(mountInfo string) (string, error) {
	mInfoFile, err := os.Open(mountInfo)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", mountInfo, err)
	}
	defer func(f *os.File, fileName string) {
		closeErr := f.Close()
		slog.Error("Unable to close file %s: %w", fileName, closeErr)
	}(mInfoFile, mountInfo)

	fileScanner := bufio.NewScanner(mInfoFile)
	fileScanner.Split(bufio.ScanLines)

	var lines []string
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}

	for _, line := range lines {
		splitLine := strings.Split(line, " ")
		for _, word := range splitLine {
			containerID := getContainerIDFromPatterns(word)
			if containerID != "" {
				return containerID, nil
			}
		}
	}

	return "", fmt.Errorf("container ID not found in %s", mountInfo)
}

func getContainerIDFromPatterns(word string) string {
	slices := scopePattern.FindStringSubmatch(word)
	if containsContainerID(slices) {
		return slices[1]
	}

	slices = basePattern.FindStringSubmatch(word)
	if containsContainerID(slices) {
		return slices[1]
	}

	slices = colonPattern.FindStringSubmatch(word)
	if containsContainerID(slices) {
		return slices[1]
	}

	slices = containersPattern.FindStringSubmatch(word)
	if containsContainerID(slices) {
		return slices[1]
	}

	slices = containerdPattern.FindStringSubmatch(word)
	if containsContainerID(slices) {
		return slices[1]
	}

	return ""
}

func containsContainerID(slices []string) bool {
	return len(slices) >= 2 && len(slices[1]) == lengthOfContainerID
}

func getHostID(ctx context.Context) string {
	res, err, _ := singleflightGroup.Do(GetSystemUUIDKey, func() (interface{}, error) {
		var err error

		hostID, err := host.HostIDWithContext(ctx)
		if err != nil {
			slog.WarnContext(ctx, "Unable to get host ID", "error", err)
			return "", err
		}

		return uuid.NewMD5(uuid.Nil, []byte(hostID)).String(), err
	})

	if err != nil {
		slog.WarnContext(ctx, "Unable to get host ID", "error", err)
		return ""
	}

	if result, ok := res.(string); ok {
		return result
	}

	return ""
}

func getReleaseInfo(ctx context.Context, osReleaseFile string) (releaseInfo *v1.ReleaseInfo) {
	hostReleaseInfo := getHostReleaseInfo(ctx)
	osRelease, err := getOsRelease(osReleaseFile)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read from os release file: %w", err)

		return hostReleaseInfo
	}

	return mergeHostAndOsReleaseInfo(hostReleaseInfo, osRelease)
}

func getHostReleaseInfo(ctx context.Context) (releaseInfo *v1.ReleaseInfo) {
	hostInfo, err := host.InfoWithContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Could not read release information for host: %w", err)
		return &v1.ReleaseInfo{}
	}

	return &v1.ReleaseInfo{
		VersionId: hostInfo.PlatformVersion,
		Version:   hostInfo.KernelVersion,
		Codename:  hostInfo.OS,
		Name:      hostInfo.PlatformFamily,
		Id:        hostInfo.Platform,
	}
}

func getOsRelease(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("release file %s is unreadable: %w", path, err)
	}
	defer func(f *os.File, fileName string) {
		closeErr := f.Close()
		slog.Error("Unable to close file %s: %w", fileName, closeErr)
	}(f, path)

	info, err := parseOsReleaseFile(f)
	if err != nil {
		return nil, fmt.Errorf("release file %s is unparsable: %w", path, err)
	}

	return info, nil
}

func parseOsReleaseFile(reader io.Reader) (map[string]string, error) {
	osReleaseInfoMap := map[string]string{"NAME": "unix"}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		field := strings.Split(line, "=")
		if len(field) < 2 {
			continue
		}
		osReleaseInfoMap[field[0]] = strings.Trim(field[1], "\"")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not parse os release file %w", err)
	}

	return osReleaseInfoMap, nil
}

func mergeHostAndOsReleaseInfo(releaseInfo *v1.ReleaseInfo,
	osReleaseInfo map[string]string,
) (release *v1.ReleaseInfo) {
	if len(osReleaseInfo[versionID]) == 0 {
		osReleaseInfo[versionID] = releaseInfo.GetVersionId()
	}
	if len(osReleaseInfo[version]) == 0 {
		osReleaseInfo[version] = releaseInfo.GetVersion()
	}
	if len(osReleaseInfo[codeName]) == 0 {
		osReleaseInfo[codeName] = releaseInfo.GetCodename()
	}
	if len(osReleaseInfo[name]) == 0 {
		osReleaseInfo[name] = releaseInfo.GetName()
	}
	if len(osReleaseInfo[id]) == 0 {
		osReleaseInfo[id] = releaseInfo.GetId()
	}

	return &v1.ReleaseInfo{
		VersionId: osReleaseInfo[versionID],
		Version:   osReleaseInfo[version],
		Codename:  osReleaseInfo[codeName],
		Name:      osReleaseInfo[name],
		Id:        osReleaseInfo[id],
	}
}
