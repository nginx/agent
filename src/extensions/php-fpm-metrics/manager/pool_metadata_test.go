package manager_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/nginx/agent/v2/src/core"

	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/manager"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pool/worker"
	"github.com/stretchr/testify/assert"
)

func TestPhpFpmPoolMetaData(t *testing.T) {
	localDirectory := "../../../../test/testdata/configs/php/pool"
	agentVersion := "2.28.0"
	parentLocalId := uuid.New().String()
	uuid := uuid.New().String()
	tests := []struct {
		name    string
		shell   core.Shell
		manager *manager.Pool
		expect  map[string]*worker.MetaData
	}{
		{
			name:    "get metadata",
			manager: manager.NewPool(uuid, agentVersion, parentLocalId, localDirectory, "Ubuntu"),
			expect: map[string]*worker.MetaData{
				core.GenerateID("%s_%s", parentLocalId, "sample-site"): {
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
				core.GenerateID("%s_%s", parentLocalId, "www"): {
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
				core.GenerateID("%s_%s", parentLocalId, "www-site3"): {
					Type:            "phpfpm_pool",
					Uuid:            uuid,
					Name:            "www-site3",
					DisplayName:     "phpfpm www-site3 @ Ubuntu",
					Listen:          "/run/php/php7.4-fpm.sock",
					Flisten:         "/run/php/php7.4-fpm.sock",
					StatusPath:      "/site3_status",
					CanHaveChildren: false,
					Agent:           agentVersion,
					ParentLocalId:   parentLocalId,
					LocalId:         core.GenerateID("%s_%s", parentLocalId, "www-site3"),
					Includes:        []string{},
				},
			},
		},
		{
			name:    "no metadata found",
			manager: manager.NewPool(uuid, agentVersion, parentLocalId, "do not exist", "Ubuntu"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, _ := tt.manager.GetMetaData()
			if actual == nil {
				assert.Nil(t, tt.expect)
			} else {
				for _, md := range actual {
					assert.Equal(t, tt.expect[md.LocalId], md)
				}
			}
		})
	}
}
