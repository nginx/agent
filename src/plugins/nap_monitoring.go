package plugins

import (
	"context"
	events "github.com/nginx/agent/sdk/v2/proto/events"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/manager"
)

const (
	napMonitoringPluginName    = "Nginx App Protect Monitor"
	napMonitoringPluginVersion = "v0.0.1"
	reportInterval             = 5 * time.Second
	reportCount                = 50
)

type NAPMonitoring struct {
	monitorMgr      *manager.Manager
	messagePipeline core.MessagePipeInterface
	ctx             context.Context
	ctxCancel       context.CancelFunc
}

func NewNAPMonitoring(config *config.Config) (*NAPMonitoring, error) {
	m, err := manager.NewManager(config)
	if err != nil {
		return nil, err
	}

	return &NAPMonitoring{
		monitorMgr: m,
	}, nil
}

func (n *NAPMonitoring) Info() *core.Info {
	return core.NewInfo(napMonitoringPluginName, napMonitoringPluginVersion)
}

func (n *NAPMonitoring) Init(pipeline core.MessagePipeInterface) {
	log.Infof("%s initializing", napMonitoringPluginName)
	n.messagePipeline = pipeline
	ctx, cancel := context.WithCancel(n.messagePipeline.Context())
	n.ctx = ctx
	n.ctxCancel = cancel
	go n.monitorMgr.Run(ctx)
	go n.run()
}

// TODO: https://nginxsoftware.atlassian.net/browse/NMS-38140
//   - Identify if we need to process any interactions with NGINX
func (n *NAPMonitoring) Process(msg *core.Message) {}

// TODO: https://nginxsoftware.atlassian.net/browse/NMS-38140
//   - Subscribe for Agent config updates
func (n *NAPMonitoring) Subscriptions() []string {
	return []string{}
}

func (n *NAPMonitoring) Close() {
	log.Infof("%s is wrapping up", napMonitoringPluginName)
	n.ctxCancel()
}

func (n *NAPMonitoring) run() {
	defer n.Close()

	riTicker := time.NewTicker(reportInterval)
	defer riTicker.Stop()

	report := &events.EventReport{
		Events: []*events.Event{},
	}

	for {
		select {
		case event, ok := <-n.monitorMgr.OutChannel():
			if !ok {
				log.Errorf("NAP Monitoring processing channel closed unexpectedly")
				return
			}
			report.Events = append(report.Events, event)
			if len(report.Events) == reportCount {
				n.send(report, "report count reached. sending report: %v")
			}
		case <-riTicker.C:
			if len(report.Events) > 0 {
				n.send(report, "report interval reached. sending report: %v")
			}
		case <-n.ctx.Done():
			return
		}
	}
}

func (n *NAPMonitoring) send(report *events.EventReport, logMsg string) {
	log.Debugf(logMsg, report)
	n.messagePipeline.Process(core.NewMessage(core.CommMetrics, []core.Payload{report}))
	report.Events = []*events.Event{}
}
