package master

import (
	"github.com/nginx/agent/v2/src/core"
)

type Manager struct {
	uuid, host, agent string
}

func NewManager(uuid, host, agent string) *Manager {
	return &Manager{
		uuid:  uuid,
		host:  host,
		agent: agent,
	}
}

// GetMetaData returns meta-data for php-fpm master processes
func (mgr *Manager) GetMetaData() (map[string]*MetaData, error) {
	master := NewMaster(mgr.host)
	mdByPid, err := master.GetAll()
	if err != nil {
		return nil, err
	}

	for _, meta := range mdByPid {
		meta.Agent = mgr.agent
		meta.Uuid = mgr.uuid
		if len(meta.Cmd) > 0 && len(meta.ConfPath) > 0 {
			meta.LocalId = core.GenerateID("%s_%s", meta.Cmd, meta.ConfPath)
		}

	}
	return mdByPid, nil
}
