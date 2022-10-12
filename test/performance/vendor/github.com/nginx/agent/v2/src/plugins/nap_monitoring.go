package plugins

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/manager"
)

const (
	napMonitoringPluginName    = "Nginx App Protect Monitor"
	napMonitoringPluginVersion = "v0.0.1"
)

type NapMonitoring struct {
	securityEventsMgr *manager.SecurityEventManager
	messagePipeline   core.MessagePipeInterface
	ctx               context.Context
	ctxCancel         context.CancelFunc
}

func NewNapMonitoring(config *config.Config) (*NapMonitoring, error) {
	sem, err := manager.NewSecurityEventManager(config)
	if err != nil {
		return nil, err
	}

	return &NapMonitoring{
		securityEventsMgr: sem,
	}, nil
}

func (n *NapMonitoring) Info() *core.Info {
	return core.NewInfo(napMonitoringPluginName, napMonitoringPluginVersion)
}

func (n *NapMonitoring) Init(pipeline core.MessagePipeInterface) {
	log.Infof("%s initializing", napMonitoringPluginName)
	n.messagePipeline = pipeline
	ctx, cancel := context.WithCancel(n.messagePipeline.Context())
	n.ctx = ctx
	n.ctxCancel = cancel
	n.securityEventsMgr.Run(ctx)
}

func (n *NapMonitoring) Process(msg *core.Message) {}

func (n *NapMonitoring) Subscriptions() []string {
	return []string{}
}

func (n *NapMonitoring) Close() {
	log.Infof("%s is wrapping up", napMonitoringPluginName)
	n.ctxCancel()
}
