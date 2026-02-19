// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package host

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/nginx/agent/v3/pkg/host/exec"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

const (
	dockerEnvLocation      = "/.dockerenv"
	containerEnvLocation   = "/run/.containerenv"
	k8sServiceAcctLocation = "/var/run/secrets/kubernetes.io/serviceaccount"

	selfCgroupLocation = "/proc/self/cgroup"
	mountInfoLocation  = "/proc/self/mountinfo"
	osReleaseLocation  = "/etc/os-release"

	ecsMetadataEnvV4 = "ECS_CONTAINER_METADATA_URI_V4"

	k8sKind    = "kubepods"
	docker     = "docker"
	containerd = "containerd"
	ecsPrefix  = "ecs"     // AWS ECS Fargate
	fargate    = "fargate" // AWS EKS Fargate

	numberOfKeysAndValues = 2
	lengthOfContainerID   = 64

	versionID = "VERSION_ID"
	version   = "VERSION"
	codeName  = "VERSION_CODENAME"
	id        = "ID"
	name      = "NAME"
)

var (
	// example: /docker/f244832c5a58377c3f1c7581b311c5bd8479808741f3e912d8bea8afe6431cb4
	basePattern = regexp.MustCompile("/([a-f0-9]{64})$")
	//nolint:lll // needs to be in one line
	// example: /system.slice/containerd.service/kubepods-besteffort-pod214f3ba8_4b69_4bdb_a7d5_5ecc73f04ae9.slice:cri-containerd:d4e8e05a546c86b6443f101966c618e47753ed01fa9929cae00d3b692f7a9f80
	colonPattern = regexp.MustCompile(":([a-f0-9]{64})$")

	// example: /system.slice/crio-9e524432d716aa750574c9b6c01dee49e4b453445006684aad94c3d6df849e5c.scope
	scopePattern = regexp.MustCompile(`/.+-(.+?).scope$`)
	//nolint:lll // needs to be in one line
	// example: /containers/storage/overlay-containers/ba0be90007be48bca767be0a462390ad2c9b0e910608158f79c8d6a984302b7e/userdata/hostname
	containersPattern = regexp.MustCompile("containers/([a-f0-9]{64})")
	//nolint:lll // needs to be in one line
	// example: /var/lib/containerd/io.containerd.grpc.v1.cri/sandboxes/d7cb24ec5dede02990283dec30bd1e6ae1f93e3e19b152b708b7e0e133c6baec/hostname
	containerdPattern = regexp.MustCompile("sandboxes/([a-f0-9]{64})")
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.11.2 -generate
//counterfeiter:generate . InfoInterface

type (
	// InfoInterface is an interface that defines methods to get information about the host or container.
	InfoInterface interface {
		IsContainer() (bool, error)
		ResourceID(ctx context.Context) (string, error)
		ContainerInfo(ctx context.Context) (*v1.Resource_ContainerInfo, error)
		HostInfo(ctx context.Context) (*v1.Resource_HostInfo, error)
	}

	Info struct {
		exec               exec.ExecInterface
		selfCgroupLocation string
		mountInfoLocation  string
		osReleaseLocation  string

		// containerSpecificFiles are files that are only created in containers.
		// We use this to determine if an instance is running in a container or not
		containerSpecificFiles []string
	}
)

// NewInfo creates and returns a new Info instance with default settings for container detection
// and operating system information retrieval.
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

// IsContainer determines if the current environment is running inside a container.
// It checks for container-specific files and container references in cgroup.
// Returns true if running in a container, false otherwise.
func (i *Info) IsContainer() (bool, error) {
	for _, filename := range i.containerSpecificFiles {
		if _, err := os.Stat(filename); err == nil {
			return true, nil
		}
	}

	ref, err := containsContainerReference(i.selfCgroupLocation)
	if ref {
		return true, nil
	}

	if os.Getenv(ecsMetadataEnvV4) != "" {
		return true, nil
	}

	if err != nil {
		return false, err
	}

	return false, nil
}

// ResourceID returns a unique identifier for the resource.
// If running in a container, it returns the container ID.
// Otherwise, it returns the host ID.
func (i *Info) ResourceID(ctx context.Context) (string, error) {
	isContainer, _ := i.IsContainer()
	if isContainer {
		return i.containerID(ctx)
	}

	return i.hostID(ctx)
}

// ContainerInfo returns container-specific information including container ID, hostname,
// and operating system release details when running in a containerized environment.
func (i *Info) ContainerInfo(ctx context.Context) (*v1.Resource_ContainerInfo, error) {
	hostname, err := i.exec.Hostname()
	if err != nil {
		return nil, err
	}
	containerId, err := i.containerID(ctx)
	if err != nil {
		return nil, err
	}
	releaseInfo, err := i.releaseInfo(ctx, i.osReleaseLocation)
	if err != nil {
		return nil, err
	}

	return &v1.Resource_ContainerInfo{
		ContainerInfo: &v1.ContainerInfo{
			ContainerId: containerId,
			Hostname:    hostname,
			ReleaseInfo: releaseInfo,
		},
	}, nil
}

// HostInfo returns information about the host system including host ID, hostname,
// and operating system release details.
func (i *Info) HostInfo(ctx context.Context) (*v1.Resource_HostInfo, error) {
	hostname, err := i.exec.Hostname()
	if err != nil {
		return nil, err
	}
	hostID, err := i.hostID(ctx)
	if err != nil {
		return nil, err
	}
	releaseInfo, err := i.releaseInfo(ctx, i.osReleaseLocation)
	if err != nil {
		return nil, err
	}

	return &v1.Resource_HostInfo{
		HostInfo: &v1.HostInfo{
			HostId:      hostID,
			Hostname:    hostname,
			ReleaseInfo: releaseInfo,
		},
	}, nil
}

// hostID returns a unique identifier for the host system.
func (i *Info) hostID(ctx context.Context) (string, error) {
	hostID, err := i.exec.HostID(ctx)
	if err != nil {
		return "", err
	}

	return uuid.NewMD5(uuid.Nil, []byte(hostID)).String(), err
}

// releaseInfo retrieves the operating system release information.
func (i *Info) releaseInfo(ctx context.Context, osReleaseLocation string) (*v1.ReleaseInfo, error) {
	hostReleaseInfo, err := i.exec.ReleaseInfo(ctx)
	if err != nil {
		return hostReleaseInfo, err
	}
	osRelease, err := readOsRelease(osReleaseLocation)
	if err != nil {
		//nolint:nilerr // If there is an error reading the OS release file just return the host release info instead
		return hostReleaseInfo, nil
	}

	return mergeHostAndOsReleaseInfo(hostReleaseInfo, osRelease), nil
}

// containerID returns the container ID of the current running environment.
func (i *Info) containerID(ctx context.Context) (string, error) {
	var errs error

	// Try to get container ID from mount info first
	if containerIDMount, err := containerIDFromMountInfo(i.mountInfoLocation); err == nil && containerIDMount != "" {
		return uuid.NewMD5(uuid.NameSpaceDNS, []byte(containerIDMount)).String(), nil
	} else if err != nil {
		errs = errors.Join(errs, err)
	}

	// Try to get container ID from ECS metadata if available
	if metadataURI := os.Getenv(ecsMetadataEnvV4); metadataURI != "" {
		if cid, err := i.containerIDFromECS(ctx, metadataURI); err == nil && cid != "" {
			return uuid.NewMD5(uuid.NameSpaceDNS, []byte(cid)).String(), nil
		} else if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return "", errs
}

// containsContainerReference checks if the cgroup file contains references to container runtimes.
func containsContainerReference(cgroupFile string) (bool, error) {
	data, err := os.ReadFile(cgroupFile)
	if err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, k8sKind) || strings.Contains(line, docker) || strings.Contains(line, containerd) ||
			strings.Contains(line, ecsPrefix) || strings.Contains(line, fargate) {
			return true, nil
		}
	}

	return false, nil
}

// containerIDFromMountInfo returns the container ID of the current running environment.
// Supports cgroup v1 and v2. Reading "/proc/1/cpuset" would only work for cgroups v1
// mountInfo is the path: "/proc/self/mountinfo"
func containerIDFromMountInfo(mountInfo string) (string, error) {
	var errs error
	mInfoFile, err := os.Open(mountInfo)
	defer func(f *os.File) {
		closeErr := f.Close()
		if closeErr != nil {
			errs = errors.Join(err, closeErr)
		}
	}(mInfoFile)

	if err != nil {
		return "", errors.Join(errs, err)
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

	return "", errors.Join(errs, fmt.Errorf("container ID not found in %s", mountInfo))
}

// containerIDFromPatterns checks a word against multiple regex patterns to extract the container ID.
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

// containsContainerID checks if the provided slices contain a valid container ID.
func containsContainerID(slices []string) bool {
	return len(slices) >= 2 && len(slices[1]) == lengthOfContainerID
}

func readOsRelease(path string) (map[string]string, error) {
	var errs error
	f, err := os.Open(path)
	defer func(f *os.File) {
		closeErr := f.Close()
		if closeErr != nil {
			errs = errors.Join(err, closeErr)
		}
	}(f)
	if err != nil {
		return nil, errors.Join(errs, fmt.Errorf("release file %s is unreadable: %w", path, err))
	}

	info, err := parseOsReleaseFile(f)
	if err != nil {
		return nil, errors.Join(errs, fmt.Errorf("release file %s is unparsable: %w", path, err))
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

func (i *Info) containerIDFromECS(ctx context.Context, uri string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata endpoint %s returned status %d", uri, resp.StatusCode)
	}

	var metadata struct {
		DockerId string `json:"DockerId"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return "", err
	}

	return metadata.DockerId, nil
}
