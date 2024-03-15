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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/google/uuid"
	"github.com/klauspost/cpuid/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/process"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
	"golang.org/x/sys/unix"

	"github.com/nginx/agent/sdk/v2/files"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/network"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate
//counterfeiter:generate -o fake_environment_test.go . Environment
//go:generate sh -c "grep -v github.com/nginx/agent/v2/src/core fake_environment_test.go | sed -e s\\/core\\\\.\\/\\/g > fake_environment_fixed.go"
//go:generate mv fake_environment_fixed.go fake_environment_test.go
type Environment interface {
	NewHostInfo(agentVersion string, tags *[]string, configDirs string, clearCache bool) *proto.HostInfo
	// NewHostInfoWithContext(agentVersion string, tags *[]string, configDirs string, clearCache bool) *proto.HostInfo
	GetHostname() (hostname string)
	GetSystemUUID() (hostId string)
	ReadDirectory(dir string, ext string) ([]string, error)
	WriteFiles(backup ConfigApplyMarker, files []*proto.File, prefix string, allowedDirs map[string]struct{}) error
	WriteFile(backup ConfigApplyMarker, file *proto.File, confPath string) error
	DeleteFile(backup ConfigApplyMarker, fileName string) error
	Processes() (result []*Process)
	FileStat(path string) (os.FileInfo, error)
	Disks() ([]*proto.DiskPartition, error)
	DiskDevices() ([]string, error)
	DiskUsage(mountPoint string) (*DiskUsage, error)
	GetContainerID() (string, error)
	GetNetOverflow() (float64, error)
	IsContainer() bool
	Virtualization() (string, string)
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

type DiskUsage struct {
	Total          float64
	Used           float64
	Free           float64
	UsedPercentage float64
}

const (
	lengthOfContainerId = 64
	versionId           = "VERSION_ID"
	version             = "VERSION"
	codeName            = "VERSION_CODENAME"
	id                  = "ID"
	name                = "NAME"
	CTLKern             = 1  // "high kernel": proc, limits
	KernProc            = 14 // struct: process entries
	KernProcPathname    = 12 // path to executable
	SYS_SYSCTL          = 202
	IsContainerKey      = "isContainer"
	GetContainerIDKey   = "GetContainerID"
	GetSystemUUIDKey    = "GetSystemUUIDKey"
)

var (
	virtualizationFunc = host.VirtualizationWithContext
	singleflightGroup  = &singleflight.Group{}
	basePattern        = regexp.MustCompile("/([a-f0-9]{64})$")
	colonPattern       = regexp.MustCompile(":([a-f0-9]{64})$")
	scopePattern       = regexp.MustCompile(`/.+-(.+?).scope$`)
	containersPattern  = regexp.MustCompile("containers/([a-f0-9]{64})")
	containerdPattern  = regexp.MustCompile("sandboxes/([a-f0-9]{64})")
)

func (env *EnvironmentType) NewHostInfo(agentVersion string, tags *[]string, configDirs string, clearCache bool) *proto.HostInfo {
	ctx := context.Background()
	defer ctx.Done()
	return env.NewHostInfoWithContext(ctx, agentVersion, tags, configDirs, clearCache)
}

func (env *EnvironmentType) NewHostInfoWithContext(ctx context.Context, agentVersion string, tags *[]string, configDirs string, clearCache bool) *proto.HostInfo {
	defer ctx.Done()
	// temp cache measure
	if env.host == nil || clearCache {
		hostInformation, err := host.InfoWithContext(ctx)
		if err != nil {
			log.Warnf("Unable to collect dataplane host information: %v, defaulting value", err)
			return &proto.HostInfo{}
		}

		if tags == nil {
			tags = &[]string{}
		}

		disks, err := env.Disks()
		if err != nil {
			log.Warnf("Unable to get disks information from the host: %v", err)
			disks = nil
		}

		hostInfoFacacde := &proto.HostInfo{
			Agent:               agentVersion,
			Boot:                hostInformation.BootTime,
			Hostname:            hostInformation.Hostname,
			DisplayName:         hostInformation.Hostname,
			OsType:              hostInformation.OS,
			Uuid:                env.GetSystemUUID(),
			Uname:               getUnixName(),
			Partitons:           disks,
			Network:             env.networks(),
			Processor:           env.processors(hostInformation.KernelArch),
			Release:             releaseInfo("/etc/os-release"),
			Tags:                *tags,
			AgentAccessibleDirs: configDirs,
		}

		log.Tracef("HostInfo created: %v", hostInfoFacacde)
		env.host = hostInfoFacacde
	}
	return env.host
}

// getUnixName returns details about this operating system formatted as "sysname
// nodename release version machine". Returns "" if unix name cannot be
// determined.
//
//   - sysname: Name of the operating system implementation.
//   - nodename: Network name of this machine.
//   - release: Release level of the operating system.
//   - version: Version level of the operating system.
//   - machine: Machine hardware platform.
//
// Different platforms have different [Utsname] struct definitions.
//
// TODO :- Make this function platform agnostic to pull uname (uname -a).
//
// [Utsname]: https://cs.opensource.google/search?q=utsname&ss=go%2Fx%2Fsys&start=1
func getUnixName() string {
	var utsname unix.Utsname
	err := unix.Uname(&utsname)
	if err != nil {
		log.Warnf("Unable to read Uname. Error: %v", err)
		return ""
	}

	toStr := func(buf []byte) string {
		idx := bytes.IndexByte(buf, 0)
		if idx == -1 {
			return "unknown"
		}
		return string(buf[:idx])
	}

	sysName := toStr(utsname.Sysname[:])
	nodeName := toStr(utsname.Nodename[:])
	release := toStr(utsname.Release[:])
	version := toStr(utsname.Version[:])
	machine := toStr(utsname.Machine[:])
	return fmt.Sprintf("%s %s %s %s %s", sysName, nodeName, release, version, machine)
}

func (env *EnvironmentType) GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Warnf("Unable to read hostname from dataplane, defaulting value. Error: %v", err)
		hostname = ""
	}
	return hostname
}

func (env *EnvironmentType) GetSystemUUID() string {
	res, err, _ := singleflightGroup.Do(GetSystemUUIDKey, func() (interface{}, error) {
		var err error
		ctx := context.Background()
		defer ctx.Done()

		if env.IsContainer() {
			containerID, err := env.GetContainerID()
			if err != nil {
				return "", fmt.Errorf("unable to read docker container ID: %v", err)
			}
			return uuid.NewMD5(uuid.NameSpaceDNS, []byte(containerID)).String(), err
		}

		hostID, err := host.HostIDWithContext(ctx)
		if err != nil {
			log.Warnf("Unable to read host id from dataplane, defaulting value. Error: %v", err)
			return "", err
		}
		return uuid.NewMD5(uuid.Nil, []byte(hostID)).String(), err
	})
	if err != nil {
		log.Warnf("Unable to set hostname due to %v", err)
		return ""
	}

	return res.(string)
}

func (env *EnvironmentType) ReadDirectory(dir string, ext string) ([]string, error) {
	var filesList []string
	fileInfo, err := os.ReadDir(dir)
	if err != nil {
		log.Warnf("Unable to read directory %s: %v ", dir, err)
		return filesList, err
	}

	for _, file := range fileInfo {
		filesList = append(filesList, strings.Replace(file.Name(), ext, "", -1))
	}

	return filesList, nil
}

func (env *EnvironmentType) WriteFiles(backup ConfigApplyMarker, files []*proto.File, confPath string, allowedDirs map[string]struct{}) error {
	err := allowedFiles(files, allowedDirs)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err = env.WriteFile(backup, file, confPath); err != nil {
			return err
		}
	}
	return nil
}

// WriteFile writes the provided file content to disk. If the file.GetName() returns an absolute path, it'll be written
// to the path. Otherwise, it'll be written to the path relative to the provided confPath.
func (env *EnvironmentType) WriteFile(backup ConfigApplyMarker, file *proto.File, confPath string) error {
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
		log.Debugf("Creating directory %s with permissions 750", directory)
		err = os.MkdirAll(directory, 0o750)
		if err != nil {
			return err
		}
	}

	if err := os.WriteFile(fileFullPath, file.GetContents(), permissions); err != nil {
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

func (env *EnvironmentType) DeleteFile(backup ConfigApplyMarker, fileName string) error {
	if found, foundErr := FileExists(fileName); !found {
		if foundErr == nil {
			log.Debugf("skip delete for non-existing file: %s", fileName)
			return nil
		}
		// possible perm deny, depends on platform
		log.Warnf("file exists returned for %s: %s", fileName, foundErr)
		return foundErr
	}
	if err := backup.MarkAndSave(fileName); err != nil {
		return err
	}
	if err := os.Remove(fileName); err != nil {
		return err
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

	res, err, _ := singleflightGroup.Do(IsContainerKey, func() (interface{}, error) {
		for _, filename := range []string{dockerEnv, containerEnv, k8sServiceAcct} {
			if _, err := os.Stat(filename); err == nil {
				log.Debugf("Is a container because (%s) exists", filename)
				return true, nil
			}
		}
		// v1 check
		if result, err := cGroupV1Check(selfCgroup); err == nil && result {
			return result, err
		}
		return false, nil
	})

	if err != nil {
		log.Warnf("Unable to retrieve values from cache (%v)", err)
	}

	return res.(bool)
}

// cGroupV1Check returns if running cgroup v1
func cGroupV1Check(cgroupFile string) (bool, error) {
	const (
		k8sKind    = "kubepods"
		docker     = "docker"
		conatinerd = "containerd"
	)

	data, err := os.ReadFile(cgroupFile)
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
	res, err, _ := singleflightGroup.Do(GetContainerIDKey, func() (interface{}, error) {
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
	})

	return res.(string), err
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
	err = mInfoFile.Close()
	if err != nil {
		return "", fmt.Errorf("unable to close file %s: %v", mountInfo, err)
	}

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

func (env *EnvironmentType) Disks() (partitions []*proto.DiskPartition, err error) {
	ctx := context.Background()
	defer ctx.Done()
	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		// return an array of 0
		log.Errorf("Could not read disk partitions for host: %v", err)
		return []*proto.DiskPartition{}, err
	}
	for _, part := range parts {
		pm := proto.DiskPartition{
			MountPoint: part.Mountpoint,
			Device:     part.Device,
			FsType:     part.Fstype,
		}
		partitions = append(partitions, &pm)
	}
	return partitions, nil
}

func (env *EnvironmentType) DiskUsage(mountPoint string) (*DiskUsage, error) {
	ctx := context.Background()
	defer ctx.Done()
	usage, err := disk.UsageWithContext(ctx, mountPoint)
	if err != nil {
		return nil, errors.New("unable to obtain disk usage stats")
	}

	return &DiskUsage{
		Total:          float64(usage.Total),
		Used:           float64(usage.Used),
		Free:           float64(usage.Free),
		UsedPercentage: float64(usage.UsedPercent),
	}, nil
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

	dir, err := os.ReadDir(SysBlockDir)
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

func (env *EnvironmentType) FileStat(path string) (os.FileInfo, error) {
	// TODO: check if allowed list
	return os.Stat(path)
}

// Processes returns a slice of nginx master and nginx worker processes currently running
func (env *EnvironmentType) Processes() (result []*Process) {
	var processList []*Process
	ctx := context.Background()
	defer ctx.Done()

	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		log.Errorf("failed to read pids for dataplane host: %v", err)
		return processList
	}

	nginxProcesses := make(map[int32]*process.Process)
	for _, pid := range pids {

		p, _ := process.NewProcessWithContext(ctx, pid)
		name, _ := p.NameWithContext(ctx)
		cmd, _ := p.CmdlineWithContext(ctx)

		if env.isNginxProcess(name, cmd) {
			nginxProcesses[pid] = p
		}

	}

	for pid, nginxProcess := range nginxProcesses {
		name, _ := nginxProcess.NameWithContext(ctx)
		createTime, _ := nginxProcess.CreateTimeWithContext(ctx)
		status, _ := nginxProcess.StatusWithContext(ctx)
		running, _ := nginxProcess.IsRunningWithContext(ctx)
		user, _ := nginxProcess.UsernameWithContext(ctx)
		ppid, _ := nginxProcess.PpidWithContext(ctx)
		cmd, _ := nginxProcess.CmdlineWithContext(ctx)
		isMaster := false

		_, ok := nginxProcesses[ppid]
		if !ok {
			isMaster = true
		}

		var exe string
		if isMaster {
			exe = getNginxProcessExe(nginxProcess)
		} else {
			for potentialParentPid, potentialParentNginxProcess := range nginxProcesses {
				if potentialParentPid == ppid {
					exe = getNginxProcessExe(potentialParentNginxProcess)
				}
			}
		}

		newProcess := &Process{
			Pid:        pid,
			Name:       name,
			CreateTime: createTime, // Running time is probably different
			Status:     strings.Join(status, " "),
			IsRunning:  running,
			Path:       exe,
			User:       user,
			ParentPid:  ppid,
			Command:    cmd,
			IsMaster:   isMaster,
		}

		processList = append(processList, newProcess)
	}
	return processList
}

func (env *EnvironmentType) isNginxProcess(name string, cmd string) bool {
	return name == "nginx" && !strings.Contains(cmd, "upgrade") && strings.HasPrefix(cmd, "nginx:")
}

func getNginxProcessExe(nginxProcess *process.Process) string {
	exe, exeErr := nginxProcess.Exe()
	if exeErr != nil {
		out, commandErr := exec.Command("sh", "-c", "command -v nginx").CombinedOutput()
		if commandErr != nil {
			// process.Exe() is not implemented yet for FreeBSD.
			// This is a temporary workaround  until the gopsutil library supports it.
			var err error
			exe, err = getExe(nginxProcess.Pid)
			if err != nil {
				log.Tracef("Failed to find exe information for process: %d. Failed for the following errors: %v, %v, %v", nginxProcess.Pid, exeErr, commandErr, err)
				log.Errorf("Unable to find NGINX executable for process %d", nginxProcess.Pid)
			}
		} else {
			exe = strings.TrimSuffix(string(out), "\n")
		}
	}

	return exe
}

func getExe(pid int32) (string, error) {
	mib := []int32{CTLKern, KernProc, KernProcPathname, pid}
	buf, _, err := callSyscall(mib)
	if err != nil {
		return "", err
	}

	return strings.Trim(string(buf), "\x00"), nil
}

func callSyscall(mib []int32) ([]byte, uint64, error) {
	mibptr := unsafe.Pointer(&mib[0])
	miblen := uint64(len(mib))

	// get required buffer size
	length := uint64(0)
	_, _, err := unix.Syscall6(
		SYS_SYSCTL,
		uintptr(mibptr),
		uintptr(miblen),
		0,
		uintptr(unsafe.Pointer(&length)),
		0,
		0)
	if err != 0 {
		var b []byte
		return b, length, err
	}
	if length == 0 {
		var b []byte
		return b, length, err
	}
	// get proc info itself
	buf := make([]byte, length)
	_, _, err = unix.Syscall6(
		SYS_SYSCTL,
		uintptr(mibptr),
		uintptr(miblen),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&length)),
		0,
		0)
	if err != 0 {
		return buf, length, err
	}

	return buf, length, nil
}

func (env *EnvironmentType) processors(architecture string) (res []*proto.CpuInfo) {
	log.Debug("Reading CPU information for dataplane host")
	cpus, err := cpu.Info()
	if err != nil {
		log.Warnf("%v", err)
		return []*proto.CpuInfo{}
	}

	hypervisor, virtual := env.Virtualization()

	for _, item := range cpus {
		processor := proto.CpuInfo{
			// TODO: Model is a number
			// wait to see if unmarshalling error on control plane side is fixed with switch in models
			// https://stackoverflow.com/questions/21151765/cannot-unmarshal-string-into-go-value-of-type-int64
			Model: item.Model,
			Cores: item.Cores,
			// cpu_info does not provide architecture info.
			// Fix was to add KernelArch field in InfoStat struct that returns 'uname -m'
			// https://github.com/shirou/gopsutil/issues/737
			Architecture: architecture,
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
	cache := getProcessorCacheInfo(cpuid.CPU)
	if cpuid.CPU.Supports(cpuid.SSE, cpuid.SSE2) {
		cache["SIMD 2:"] = "Streaming SIMD 2 Extensions"
	}
	return cache
}

type Shell interface {
	Exec(cmd string, arg ...string) ([]byte, error)
}

type execShellCommand struct{}

func (e execShellCommand) Exec(cmd string, arg ...string) ([]byte, error) {
	execCmd := exec.Command(cmd, arg...)
	return execCmd.Output()
}

var shell Shell = execShellCommand{}

func getProcessorCacheInfo(cpuInfo cpuid.CPUInfo) map[string]string {
	cache := getDefaultProcessorCacheInfo(cpuInfo)
	return getCacheInfo(cache)
}

func getCacheInfo(cache map[string]string) map[string]string {
	out, err := shell.Exec("lscpu")
	if err != nil {
		log.Tracef("Install lscpu on host to get processor info: %v", err)
		return cache
	}
	return parseLscpu(string(out), cache)
}

func parseLscpu(lscpuInfo string, cache map[string]string) map[string]string {
	lscpuInfos := strings.TrimSpace(lscpuInfo)
	lines := strings.Split(lscpuInfos, "\n")
	lscpuInfoMap := map[string]string{}
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])
		lscpuInfoMap[key] = strings.Trim(value, "\"")
	}

	if l1dCache, ok := lscpuInfoMap["L1d cache"]; ok {
		cache["L1d"] = l1dCache
	}
	if l1iCache, ok := lscpuInfoMap["L1i cache"]; ok {
		cache["L1i"] = l1iCache
	}
	if l2Cache, ok := lscpuInfoMap["L2 cache"]; ok {
		cache["L2"] = l2Cache
	}
	if l3Cache, ok := lscpuInfoMap["L3 cache"]; ok {
		cache["L3"] = l3Cache
	}

	return cache
}

func getDefaultProcessorCacheInfo(cpuInfo cpuid.CPUInfo) map[string]string {
	// Find a library that supports multiple CPUs
	return map[string]string{
		"L1d":       formatBytes(cpuInfo.Cache.L1D),
		"L1i":       formatBytes(cpuInfo.Cache.L1D),
		"L2":        formatBytes(cpuInfo.Cache.L2),
		"L3":        formatBytes(cpuInfo.Cache.L3),
		"Features:": strings.Join(cpuInfo.FeatureSet(), ","),
		// "Flags:": strings.Join(item.Flags, ","),
		"Cacheline bytes:": fmt.Sprintf("%v", cpuInfo.CacheLine),
	}
}

func formatBytes(bytes int) string {
	if bytes <= -1 {
		return "-1"
	}
	mib := 1024 * 1024
	kib := 1024

	if bytes >= mib {
		return fmt.Sprint(bytes/mib) + " MiB"
	} else if bytes >= kib {
		return fmt.Sprint(bytes/kib) + " KiB"
	} else {
		return fmt.Sprint(bytes) + " B"
	}
}

func (env *EnvironmentType) Virtualization() (string, string) {
	ctx := context.Background()
	defer ctx.Done()
	// doesn't check k8s
	virtualizationSystem, virtualizationRole, err := virtualizationFunc(ctx)
	if err != nil {
		log.Warnf("Error reading virtualization: %v", err)
		return "", "host"
	}

	if virtualizationSystem == "docker" || env.IsContainer() {
		log.Debugf("Virtualization detected as container with role %v", virtualizationRole)
		return "container", virtualizationRole
	}
	log.Debugf("Virtualization detected as %v with role %v", virtualizationSystem, virtualizationRole)
	return virtualizationSystem, virtualizationRole
}

func isContainer() bool {
	const (
		dockerEnv      = "/.dockerenv"
		containerEnv   = "/run/.containerenv"
		selfCgroup     = "/proc/self/cgroup"
		k8sServiceAcct = "/var/run/secrets/kubernetes.io/serviceaccount"
	)

	res, err, _ := singleflightGroup.Do(IsContainerKey, func() (interface{}, error) {
		for _, filename := range []string{dockerEnv, containerEnv, k8sServiceAcct} {
			if _, err := os.Stat(filename); err == nil {
				log.Debugf("Is a container because (%s) exists", filename)
				return true, nil
			}
		}
		// v1 check
		if result, err := cGroupV1Check(selfCgroup); err == nil && result {
			return result, err
		}
		return false, nil
	})

	if err != nil {
		log.Warnf("Unable to retrieve values from cache (%v)", err)
	}

	return res.(bool)
}

func releaseInfo(osReleaseFile string) (release *proto.ReleaseInfo) {
	hostReleaseInfo := getHostReleaseInfo()
	osRelease, err := getOsRelease(osReleaseFile)
	if err != nil {
		log.Warnf("Could not read from osRelease file: %v", err)
		return hostReleaseInfo
	}
	return mergeHostAndOsReleaseInfo(hostReleaseInfo, osRelease)
}

func mergeHostAndOsReleaseInfo(hostReleaseInfo *proto.ReleaseInfo,
	osReleaseInfo map[string]string,
) (release *proto.ReleaseInfo) {
	// override os-release info with host info,
	// if os-release info is empty.
	if len(osReleaseInfo[versionId]) == 0 {
		osReleaseInfo[versionId] = hostReleaseInfo.VersionId
	}
	if len(osReleaseInfo[version]) == 0 {
		osReleaseInfo[version] = hostReleaseInfo.Version
	}
	if len(osReleaseInfo[codeName]) == 0 {
		osReleaseInfo[codeName] = hostReleaseInfo.Codename
	}
	if len(osReleaseInfo[name]) == 0 {
		osReleaseInfo[name] = hostReleaseInfo.Name
	}
	if len(osReleaseInfo[id]) == 0 {
		osReleaseInfo[id] = hostReleaseInfo.Id
	}

	return &proto.ReleaseInfo{
		VersionId: osReleaseInfo[versionId],
		Version:   osReleaseInfo[version],
		Codename:  osReleaseInfo[codeName],
		Name:      osReleaseInfo[name],
		Id:        osReleaseInfo[id],
	}
}

func getHostReleaseInfo() (release *proto.ReleaseInfo) {
	ctx := context.Background()
	defer ctx.Done()

	hostInfo, err := host.InfoWithContext(ctx)
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

// getOsRelease reads the path and returns release information for host.
func getOsRelease(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("release file %s unreadable: %w", path, err)
	}

	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()

	info, err := parseOsReleaseFile(f)
	if err != nil {
		return nil, fmt.Errorf("release file %s unparsable: %w", path, err)
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
		return nil, fmt.Errorf("could not parse os-release file %w", err)
	}

	return osReleaseInfoMap, nil
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
