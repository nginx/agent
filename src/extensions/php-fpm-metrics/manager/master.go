package manager

import (
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/master"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pkg/phpfpm"
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
func (m_mg *Master) GetMetaData(process []*phpfpm.PhpProcess) (map[int32]*master.MetaData, error) {
	master := master.New(m_mg.host)
	mdByPid, err := master.GetAll(process)
	if err != nil {
		return nil, err
	}

	for _, meta := range mdByPid {
		meta.Agent = m_mg.agent
		meta.Uuid = m_mg.uuid
		meta.Status = phpfpm.RUNNING.String()
		if len(meta.Cmd) > 0 && len(meta.ConfPath) > 0 {
			meta.LocalId = core.GenerateID("%s_%s", meta.Cmd, meta.ConfPath)
		}
	}
	return mdByPid, nil
}
