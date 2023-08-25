package phpfpm

import (
	"fmt"
	"testing"

	"github.com/nginx/agent/v2/src/core"
	"github.com/stretchr/testify/assert"
)

const phpFpmProcessRunning = `
root      654040  0.0  0.4 192784 17520 ?        Ss   Aug24   0:03 php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)
wordpre+  654042  0.0  0.2 193228  9688 ?        S    Aug24   0:00 php-fpm: pool sample_site
wordpre+  654043  0.0  0.2 193228  9748 ?        S    Aug24   0:00 php-fpm: pool sample_site
wordpre+  654044  0.0  0.2 193228  8560 ?        S    Aug24   0:00 php-fpm: pool sample_site
wordpre+  654045  0.0  0.2 193228  8560 ?        S    Aug24   0:00 php-fpm: pool sample_site
www-data  654052  0.0  0.1 193148  7412 ?        S    Aug24   0:00 php-fpm: pool www
www-data  654053  0.0  0.1 193148  7412 ?        S    Aug24   0:00 php-fpm: pool www
`

const phpFpmProcessInstalled = `
7.4
`

func TestPhpFpmStatus(t *testing.T) {
	tempShellCommander := shell
	defer func() { shell = tempShellCommander }()
	tests := []struct {
		name        string
		shell       core.Shell
		processInfo map[string]string
		expect      Status
	}{
		{
			name: "php status running",
			shell: &core.FakeShell{
				Output: map[string]string{
					"bash -c ps aux | grep php-fpm": phpFpmProcessRunning,
				},
			},
			expect: RUNNING,
		},
		{
			name: "php process installed",
			shell: &core.FakeShell{
				Output: map[string]string{
					"bash -c ps aux | grep php-fpm": ``,
					"ls /etc/php/":                  phpFpmProcessInstalled,
				},
			},
			expect: INSTALLED,
		},
		{
			name: "error retreiving php-fpm process",
			shell: &core.FakeShell{
				Errors: map[string]error{
					"bash -c ps aux | grep php-fpm": fmt.Errorf("unexpected error"),
				},
			},
			expect: MISSING,
		},
		{
			name: "no php process",
			shell: &core.FakeShell{
				Output: map[string]string{
					"bash -c ps aux | grep php-fpm": ``,
				},
				Errors: map[string]error{
					"ls /etc/php/": fmt.Errorf(" No such file or directory"),
				},
			},
			expect: MISSING,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shell = tt.shell
			actual, _ := GetPhpFpmStatus()
			assert.Equal(t, tt.expect, actual)
		})
	}
}
