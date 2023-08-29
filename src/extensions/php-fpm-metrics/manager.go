package phpfpm

import (
	"fmt"

	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/master"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pool"
	log "github.com/sirupsen/logrus"
)

type Manager struct {
	Uuid, Host, Agent string
}

func NewManager(uuid, host, agent string) *Manager {
	return &Manager{
		Uuid:  uuid,
		Host:  host,
		Agent: agent,
	}
}

// GetMetaData returns meta-data for list of phpfpm master processes
func (m *Manager) GetMetaData() (map[string]*master.MetaData, error) {
	// search for the php fpm master process.
	master_mg := master.NewManager(m.Uuid, m.Host, m.Agent)
	md, err := master_mg.GetMetaData()
	if err != nil {
		return nil, err
	}

	for ppid, meta := range md {
		// Todo : Get process (i.e ppid) health status
		dir := fmt.Sprintf("/etc/php/%s/fpm/pool.d/", meta.Version)
		pool_mg := pool.NewManager(meta.Uuid, meta.Agent, meta.LocalId, dir, m.Host)
		meta.Pools, err = pool_mg.GetMetaData()
		if err != nil {
			log.Warnf("failed to retrieve children for master process : %s, %v", ppid, err)
		}
	}

	return md, nil
}
