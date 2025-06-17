// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package host

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
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
	containerd = "containerd"

	numberOfKeysAndValues = 2
	lengthOfContainerID   = 64

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

	// example: /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4
	basePattern = regexp.MustCompile("/([a-f0-9]{64})$")
	// nolint: lll
	// example: /system.slice/containerd.service/kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80
	colonPattern = regexp.MustCompile(":([a-f0-9]{64})$")
	// example: /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope
	scopePattern = regexp.MustCompile(`/.+-(.+?).scope$`)
	// nolint: lll
	// example: /containers/storage/overlay-containers/ba0be90007be48bca767be0a462390ad2c9b0e910608158f79c8d6a984302b7e/userdata/hostname
	containersPattern = regexp.MustCompile("containers/([a-f0-9]{64})")
	// nolint: lll
	// example: /var/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/d7cb24ec5dede02990283dec30bd1e6ae1f93e3e19b152b708b7e0e133c6baec/hostname
	containerdPattern = regexp.MustCompile("sandboxes/([a-f0-9]{64})")
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . InfoInterface

type (
	InfoInterface interface {
		IsContainer() bool
		ResourceID(ctx context.Context) string
		ContainerInfo(ctx context.Context) *v1.Resource_ContainerInfo
		HostInfo(ctx context.Context) *v1.Resource_HostInfo
	}

	Info struct {
		// containerSpecificFiles are files that are only created in containers.
		// We use this to determine if an instance is running in a container or not
		exec                   exec.ExecInterface
		selfCgroupLocation     string
		mountInfoLocation      string
		osReleaseLocation      string
		containerSpecificFiles []string
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
		osReleaseLocation:  osReleaseLocation,
		exec:               &exec.Exec{},
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

func (i *Info) ResourceID(ctx context.Context) string {
	if i.IsContainer() {
		return i.containerID()
	}

	return i.hostID(ctx)
}

func (i *Info) ContainerInfo(ctx context.Context) *v1.Resource_ContainerInfo {
	hostname, err := i.exec.Hostname()
	if err != nil {
		slog.WarnContext(ctx, "Unable to get hostname", "error", err)
	}

	return &v1.Resource_ContainerInfo{
		ContainerInfo: &v1.ContainerInfo{
			ContainerId: i.containerID(),
			Hostname:    hostname,
			ReleaseInfo: i.releaseInfo(ctx, i.osReleaseLocation),
		},
	}
}

func (i *Info) HostInfo(ctx context.Context) *v1.Resource_HostInfo {
	hostname, err := i.exec.Hostname()
	if err != nil {
		slog.WarnContext(ctx, "Unable to get hostname", "error", err)
	}

	return &v1.Resource_HostInfo{
		HostInfo: &v1.HostInfo{
			HostId:      i.hostID(ctx),
			Hostname:    hostname,
			ReleaseInfo: i.releaseInfo(ctx, i.osReleaseLocation),
		},
	}
}

func containsContainerReference(cgroupFile string) bool {
	data, err := os.ReadFile(cgroupFile)
	if err != nil {
		slog.Warn("Unable to check if cgroup file contains a container reference", "error", err)
		return false
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, k8sKind) || strings.Contains(line, docker) || strings.Contains(line, containerd) {
			return true
		}
	}

	return false
}

func (i *Info) containerID() string {
	res, err, _ := singleflightGroup.Do(GetContainerIDKey, func() (interface{}, error) {
		containerID, err := containerIDFromMountInfo(i.mountInfoLocation)
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

// containerID returns the container ID of the current running environment.
// Supports cgroup v1 and v2. Reading "/proc/1/cpuset" would only work for cgroups v1
// mountInfo is the path: "/proc/self/mountinfo"
func containerIDFromMountInfo(mountInfo string) (string, error) {
	mInfoFile, err := os.Open(mountInfo)
	defer func(f *os.File, fileName string) {
		closeErr := f.Close()
		if closeErr != nil {
			slog.Error("Unable to close file", "file", fileName, "error", closeErr)
		}
	}(mInfoFile, mountInfo)

	if err != nil {
		return "", fmt.Errorf("could not read %s: %w", mountInfo, err)
	}

	fileScanner := bufio.NewScanner(mInfoFile)
	fileScanner.Split(bufio.ScanLines)

	lines := make([]string, 0)
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}

	for _, line := range lines {
		splitLine := strings.Split(line, " ")
		for _, word := range splitLine {
			containerID := containerIDFromPatterns(word)
			if containerID != "" {
				return containerID, nil
			}
		}
	}

	return "", fmt.Errorf("container ID not found in %s", mountInfo)
}

func containerIDFromPatterns(word string) string {
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

func (i *Info) hostID(ctx context.Context) string {
	res, err, _ := singleflightGroup.Do(GetSystemUUIDKey, func() (interface{}, error) {
		var err error

		hostID, err := i.exec.HostID(ctx)
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

func (i *Info) releaseInfo(ctx context.Context, osReleaseLocation string) (releaseInfo *v1.ReleaseInfo) {
	hostReleaseInfo := i.exec.ReleaseInfo(ctx)
	osRelease, err := readOsRelease(osReleaseLocation)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read from os release file", "error", err)

		return hostReleaseInfo
	}

	return mergeHostAndOsReleaseInfo(hostReleaseInfo, osRelease)
}

func readOsRelease(path string) (map[string]string, error) {
	f, err := os.Open(path)
	defer func(f *os.File, fileName string) {
		closeErr := f.Close()
		if closeErr != nil {
			slog.Error("Unable to close file", "file", fileName, "error", closeErr)
		}
	}(f, path)
	if err != nil {
		return nil, fmt.Errorf("release file %s is unreadable: %w", path, err)
	}

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
		if len(field) < numberOfKeysAndValues {
			continue
		}
		osReleaseInfoMap[field[0]] = strings.Trim(field[1], "\"")
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not parse os release file %w", err)
	}

	return osReleaseInfoMap, nil
}

func mergeHostAndOsReleaseInfo(
	releaseInfo *v1.ReleaseInfo,
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
