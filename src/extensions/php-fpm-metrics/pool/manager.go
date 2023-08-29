package pool

import (
	"fmt"
	"os"
	"strings"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pool/worker"
	log "github.com/sirupsen/logrus"
)

// Todo: Leverage gopsutil
var Shell core.Shell = core.ExecShellCommand{}

type Manager struct {
	uuid, agent, masterUuid, dir, host string
}

func NewManager(uuid, agent, masterUuid, dir, host string) *Manager {
	return &Manager{
		uuid:       uuid,
		agent:      agent,
		masterUuid: masterUuid,
		dir:        dir,
		host:       host,
	}
}

// GetMetaData returns meta data for phpfpm workers
func (m *Manager) GetMetaData() ([]*worker.MetaData, error) {
	output, err := Shell.Exec("ls", m.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve pool conf files in dir %s: %v", m.dir, err)
	}

	files := strings.Fields(string(output))
	if len(files) == 0 {
		return nil, fmt.Errorf("no conf files in dir %s. pool configurations must be located in this dir. Err: %v", m.dir, err)
	}

	children := []*worker.MetaData{}
	for _, file := range files {
		b, err := os.ReadFile(fmt.Sprintf("%s/%s", m.dir, file))
		if err != nil {
			log.Warnf("error reading file to get phpfpm pool info: %v", err)
			continue
		}
		worker := worker.New(string(b), m.dir, m.host)
		pool := worker.GetMetaData()
		pool.CanHaveChildren = false
		pool.Agent = m.agent
		pool.Uuid = m.uuid
		pool.ParentLocalId = m.masterUuid
		pool.Type = "phpfpm_pool"
		pool.LocalId = core.GenerateID("%s_%s", pool.ParentLocalId, pool.Name)
		children = append(children, pool)
	}

	return children, nil
}
