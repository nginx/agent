package phpfpm

import (
	"context"
	"fmt"
	"strings"

	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/manager"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/master"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pkg/phpfpm"
	"github.com/shirou/gopsutil/process"
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
func (m *Manager) GetMetaData() (map[int32]*master.MetaData, error) {
	phpProcess, err := m.processes()
	if err != nil {
		return nil, err
	}

	for _, ps := range phpProcess {
		log.Infof("%v", ps)
	}

	master_mg := manager.NewMaster(m.Uuid, m.Host, m.Agent)
	md, err := master_mg.GetMetaData(phpProcess)
	if err != nil {
		return nil, err
	}

	for ppid, meta := range md {
		cmdSplit := strings.Split(meta.Cmd, "/")
		dir := fmt.Sprintf("/etc/php/%s/fpm/pool.d/", cmdSplit[3])
		pool_mg := manager.NewPool(meta.Uuid, meta.Agent, meta.LocalId, dir, m.Host)
		meta.Pools, err = pool_mg.GetMetaData()
		if err != nil {
			log.Errorf("failed to retrieve children for master process : %d, %v", ppid, err)
		}
	}

	return md, nil
}

// Processes returns a slice of phpfpm master and worker processes currently running
func (m *Manager) processes() ([]*phpfpm.PhpProcess, error) {
	var phpProcessList []*phpfpm.PhpProcess
	ctx := context.Background()
	defer ctx.Done()

	pids, err := process.PidsWithContext(ctx)
	if err != nil {
		return nil, err
	}

	phpProcesses := make(map[int32]*process.Process)
	for _, pid := range pids {

		p, _ := process.NewProcessWithContext(ctx, pid)
		name, _ := p.NameWithContext(ctx)

		if strings.Contains(name, "php-fpm") {
			phpProcesses[pid] = p
		}
	}

	for pid, phpProcess := range phpProcesses {
		name, _ := phpProcess.NameWithContext(ctx)
		ppid, _ := phpProcess.PpidWithContext(ctx)
		cmd, _ := phpProcess.CmdlineWithContext(ctx)
		exe, _ := phpProcess.Exe()
		isMaster := false

		_, ok := phpProcesses[ppid]
		if !ok {
			isMaster = true
		}

		newPhpProcess := &phpfpm.PhpProcess{
			Pid:       pid,
			Name:      name,
			ParentPid: ppid,
			Command:   cmd,
			IsMaster:  isMaster,
			BinPath:   exe,
		}

		phpProcessList = append(phpProcessList, newPhpProcess)
	}

	return phpProcessList, nil
}
