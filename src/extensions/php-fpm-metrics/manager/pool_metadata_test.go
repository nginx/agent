package manager_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/nginx/agent/v2/src/core"

	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/manager"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pool"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pool/worker"
	sysutils "github.com/nginx/agent/v2/test/utils/system"
	"github.com/stretchr/testify/assert"
)

func TestPhpFpmPoolMetaData(t *testing.T) {
	localDirectory := "../../../../test/testdata/configs/php/pool"
	phpConfigFiles := "sample_pool.conf   www.conf"
	tempShellCommander := pool.Shell
	agentVersion := "2.28.0"
	parentLocalId := uuid.New().String()
	uuid := uuid.New().String()
	defer func() { pool.Shell = tempShellCommander }()
	tests := []struct {
		name    string
		shell   core.Shell
		manager *manager.Pool
		expect  []*worker.MetaData
	}{
		{
			name: "get metadata",
			shell: &sysutils.FakeShell{
				Output: map[string]string{
					"ls " + localDirectory: phpConfigFiles,
				},
			},
			manager: manager.NewPool(uuid, agentVersion, parentLocalId, localDirectory, "Ubuntu"),
			expect: []*worker.MetaData{
				{
					Type:            "phpfpm_pool",
					Uuid:            uuid,
					Name:            "sample-site",
					DisplayName:     "phpfpm sample-site @ Ubuntu",
					Listen:          "/var/run/php/php7.4-fpm-$pool.sock",
					Flisten:         "/var/run/php/php7.4-fpm-sample-site.sock",
					StatusPath:      "/fpm_status",
					CanHaveChildren: false,
					Agent:           agentVersion,
					ParentLocalId:   parentLocalId,
					Includes:        []string{},
					LocalId:         core.GenerateID("%s_%s", parentLocalId, "sample-site"),
				},
				{
					Type:            "phpfpm_pool",
					Uuid:            uuid,
					Name:            "www",
					DisplayName:     "phpfpm www @ Ubuntu",
					Listen:          "/run/php/php7.4-fpm.sock",
					Flisten:         "/run/php/php7.4-fpm.sock",
					StatusPath:      "/status",
					CanHaveChildren: false,
					Agent:           agentVersion,
					ParentLocalId:   parentLocalId,
					Includes:        []string{localDirectory + "/site1", localDirectory + "/site2", localDirectory + "/sample_site3.conf"},
					LocalId:         core.GenerateID("%s_%s", parentLocalId, "www"),
				},
			},
		},
		{
			name: "no metadata found",
			shell: &sysutils.FakeShell{
				Errors: map[string]error{
					"ls " + localDirectory: errors.New("no such directory or file"),
				},
			},
			manager: manager.NewPool(uuid, agentVersion, parentLocalId, localDirectory, "Ubuntu"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool.Shell = tt.shell
			actual, _ := tt.manager.GetMetaData()
			assert.Equal(t, tt.expect, actual)
		})
	}
}
