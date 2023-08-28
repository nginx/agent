package phpfpm_test

import (
	"fmt"
	"testing"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
	pf "github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pkg/phpfpm"
)

var phpFpmProcessRunning = `
root      654040  0.0  0.4 192784 17520 ?        Ss   Aug24   0:03 php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)
wordpre+  654042  0.0  0.2 193228  9688 ?        S    Aug24   0:00 php-fpm: pool sample_site
wordpre+  654043  0.0  0.2 193228  9748 ?        S    Aug24   0:00 php-fpm: pool sample_site
wordpre+  654044  0.0  0.2 193228  8560 ?        S    Aug24   0:00 php-fpm: pool sample_site
wordpre+  654045  0.0  0.2 193228  8560 ?        S    Aug24   0:00 php-fpm: pool sample_site
www-data  654052  0.0  0.1 193148  7412 ?        S    Aug24   0:00 php-fpm: pool www
www-data  654053  0.0  0.1 193148  7412 ?        S    Aug24   0:00 php-fpm: pool www
`

var phpFpmProcessInstalled = `
cli  fpm  mods-available
`

func TestPhpFpmStatus(t *testing.T) {
	tempShellCommander := pf.Shell
	defer func() { pf.Shell = tempShellCommander }()
	tests := []struct {
		name        string
		ppid        string
		version     string
		shell       core.Shell
		processInfo map[string]string
		expect      pf.Status
	}{
			{
			name: "phpfpm process running",
			shell: &utils.FakeShell{
				Output: map[string]string{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": phpFpmProcessRunning,
				},
			},
			expect:  pf.RUNNING,
			ppid:    "654040",
			version: "7.4",
		},
		{
			name: "phpfpm process installed",
			shell: &utils.FakeShell{
				Output: map[string]string{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": ``,
					"ls /etc/php/7.4": phpFpmProcessInstalled,
				},
			},
			expect:  pf.INSTALLED,
			ppid:    "654040",
			version: "7.4",
		},
		{
			name: "error retrieving phpfpm process",
			shell: &utils.FakeShell{
				Errors: map[string]error{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": fmt.Errorf("unexpected error"),
				},
			},
			expect:  pf.UNKNOWN,
			ppid:    "654040",
			version: "7.4",
		},
		{
			name: "error retrieving phpfpm version",
			shell: &utils.FakeShell{
				Output: map[string]string{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": ``,
				},
				Errors: map[string]error{
					"ls /etc/php/7.4": fmt.Errorf(" No such file or directory"),
				},
			},
			expect:  pf.UNKNOWN,
			ppid:    "654040",
			version: "7.4",
		},
		{
			name: "missing phpfpm process",
			shell: &utils.FakeShell{
				Output: map[string]string{
					"ps xao pid,ppid,command | grep 'php-fpm[:]'": ``,
					"ls /etc/php/7.4": ``,
				},
			},
			expect:  pf.MISSING,
			ppid:    "654040",
			version: "7.4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pf.Shell = tt.shell
			actual, _ := pf.GetStatus(tt.ppid, tt.version)
			assert.Equal(t, tt.expect, actual)
		})
	}
}
