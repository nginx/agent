package manager_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/manager"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/master"
	phpfpm "github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pkg/phpfpm"

	sysutils "github.com/nginx/agent/v2/test/utils/system"
	"github.com/stretchr/testify/assert"
)

var phpFpmProcesss = []*phpfpm.PhpProcess{
	{
		Pid:      int32(856283),
		Name:     "php-fpm7.3",
		IsMaster: true,
		Command:  "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)",
		BinPath:  "/usr/sbin/php-fpm7.3",
	},
	{
		Pid:       int32(856284),
		Name:      "php-fpm7.3",
		Command:   "php-fpm: pool samplepress-site",
		BinPath:   "/usr/sbin/php-fpm7.3",
		ParentPid: int32(856283),
	},
	{
		Pid:       int32(856285),
		Name:      "php-fpm7.3",
		Command:   "php-fpm: pool samplepress-site",
		BinPath:   "/usr/sbin/php-fpm7.3",
		ParentPid: int32(856283),
	},
	{
		Pid:       int32(856286),
		Name:      "php-fpm7.3",
		Command:   "php-fpm: pool samplepress-site",
		BinPath:   "/usr/sbin/php-fpm7.3",
		ParentPid: int32(856283),
	},
	{
		Pid:      int32(654040),
		Name:     "php-fpm7.4",
		IsMaster: true,
		Command:  "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)",
		BinPath:  "/usr/sbin/php-fpm7.4",
	},
	{
		Pid:       int32(654041),
		Name:      "php-fpm7.4",
		Command:   "php-fpm: pool sample-site",
		BinPath:   "/usr/sbin/php-fpm7.4",
		ParentPid: int32(654040),
	},
	{
		Pid:       int32(654042),
		Name:      "php-fpm7.4",
		Command:   "php-fpm: pool sample-site",
		BinPath:   "/usr/sbin/php-fpm7.4",
		ParentPid: int32(654040),
	},
}

var phpFpmBin7_3_version = `PHP 7.3.33-12+ubuntu20.04.1+deb.sury.org+1 (fpm-fcgi) (built: Aug 14 2023 06:41:12)
Copyright (c) 1997-2018 The PHP Group
Zend Engine v3.3.33, Copyright (c) 1998-2018 Zend Technologies
    with Zend OPcache v7.3.33-12+ubuntu20.04.1+deb.sury.org+1, Copyright (c) 1999-2018, by Zend Technologies
`

var phpFpmBin7_4_version = `PHP 7.4.33 (fpm-fcgi) (built: Feb 14 2023 18:31:23)
Copyright (c) The PHP Group
Zend Engine v3.4.0, Copyright (c) Zend Technologies
    with Zend OPcache v7.4.33, Copyright (c), by Zend Technologies
`

var phpFpmProcesssRunning = []*phpfpm.PhpProcess{
	{
		Pid:      int32(654040),
		Name:     "php-fpm7.4",
		IsMaster: true,
		Command:  "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)",
		BinPath:  "/usr/sbin/php-fpm7.4",
	},
}

func TestPhpFpmMasterMetaData(t *testing.T) {
	uuid := uuid.New().String()
	host := "Ubuntu"
	agent := "2.28.0"
	tests := []struct {
		name    string
		shell   core.Shell
		expect  map[int32]*master.MetaData
		input   []*phpfpm.PhpProcess
		manager *manager.Master
	}{
		{
			name: "get meta data with master and worker processes running",
			shell: &sysutils.FakeShell{
				Output: map[string]string{
					"/usr/sbin/php-fpm7.3 --version": phpFpmBin7_3_version,
					"/usr/sbin/php-fpm7.4 --version": phpFpmBin7_4_version,
				},
			},
			input:   phpFpmProcesss,
			manager: manager.NewMaster(uuid, host, agent),
			expect: map[int32]*master.MetaData{
				int32(654040): {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.4/fpm/php-fpm.conf",
					NumWorkers:  2,
					Version:     "7.4.33",
					VersionLine: "PHP 7.4.33 (fpm-fcgi) (built: Feb 14 2023 18:31:23)",
					Pid:         654040,
					BinPath:     "/usr/sbin/php-fpm7.4",
					Agent:       "2.28.0",
					Status:      "RUNNING",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)", "/etc/php/7.4/fpm/php-fpm.conf"),
				},
				int32(856283): {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.3/fpm/php-fpm.conf",
					NumWorkers:  3,
					Version:     "7.3.33-12",
					VersionLine: "PHP 7.3.33-12+ubuntu20.04.1+deb.sury.org+1 (fpm-fcgi) (built: Aug 14 2023 06:41:12)",
					Pid:         856283,
					BinPath:     "/usr/sbin/php-fpm7.3",
					Agent:       "2.28.0",
					Status:      "RUNNING",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)", "/etc/php/7.3/fpm/php-fpm.conf"),
				},
			},
		},
		{
			name:    "error in finding version",
			input:   phpFpmProcesss,
			manager: manager.NewMaster(uuid, host, agent),
			shell: &sysutils.FakeShell{
				Errors: map[string]error{
					"sudo /usr/sbin/php-fpm7.3 --version": errors.New("proc not supported"),
					"sudo /usr/sbin/php-fpm7.4 --version": errors.New("proc not supported"),
				},
			},
			expect: map[int32]*master.MetaData{
				int32(654040): {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.4/fpm/php-fpm.conf",
					NumWorkers:  2,
					Pid:         654040,
					Version:     "7.4",
					Agent:       "2.28.0",
					Status:      "RUNNING",
					BinPath:     "/usr/sbin/php-fpm7.4",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)", "/etc/php/7.4/fpm/php-fpm.conf"),
				},
				int32(856283): {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.3/fpm/php-fpm.conf",
					Version:     "7.3",
					NumWorkers:  3,
					Pid:         856283,
					Agent:       "2.28.0",
					Status:      "RUNNING",
					BinPath:     "/usr/sbin/php-fpm7.3",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)", "/etc/php/7.3/fpm/php-fpm.conf"),
				},
			},
		},
		{
			name:    "get meta data for phpfpm process with only master running",
			manager: manager.NewMaster(uuid, host, agent),
			input:   phpFpmProcesssRunning,
			shell: &sysutils.FakeShell{
				Output: map[string]string{
					"/usr/sbin/php-fpm7.4 --version": phpFpmBin7_4_version,
				},
			},
			expect: map[int32]*master.MetaData{
				int32(654040): {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.4/fpm/php-fpm.conf",
					NumWorkers:  0,
					Version:     "7.4.33",
					VersionLine: "PHP 7.4.33 (fpm-fcgi) (built: Feb 14 2023 18:31:23)",
					Pid:         654040,
					BinPath:     "/usr/sbin/php-fpm7.4",
					Agent:       "2.28.0",
					Status:      "RUNNING",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)", "/etc/php/7.4/fpm/php-fpm.conf"),
				},
			},
		},
		{
			name:    "no master process running",
			manager: manager.NewMaster(uuid, host, agent),
			expect:  map[int32]*master.MetaData{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			master.Shell = tt.shell
			actual, _ := tt.manager.GetMetaData(tt.input)
			assert.Equal(t, tt.expect, actual)
		})
	}
}
