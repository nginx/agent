package manager

import (
	"fmt"
	"os"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pool"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pool/worker"
	log "github.com/sirupsen/logrus"
)

type Pool struct {
	uuid, agent, masterUuid, dir, host string
}

func NewPool(uuid, agent, masterUuid, dir, host string) *Pool {
	return &Pool{
		uuid:       uuid,
		agent:      agent,
		masterUuid: masterUuid,
		dir:        dir,
		host:       host,
	}
}

// GetMetaData returns meta data for phpfpm workers
func (p_mg *Pool) GetMetaData() ([]*worker.MetaData, error) {
	p := pool.New(p_mg.dir)
	files, err := p.GetConfigs(p_mg.dir)
	if err != nil {
		return nil, err
	}

	children := []*worker.MetaData{}
	for _, file := range files {
		b, err := os.ReadFile(fmt.Sprintf("%s/%s", p_mg.dir, file))
		if err != nil {
			log.Warnf("error reading file to get phpfpm pool info: %v", err)
			continue
		}
		worker := worker.New(string(b), p_mg.dir, p_mg.host)
		pool := worker.GetMetaData()
		pool.CanHaveChildren = false
		pool.Agent = p_mg.agent
		pool.Uuid = p_mg.uuid
		pool.ParentLocalId = p_mg.masterUuid
		pool.Type = "phpfpm_pool"
		pool.LocalId = core.GenerateID("%s_%s", pool.ParentLocalId, pool.Name)
		children = append(children, pool)
	}

	return children, nil
}
