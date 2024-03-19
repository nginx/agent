package utils

import (
	"os"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/stretchr/testify/mock"
)

type MockEnvironment struct {
	mock.Mock
}

// Disks implements core.Environment.
func (*MockEnvironment) Disks() ([]*proto.DiskPartition, error) {
	return []*proto.DiskPartition{
		{
			Device:     "sd01",
			MountPoint: "/sd01",
			FsType:     "ext4",
		},
		{
			Device:     "sd02",
			MountPoint: "/sd02",
			FsType:     "ext4",
		},
	}, nil
}

func (*MockEnvironment) DiskUsage(mountpoint string) (*core.DiskUsage, error) {
	return &core.DiskUsage{
		Total:          20,
		Used:           10,
		Free:           10,
		UsedPercentage: 100,
	}, nil
}

func GetProcesses() []*core.Process {
	return []*core.Process{
		{Pid: 1, Name: "12345", IsMaster: true},
		{Pid: 2, ParentPid: 1, Name: "worker-1", IsMaster: false},
		{Pid: 3, ParentPid: 1, Name: "worker-2", IsMaster: false},
	}
}

func GetMockEnv() *MockEnvironment {
	env := NewMockEnvironment()
	env.On("NewHostInfo", mock.Anything, mock.Anything, mock.Anything).Return(&proto.HostInfo{
		Hostname: "test-host",
	})
	return env
}

func GetMockEnvWithProcess() *MockEnvironment {
	env := NewMockEnvironment()
	env.On("Processes", mock.Anything).Return(GetProcesses())
	return env
}

func GetMockEnvWithHostAndProcess() *MockEnvironment {
	env := GetMockEnv()
	env.On("Processes", mock.Anything).Return(GetProcesses())
	return env
}

func NewMockEnvironment() *MockEnvironment {
	return &MockEnvironment{}
}

var _ core.Environment = NewMockEnvironment()

func (m *MockEnvironment) NewHostInfo(agentVersion string, tags *[]string, configDirs string, clearCache bool) *proto.HostInfo {
	args := m.Called(agentVersion, tags)
	returned, ok := args.Get(0).(*proto.HostInfo)
	if !ok {
		return &proto.HostInfo{
			Agent:       agentVersion,
			Boot:        0,
			Hostname:    "test-host",
			DisplayName: "",
			OsType:      "",
			Uuid:        "",
			Uname:       "",
			Partitons:   []*proto.DiskPartition{},
			Network:     &proto.Network{},
			Processor:   []*proto.CpuInfo{},
			Release:     &proto.ReleaseInfo{},
		}
	}
	return returned
}

func (m *MockEnvironment) GetHostname() string {
	return "test-host"
}

func (m *MockEnvironment) GetSystemUUID() string {
	return "12345678"
}

func (m *MockEnvironment) ReadDirectory(dir string, ext string) ([]string, error) {
	m.Called(dir, ext)
	return []string{}, nil
}

func (m *MockEnvironment) ReadFile(file string) ([]byte, error) {
	m.Called(file)
	return []byte{}, nil
}

func (m *MockEnvironment) Processes() (result []*core.Process) {
	ret := m.Called()
	return ret.Get(0).([]*core.Process)
}

func (m *MockEnvironment) WriteFiles(backup core.ConfigApplyMarker, files []*proto.File, prefix string, allowedDirs map[string]struct{}) error {
	m.Called(backup, files, prefix, allowedDirs)
	return nil
}

func (m *MockEnvironment) WriteFile(backup core.ConfigApplyMarker, file *proto.File, confPath string) error {
	m.Called(backup, file, confPath)
	return nil
}

func (m *MockEnvironment) DeleteFile(backup core.ConfigApplyMarker, fileName string) error {
	m.Called(backup, fileName)
	return nil
}

func (m *MockEnvironment) FileStat(path string) (os.FileInfo, error) {
	m.Called(path)
	return os.Stat(path)
}

func (m *MockEnvironment) DiskDevices() ([]string, error) {
	ret := m.Called()
	return ret.Get(0).([]string), ret.Error(1)
}

func (m *MockEnvironment) GetNetOverflow() (float64, error) {
	m.Called()
	return 0.0, nil
}

func (m *MockEnvironment) GetContainerID() (string, error) {
	m.Called()
	return "12345", nil
}

func (m *MockEnvironment) IsContainer() bool {
	ret := m.Called()
	return ret.Get(0).(bool)
}

func (m *MockEnvironment) Virtualization() (string, string) {
	ret := m.Called()
	return ret.Get(0).(string), ret.Get(0).(string)
}
