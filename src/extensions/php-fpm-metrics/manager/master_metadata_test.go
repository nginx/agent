package manager_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/manager"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/master"
	sysutils "github.com/nginx/agent/v2/test/utils/system"
	"github.com/stretchr/testify/assert"
)

var phpFpmProcess = `
654040       1 php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)
654042  654040 php-fpm: pool sample_site
654043  654040 php-fpm: pool sample_site
654044  654040 php-fpm: pool sample_site
654045  654040 php-fpm: pool sample_site
654046  654040 php-fpm: pool sample_site
654047  654040 php-fpm: pool sample_site
654048  654040 php-fpm: pool sample_site
654049  654040 php-fpm: pool sample_site
654050  654040 php-fpm: pool sample_site
654051  654040 php-fpm: pool sample_site
654052  654040 php-fpm: pool www
654053  654040 php-fpm: pool www
856283       1 php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)
856284  856283 php-fpm: pool samplepress-site
856285  856283 php-fpm: pool samplepress-site
856286  856283 php-fpm: pool samplepress-site
856287  856283 php-fpm: pool samplepress-site
856288  856283 php-fpm: pool samplepress-site
856289  856283 php-fpm: pool samplepress-site
856290  856283 php-fpm: pool samplepress-site
856292  856283 php-fpm: pool samplepress-site
856293  856283 php-fpm: pool samplepress-site
856294  856283 php-fpm: pool samplepress-site
`

var phpFpmProcessWithMaster = `
654040       1 php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)
`

var phpFpmBin7_4 = `
lrwxrwxrwx 1 root root 0 Aug 24 21:09 /proc/654040/exe -> /usr/sbin/php-fpm7.4
`

var phpFpmBin7_3 = `
lrwxrwxrwx 1 root root 0 Aug 27 05:39 /proc/856283/exe -> /usr/sbin/php-fpm7.3
`

var phpFpmBin7_4_version = `
PHP 7.4.33 (fpm-fcgi) (built: Feb 14 2023 18:31:23)
Copyright (c) The PHP Group
Zend Engine v3.4.0, Copyright (c) Zend Technologies
    with Zend OPcache v7.4.33, Copyright (c), by Zend Technologies
`

var phpFpmBin7_3_version = `
PHP 7.3.33-12+ubuntu20.04.1+deb.sury.org+1 (fpm-fcgi) (built: Aug 14 2023 06:41:12)
Copyright (c) 1997-2018 The PHP Group
Zend Engine v3.3.33, Copyright (c) 1998-2018 Zend Technologies
    with Zend OPcache v7.3.33-12+ubuntu20.04.1+deb.sury.org+1, Copyright (c) 1999-2018, by Zend Technologies
`

func TestPhpFpmMasterMetaData(t *testing.T) {
	tempShellCommander := master.Shell
	defer func() { master.Shell = tempShellCommander }()
	uuid := uuid.New().String()
	host := "Ubuntu"
	agent := "2.28.0"
	tests := []struct {
		name    string
		shell   core.Shell
		expect  map[string]*master.MetaData
		manager *manager.Master
	}{
		{
			name: "get meta data with master and worker processes running",
			shell: &sysutils.FakeShell{
				Output: map[string]string{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": phpFpmProcess,
					"sudo ls -la /proc/856283/exe":                phpFpmBin7_3,
					"sudo /usr/sbin/php-fpm7.3 --version":         phpFpmBin7_3_version,
					"sudo ls -la /proc/654040/exe":                phpFpmBin7_4,
					"sudo /usr/sbin/php-fpm7.4 --version":         phpFpmBin7_4_version,
				},
			},
			manager: manager.NewMaster(uuid, host, agent),
			expect: map[string]*master.MetaData{
				"654040": {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.4/fpm/php-fpm.conf",
					NumWorkers:  12,
					Version:     "7.4.33",
					VersionLine: "PHP 7.4.33 (fpm-fcgi) (built: Feb 14 2023 18:31:23)",
					Pid:         654040,
					BinPath:     "/usr/sbin/php-fpm7.4",
					Agent:       "2.28.0",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)", "/etc/php/7.4/fpm/php-fpm.conf"),
				},
				"856283": {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.3/fpm/php-fpm.conf",
					NumWorkers:  10,
					Version:     "7.3.33-12",
					VersionLine: "PHP 7.3.33-12+ubuntu20.04.1+deb.sury.org+1 (fpm-fcgi) (built: Aug 14 2023 06:41:12)",
					Pid:         856283,
					BinPath:     "/usr/sbin/php-fpm7.3",
					Agent:       "2.28.0",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)", "/etc/php/7.3/fpm/php-fpm.conf"),
				},
			},
		},
		{
			name: "proc error",
			shell: &sysutils.FakeShell{
				Output: map[string]string{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": phpFpmProcess,
					"sudo /usr/sbin/php-fpm7.3 --version":         phpFpmBin7_3_version,
					"sudo /usr/sbin/php-fpm7.4 --version":         phpFpmBin7_4_version,
				},
				Errors: map[string]error{
					"sudo ls -la /proc/654040/exe": errors.New("proc not supported"),
					"sudo ls -la /proc/856283/exe": errors.New("proc not supported"),
				},
			},
			manager: manager.NewMaster(uuid, host, agent),
			expect: map[string]*master.MetaData{
				"654040": {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.4/fpm/php-fpm.conf",
					NumWorkers:  12,
					Pid:         654040,
					Agent:       "2.28.0",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)", "/etc/php/7.4/fpm/php-fpm.conf"),
				},
				"856283": {
					Type:        "phpfpm",
					Uuid:        uuid,
					Cmd:         "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)",
					Name:        "master",
					DisplayName: "phpfpm master @ Ubuntu",
					ConfPath:    "/etc/php/7.3/fpm/php-fpm.conf",
					NumWorkers:  10,
					Pid:         856283,
					Agent:       "2.28.0",
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.3/fpm/php-fpm.conf)", "/etc/php/7.3/fpm/php-fpm.conf"),
				},
			},
		},
		{
			name: "get meta data for phpfpm process with only master running",
			shell: &sysutils.FakeShell{
				Output: map[string]string{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": phpFpmProcessWithMaster,
					"sudo ls -la /proc/654040/exe":                phpFpmBin7_4,
					"sudo /usr/sbin/php-fpm7.4 --version":         phpFpmBin7_4_version,
				},
			},
			manager: manager.NewMaster(uuid, host, agent),
			expect: map[string]*master.MetaData{
				"654040": {
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
					LocalId:     core.GenerateID("%s_%s", "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)", "/etc/php/7.4/fpm/php-fpm.conf"),
				},
			},
		},
		{
			name: "no master process running",
			shell: &sysutils.FakeShell{
				Output: map[string]string{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": "",
				},
			},
			manager: manager.NewMaster(uuid, host, agent),
			expect:  map[string]*master.MetaData{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			master.Shell = tt.shell
			actual, _ := tt.manager.GetMetaData()
			assert.Equal(t, tt.expect, actual)
		})
	}
}
