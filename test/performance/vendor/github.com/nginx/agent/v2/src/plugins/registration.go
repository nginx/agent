package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/payloads"
)

const (
	// Timeout for registration attempting to gather DataplaneSoftwareDetails
	dataplaneSoftwareDetailsMaxWaitTime = time.Duration(5 * time.Second)
	// Time between attempts to gather DataplaneSoftwareDetails
	softwareDetailsOperationInterval = time.Duration(1 * time.Second)
)

type OneTimeRegistration struct {
	agentVersion             string
	tags                     *[]string
	meta                     *proto.Metadata
	config                   *config.Config
	env                      core.Environment
	host                     *proto.HostInfo
	binary                   core.NginxBinary
	dataplaneSoftwareDetails map[string]*proto.DataplaneSoftwareDetails
	pipeline                 core.MessagePipeInterface
}

func NewOneTimeRegistration(
	config *config.Config,
	binary core.NginxBinary,
	env core.Environment,
	meta *proto.Metadata,
	version string,
) *OneTimeRegistration {
	// this might be slow so do on startup
	host := env.NewHostInfo(version, &config.Tags, config.ConfigDirs, true)
	return &OneTimeRegistration{
		tags:                     &config.Tags,
		agentVersion:             version,
		meta:                     meta,
		config:                   config,
		env:                      env,
		host:                     host,
		binary:                   binary,
		dataplaneSoftwareDetails: make(map[string]*proto.DataplaneSoftwareDetails),
	}
}

func (r *OneTimeRegistration) Init(pipeline core.MessagePipeInterface) {
	log.Info("OneTimeRegistration initializing")
	r.pipeline = pipeline
	r.startRegistration()
}

func (r *OneTimeRegistration) Close() {
	log.Info("OneTimeRegistration is wrapping up")
}

func (r *OneTimeRegistration) Info() *core.Info {
	return core.NewInfo("OneTimeRegistration", "v0.0.1")
}

func (r *OneTimeRegistration) Process(msg *core.Message) {
	switch {
	case msg.Exact(core.RegistrationCompletedTopic):
		log.Info("OneTimeRegistration completed")
	}
}

func (r *OneTimeRegistration) Subscriptions() []string {
	return []string{core.RegistrationCompletedTopic}
}

func (r *OneTimeRegistration) startRegistration() {
	// Check if there are any plugins that will report dataplane software details upon registration and
	// if they have already reported their details or not.
	if pluginsReportingDataplaneSoftwareDetails(*r.config) && r.dataplaneSoftwareDetailsMissing() {
		r.registerWithDataplaneSoftwareDetails()
		return
	}
	r.registerAgent()
}

// pluginsReportingDataplaneDetails returns a bool indicating if there are any plugins
// enabled that transmit dataplane software details based off the config passed. True is
// returned if there are plugins enabled that report dataplane software details.
func pluginsReportingDataplaneSoftwareDetails(conf config.Config) bool {
	return conf.NginxAppProtect != (config.NginxAppProtect{})
}

func (r *OneTimeRegistration) registerAgent() {
	var details []*proto.NginxDetails

	for _, proc := range r.env.Processes() {
		// only need master process for registration
		if proc.IsMaster {
			details = append(details, r.binary.GetNginxDetailsFromProcess(proc))
		} else {
			log.Tracef("NGINX non-master process: %d", proc.Pid)
		}
	}
	if len(details) == 0 {
		log.Info("No master process found")
	}
	updated, err := types.TimestampProto(r.config.Updated)
	if err != nil {
		log.Warnf("failed to parse proto timestamp %s: %s, assuming now", r.config.Updated, err)
		updated = types.TimestampNow()
	}
	log.Infof("Registering %s", r.env.GetSystemUUID())

	agentConnectRequest := &proto.Command{
		Meta: r.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_AgentConnectRequest{
			AgentConnectRequest: &proto.AgentConnectRequest{
				Host: r.host,
				Meta: &proto.AgentMeta{
					Version:       r.agentVersion,
					DisplayName:   r.config.DisplayName,
					Tag:           *r.tags,
					InstanceGroup: r.config.InstanceGroup,
					Updated:       updated,
					SystemUid:     r.env.GetSystemUUID(),
				},
				Details:                  details,
				DataplaneSoftwareDetails: getSoftwareDetails(r.env, r.binary, nil),
			},
		},
	}

	log.Tracef("AgentConnectRequest: %v", agentConnectRequest)

	r.pipeline.Process(
		core.NewMessage(core.CommRegister, agentConnectRequest),
		core.NewMessage(core.RegistrationCompletedTopic, nil),
	)
}

// registerWithDataplaneSoftwareDetails Attempts to ensure that the plugins enabled that transmit dataplane
// software details have transmitted their details to OneTimeRegistration then registers.
func (r *OneTimeRegistration) registerWithDataplaneSoftwareDetails() {
	for _, plugin := range getPluginsReportingDataplaneSoftwareDetails(*r.config) {
		r.dataplaneSoftwareDetails[plugin] = nil
	}

	registrationPayload := payloads.NewRegisterWithDataplaneSoftwareDetailsPayload(r.dataplaneSoftwareDetails)
	r.pipeline.Process(core.NewMessage(core.RegisterWithDataplaneSoftwareDetails, registrationPayload))

	go r.waitAndRegister()
}

// waitAndRegister checks in a retry loop if the plugins enabled that transmit dataplane
// software details have transmitted their details to OneTimeRegistration then registers.
// If the plugins do not successfully transmit their details before the max retries is
// reached then an error will be logged then registration will start with whatever
// dataplane software details were successfully transmitted (if any).
func (r *OneTimeRegistration) waitAndRegister() {
	log.Debug("OneTimeRegistration waiting on dataplane software details to be ready for registration")
	err := sdk.WaitUntil(
		context.Background(), softwareDetailsOperationInterval, softwareDetailsOperationInterval,
		dataplaneSoftwareDetailsMaxWaitTime, r.dataplaneSoftwareDetailsReady,
	)
	if err != nil {
		log.Warn(err.Error())
	}

	r.registerAgent()
}

// dataplaneSoftwareDetailsReady Determines if all the plugins enabled that transmit dataplane
// software details have transmitted their details to OneTimeRegistration. An error is returned
// if any plugins enabled that transmit dataplane software details have not transmitted their
// details to OneTimeRegistration.
func (r *OneTimeRegistration) dataplaneSoftwareDetailsReady() error {
	pluginsMissingDetails := []string{}

	for pluginName, detailsReported := range r.dataplaneSoftwareDetails {
		if detailsReported == nil {
			pluginsMissingDetails = append(pluginsMissingDetails, pluginName)
		}
	}

	if len(pluginsMissingDetails) > 0 {
		log.Warnf("The following dataplane software details are not ready for registration - %v", pluginsMissingDetails)
		return fmt.Errorf("OneTimeRegistration max retries has been met before the following dataplane software details were ready for registration - %v", pluginsMissingDetails)
	}

	log.Debug("All dataplane software details are ready for registration")
	return nil
}

// dataplaneSoftwareDetailsMissing returns a bool indicating if the plugins enabled that
// transmit dataplane software details have already transmitted their details to
// OneTimeRegistration. If they have then false is returned, if not then true is returned.
func (r *OneTimeRegistration) dataplaneSoftwareDetailsMissing() bool {
	for _, plugin := range getPluginsReportingDataplaneSoftwareDetails(*r.config) {
		if _, ok := r.dataplaneSoftwareDetails[plugin]; !ok {
			return true
		}
	}
	return false
}


// getPluginsReportingDataplaneSoftwareDetails returns a list of plugin names that
// are enabled which transmit dataplane software details based off the config passed.
func getPluginsReportingDataplaneSoftwareDetails(conf config.Config) []string {
	plugins := make([]string, 0)

	if conf.NginxAppProtect != (config.NginxAppProtect{}) {
		plugins = append(plugins, napPluginName)
	}

	return plugins
}
