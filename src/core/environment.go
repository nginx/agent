/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/klauspost/cpuid/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/process"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2/files"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/network"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fake_environment_test.go . Environment
//go:generate sh -c "grep -v agent/product/nginx-agent/v2/core fake_environment_test.go | sed -e s\\/core\\\\.\\/\\/g > fake_environment_fixed.go"
//go:generate mv fake_environment_fixed.go fake_environment_test.go
type Environment interface {
	NewHostInfo(agentVersion string, tags *[]string, configDirs string, clearCache bool) *proto.HostInfo
	GetHostname() (hostname string)
	GetSystemUUID() (hostId string)
	ReadDirectory(dir string, ext string) ([]string, error)
	WriteFiles(backup ConfigApplyMarker, files []*proto.File, prefix string, allowedDirs map[string]struct{}) error
	Processes() (result []Process)
	FileStat(path string) (os.FileInfo, error)
	DiskDevices() ([]string, error)
	GetContainerID() (string, error)
	GetNetOverflow() (float64, error)
	IsContainer() bool
}

type ConfigApplyMarker interface {
	MarkAndSave(string) error
	RemoveFromNotExists(string)
}

type EnvironmentType struct {
	host *proto.HostInfo
}

type Process struct {
	Pid        int32
	Name       string
	CreateTime int64
	Status     string
	IsRunning  bool
	IsMaster   bool
	Path       string
	User       string
	ParentPid  int32
	Command    string
}

const lengthOfContainerId = 64

var (
	virtualizationFunc             = host.Virtualization
	_                  Environment = &EnvironmentType{}
)

func (env *EnvironmentType) NewHostInfo(agentVersion string, tags *[]string, configDirs string, clearCache bool) *proto.HostInfo {
	// temp cache measure
	if env.host == nil || clearCache {
		hostInformation, err := host.Info()
		if err != nil {
			log.Warnf("Unable to collect dataplane host information: %v, defaulting value", err)
			return &proto.HostInfo{}
		}

		hostInfoFacacde := &proto.HostInfo{
			Agent:               agentVersion,
			Boot:                hostInformation.BootTime,
			Hostname:            hostInformation.Hostname,
			DisplayName:         hostInformation.Hostname,
			OsType:              hostInformation.OS,
			Uuid:                env.GetSystemUUID(),
			Uname:               hostInformation.KernelArch,
			Partitons:           diskPartitions(),
			Network:             env.networks(),
			Processor:           processors(),
			Release:             releaseInfo(),
			Tags:                *tags,
			AgentAccessibleDirs: configDirs,
		}

		log.Tracef("HostInfo created: %v", hostInfoFacacde)
		env.host = hostInfoFacacde
	}
	return env.host
}

func (env *EnvironmentType) GetHostname() string {
	hostInformation, err := host.Info()
	if err != nil {
		log.Warnf("Unable to read hostname from dataplane, defaulting value. Error: %v", err)
		return ""
	}
	return hostInformation.Hostname
}

func (env *EnvironmentType) GetSystemUUID() string {
	if env.IsContainer() {
		containerID, err := env.GetContainerID()
		if err != nil {
			log.Errorf("Unable to read docker container ID: %v", err)
			return ""
		}
		return uuid.NewMD5(uuid.NameSpaceDNS, []byte(containerID)).String()
	}

	hostInfo, err := host.Info()
	if err != nil {
		log.Infof("Unable to read host id from dataplane, defaulting value. Error: %v", err)
		return ""
	}
	return uuid.NewMD5(uuid.Nil, []byte(hostInfo.HostID)).String()
}

func (env *EnvironmentType) ReadDirectory(dir string, ext string) ([]string, error) {
	var files []string
	fileInfo, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Warnf("Unable to reading directory %s: %v ", dir, err)
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, strings.Replace(file.Name(), ext, "", -1))
	}

	return files, nil
}

func (env *EnvironmentType) WriteFiles(backup ConfigApplyMarker, files []*proto.File, confPath string, allowedDirs map[string]struct{}) error {
	err := allowedFiles(files, allowedDirs)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err = writeFile(backup, file, confPath); err != nil {
			return err
		}
	}
	return nil
}

func (env *EnvironmentType) IsContainer() bool {
	const (
		dockerEnv      = "/.dockerenv"
		containerEnv   = "/run/.containerenv"
		selfCgroup     = "/proc/self/cgroup"
		k8sServiceAcct = "/var/run/secrets/kubernetes.io/serviceaccount"
	)

	for _, filename := range []string{dockerEnv, containerEnv, k8sServiceAcct} {
		if _, err := os.Stat(filename); err == nil {
			log.Debugf("is a container because (%s) exists", filename)
			return true
		}
	}
	// v1 check
	if result, err := cGroupV1Check(selfCgroup); err == nil && result {
		return result
	}

	return false
}

// cGroupV1Check returns if running cgroup v1
func cGroupV1Check(cgroupFile string) (bool, error) {
	const (
		k8sKind    = "kubepods"
		docker     = "docker"
		conatinerd = "containerd"
	)

	data, err := ioutil.ReadFile(cgroupFile)
	if err != nil {
		return false, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, k8sKind) || strings.Contains(line, docker) || strings.Contains(line, conatinerd) {
			return true, nil
		}
	}
	return false, errors.New("cGroup v1 information not found")
}

// GetContainerID returns the ID of the current environment if running in a container
func (env *EnvironmentType) GetContainerID() (string, error) {
	const mountInfo = "/proc/self/mountinfo"

	if !env.IsContainer() {
		return "", errors.New("not in docker")
	}

	containerID, err := getContainerID(mountInfo)
	if err != nil {
		return "", fmt.Errorf("could not get container ID: %v", err)
	}

	log.Debugf("Container ID: %s", containerID)

	return containerID, err
}

// getContainerID returns the container ID of the current running environment.
// Supports cgroup v1 and v2. Reading "/proc/1/cpuset" would only work for cgroups v1
// mountInfo is the path: "/proc/self/mountinfo"
func getContainerID(mountInfo string) (string, error) {
	mInfoFile, err := os.Open(mountInfo)
	if err != nil {
		return "", fmt.Errorf("could not read %s: %v", mountInfo, err)
	}

	fileScanner := bufio.NewScanner(mInfoFile)
	fileScanner.Split(bufio.ScanLines)

	var lines []string
	for fileScanner.Scan() {
		lines = append(lines, fileScanner.Text())
	}
	mInfoFile.Close()

	basePattern := regexp.MustCompile("/([a-f0-9]{64})$")
	colonPattern := regexp.MustCompile(":([a-f0-9]{64})$")
	scopePattern := regexp.MustCompile(`/.+-(.+?).scope$`)
	containersPattern := regexp.MustCompile("containers/([a-f0-9]{64})")
	containerdPattern := regexp.MustCompile("sandboxes/([a-f0-9]{64})")

	for _, line := range lines {
		splitLine := strings.Split(line, " ")
		for _, word := range splitLine {
			slices := scopePattern.FindStringSubmatch(word)
			if len(slices) >= 2 && len(slices[1]) == lengthOfContainerId {
				return slices[1], nil
			}

			slices = basePattern.FindStringSubmatch(word)
			if len(slices) >= 2 && len(slices[1]) == lengthOfContainerId {
				return slices[1], nil
			}

			slices = colonPattern.FindStringSubmatch(word)
			if len(slices) >= 2 && len(slices[1]) == lengthOfContainerId {
				return slices[1], nil
			}

			slices = containersPattern.FindStringSubmatch(word)
			if len(slices) >= 2 && len(slices[1]) == lengthOfContainerId {
				return slices[1], nil
			}

			slices = containerdPattern.FindStringSubmatch(word)
			if len(slices) >= 2 && len(slices[1]) == lengthOfContainerId {
				return slices[1], nil
			}
		}
	}

	return "", errors.New("no container ID found")
}

// DiskDevices returns a list of Disk Devices known by the system.
// Loop and other virtual devices are filtered out
func (env *EnvironmentType) DiskDevices() ([]string, error) {
	switch runtime.GOOS {
	case "freebsd":
		return env.getFreeBSDDiskDevices()
	case "darwin":
		return []string{}, errors.New("darwin architecture is not supported")
	default:
		return getLinuxDiskDevices()
	}
}

func (env *EnvironmentType) GetNetOverflow() (float64, error) {
	return network.GetNetOverflow()
}

func getLinuxDiskDevices() ([]string, error) {
	const (
		SysBlockDir       = "/sys/block"
		LoopDeviceMark    = "/loop"
		VirtualDeviceMark = "/virtual"
	)
	dd := []string{}
	log.Debugf("Reading directory for linux disk information: %s", SysBlockDir)

	dir, err := ioutil.ReadDir(SysBlockDir)
	if err != nil {
		return dd, err
	}

	for _, f := range dir {
		rl, e := os.Readlink(filepath.Join(SysBlockDir, f.Name()))
		if e != nil {
			continue
		}
		if strings.Contains(rl, LoopDeviceMark) || strings.Contains(rl, VirtualDeviceMark) {
			continue
		}
		dd = append(dd, f.Name())
	}

	return dd, nil
}

func (env *EnvironmentType) getFreeBSDDiskDevices() ([]string, error) {
	devices := []string{}

	geomBin, secErr := env.checkUtil("geom")
	if secErr != nil {
		return devices, secErr
	}

	outbuf, err := runCmd(geomBin, "disk", "list")
	if err != nil {
		return devices, errors.New("unable to obtain disk list")
	}

	for _, line := range strings.Split(outbuf.String(), "\n") {
		if !strings.HasPrefix(line, "Geom name:") {
			continue
		}
		geomFields := strings.Fields(line)
		devices = append(devices, geomFields[len(geomFields)-1])
	}

	return devices, nil
}

func allowedFiles(files []*proto.File, allowedDirs map[string]struct{}) error {
	for _, file := range files {
		path := file.GetName()
		if !allowedFile(path, allowedDirs) {
			return fmt.Errorf("write prohibited for: %s", path)
		}
	}
	return nil
}

func allowedFile(path string, allowedDirs map[string]struct{}) bool {
	if !filepath.IsAbs(path) {
		// if not absolute path, we'll put it at the relative to config dir for the binary
		return true
	}
	for dir := range allowedDirs {
		if strings.HasPrefix(path, dir) {
			return true
		}
	}
	return false
}

// writeFile writes the provided file content to disk. If the file.GetName() returns an absolute path, it'll be written
// to the path. Otherwise, it'll be written to the path relative to the provided confPath.
func writeFile(backup ConfigApplyMarker, file *proto.File, confPath string) error {
	fileFullPath := file.GetName()
	if !filepath.IsAbs(fileFullPath) {
		fileFullPath = filepath.Join(confPath, fileFullPath)
	}

	if err := backup.MarkAndSave(fileFullPath); err != nil {
		return err
	}
	permissions := files.GetFileMode(file.GetPermissions())

	directory := filepath.Dir(fileFullPath)
	_, err := os.Stat(directory)
	if os.IsNotExist(err) {
		log.Debugf("Creating directory %s with permissions 755", directory)
		err = os.MkdirAll(directory, 0755)
		if err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(fileFullPath, file.GetContents(), permissions); err != nil {
		// If the file didn't exist originally and failed to be created
		// Then remove that file from the backup so that the rollback doesn't try to delete the file
		if _, err := os.Stat(fileFullPath); !errors.Is(err, os.ErrNotExist) {
			backup.RemoveFromNotExists(fileFullPath)
		}
		return err
	}

	log.Debugf("Wrote file %s", fileFullPath)
	return nil
}

func (env *EnvironmentType) FileStat(path string) (os.FileInfo, error) {
	// TODO: check if allowed list
	return os.Stat(path)
}

// Processes returns a slice of nginx master and nginx worker processes currently running
func (env *EnvironmentType) Processes() (result []Process) {
	var processList []Process

	pids, err := process.Pids()
	if err != nil {
		log.Errorf("failed to read pids for dataplane host: %v", err)
		return processList
	}

	seenPids := make(map[int32]bool)
	for _, pid := range pids {
		p, _ := process.NewProcess(pid)
		name, _ := p.Name()

		if name == "nginx" {
			createTime, _ := p.CreateTime()
			status, _ := p.Status()
			running, _ := p.IsRunning()
			user, _ := p.Username()
			ppid, _ := p.Ppid()
			cmd, _ := p.Cmdline()
			exe, _ := p.Exe()

			// if the exe is empty, try get the exe from the parent
			if exe == "" {
				parentProcess, _ := process.NewProcess(ppid)
				exe, _ = parentProcess.Exe()
			}

			processList = append(processList, Process{
				Pid:        pid,
				Name:       name,
				CreateTime: createTime, // Running time is probably different
				Status:     strings.Join(status, " "),
				IsRunning:  running,
				Path:       exe,
				User:       user,
				ParentPid:  ppid,
				Command:    cmd,
			})
			seenPids[pid] = true
		}
	}

	for i := 0; i < len(processList); i++ {
		item := &processList[i]
		if seenPids[item.ParentPid] {
			item.IsMaster = false
		} else {
			item.IsMaster = true
		}
	}

	return processList
}

func processors() (res []*proto.CpuInfo) {
	log.Debug("Reading CPU information for dataplane host")
	cpus, err := cpu.Info()
	if err != nil {
		log.Warnf("%v", err)
		return []*proto.CpuInfo{}
	}

	hypervisor, virtual := virtualization()

	for _, item := range cpus {
		processor := proto.CpuInfo{
			// TODO: Model is a number
			// wait to see if unmarshalling error on control plane side is fixed with switch in models
			// https://stackoverflow.com/questions/21151765/cannot-unmarshal-string-into-go-value-of-type-int64
			Model:        item.Model,
			Cores:        item.Cores,
			Architecture: item.Family,
			Cpus:         int32(len(cpus)),
			Mhz:          item.Mhz,
			// TODO - check if this is correct
			Hypervisor:     hypervisor,
			Virtualization: virtual,
			Cache:          processorCache(item),
		}

		res = append(res, &processor)
	}

	return res
}

func processorCache(item cpu.InfoStat) map[string]string {
	// Find a library that supports multiple CPUs
	cache := map[string]string{
		// values are in bytes
		"L1d":       fmt.Sprintf("%v", cpuid.CPU.Cache.L1D),
		"L1i":       fmt.Sprintf("%v", cpuid.CPU.Cache.L1D),
		"L2":        fmt.Sprintf("%v", cpuid.CPU.Cache.L2),
		"L3":        fmt.Sprintf("%v", cpuid.CPU.Cache.L3),
		"Features:": strings.Join(cpuid.CPU.FeatureSet(), ","),
		// "Flags:": strings.Join(item.Flags, ","),
		"Cacheline bytes:": fmt.Sprintf("%v", cpuid.CPU.CacheLine),
	}

	if cpuid.CPU.Supports(cpuid.SSE, cpuid.SSE2) {
		cache["SIMD 2:"] = "Streaming SIMD 2 Extensions"
	}
	return cache
}

func virtualization() (string, string) {
	// doesn't check k8s
	virtualizationSystem, virtualizationRole, err := virtualizationFunc()
	if err != nil {
		log.Warnf("Error reading virtualization: %v", err)
		return "", "host"
	}

	if virtualizationSystem == "docker" {
		return "container", virtualizationRole
	}
	return virtualizationSystem, virtualizationRole
}

func diskPartitions() (partitions []*proto.DiskPartition) {
	parts, err := disk.Partitions(false)
	if err != nil {
		// return an array of 0
		log.Errorf("Could not read disk partitions for host: %v", err)
		return []*proto.DiskPartition{}
	}
	for _, part := range parts {
		pm := proto.DiskPartition{
			MountPoint: part.Mountpoint,
			Device:     part.Device,
			FsType:     part.Fstype,
		}
		partitions = append(partitions, &pm)
	}
	return partitions
}

func releaseInfo() (release *proto.ReleaseInfo) {
	const osReleaseFile =  "/etc/os-release"
	osRelease, err := getOsRelease(osReleaseFile)
	if err != nil {
		hostInfo, err := host.Info()
		if err != nil {
			log.Errorf("Could not read release information for host: %v", err)
			return &proto.ReleaseInfo{}
		}

		return &proto.ReleaseInfo{
			VersionId: hostInfo.PlatformVersion,
			Version:   hostInfo.KernelVersion,
			Codename:  hostInfo.OS,
			Name:      hostInfo.PlatformFamily,
			Id:        hostInfo.Platform,
		}
	}
	return osRelease
}

// getOsRelease reads osReleaseFile and returns release information for host.
// If os.Stat(osReleaseFilePath) does not find file, or
// ioutil.ReadFile(osReleaseFilePath) fails to read file, an error occurs.
func getOsRelease(osReleaseFile string) (release *proto.ReleaseInfo, err error) {
	_ , osReleaseError := os.Stat(osReleaseFile)
	if os.IsNotExist(osReleaseError) {
		log.Errorf("Could not find path for os-release file on the host")
		return &proto.ReleaseInfo{}, errors.New("unable to find " + osReleaseFile)
	}

	osReleaseInfoMap := map[string]string{}

	data, err := ioutil.ReadFile(osReleaseFile)
	if err != nil {
		log.Errorf("Could not read os-release file on the host")
		return &proto.ReleaseInfo{}, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		field := strings.Split(line, "=")
		if len(field) < 2 {
			continue
		}
		osReleaseInfoMap[field[0]] = strings.Trim(field[1], "\"")
	}

	if _, ok := osReleaseInfoMap["NAME"]; !ok {
		osReleaseInfoMap["NAME"] = "unix"
	}

	return &proto.ReleaseInfo{
		VersionId: osReleaseInfoMap["VERSION_ID"],
		Version:   osReleaseInfoMap["VERSION"],
		Codename:  osReleaseInfoMap["VERSION_CODENAME"],
		Name:      osReleaseInfoMap["NAME"],
		Id:        osReleaseInfoMap["ID"],
	}, nil
}

func (env *EnvironmentType) networks() (res *proto.Network) {
	return network.GetDataplaneNetworks()
}

func (env *EnvironmentType) checkUtil(util string) (string, error) {
	log.Infof("Trying to exec the following utility: %s", util)
	path, err := exec.LookPath(util)
	if err != nil {
		return "", err
	}

	info, err := env.FileStat(path)
	if err != nil {
		return "", err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("unable to determine binary ownership: %s", path)
	} else if stat.Uid != 0 {
		return "", fmt.Errorf("binary is not root owned: %s", path)
	}

	if info.Mode()&(os.ModeSetgid|os.ModeSetuid) != 0 {
		return "", fmt.Errorf("SetUID or SetGID bits set: %s", path)
	}

	return path, nil
}

func runCmd(cmd string, args ...string) (*bytes.Buffer, error) {
	log.Infof("Attempting to run command: %s with args %v", cmd, strings.Join(args, " "))

	command := exec.Command(cmd, args...)

	output, err := command.CombinedOutput()
	if err != nil {
		log.Warnf("%v %v failed:\n%s", cmd, strings.Join(args, " "), output)
		return bytes.NewBuffer(output), err
	}

	return bytes.NewBuffer(output), nil
}
