package manager

import (
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/master"
)

type Master struct {
	uuid, host, agent string
}

func NewMaster(uuid, host, agent string) *Master {
	return &Master{
		uuid:  uuid,
		host:  host,
		agent: agent,
	}
}

// GetMetaData returns meta-data for php-fpm master processes
func (m_mg *Master) GetMetaData() (map[string]*master.MetaData, error) {
	master := master.New(m_mg.host)
	mdByPid, err := master.GetAll()
	if err != nil {
		return nil, err
	}

	for _, meta := range mdByPid {
		meta.Agent = m_mg.agent
		meta.Uuid = m_mg.uuid
		if len(meta.Cmd) > 0 && len(meta.ConfPath) > 0 {
			meta.LocalId = core.GenerateID("%s_%s", meta.Cmd, meta.ConfPath)
		}
	}
	return mdByPid, nil
}
