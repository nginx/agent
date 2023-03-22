package monitoring

import (
	"context"
	"fmt"
	"log/syslog"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	proto "github.com/golang/protobuf/jsonpb"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	testifyMock "github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/client"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	sdkPb "github.com/nginx/agent/sdk/v2/proto"
	events "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/manager"
	"github.com/nginx/agent/v2/src/plugins"
	"github.com/nginx/agent/v2/test/component/nginx-app-protect/monitoring/mock"
	tutils "github.com/nginx/agent/v2/test/utils"
)

const (
	DEFAULT_PLUGIN_SIZE = 100
)

func TestNAPMonitoring(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	cfg := &config.Config{
		Server: config.Server{
			Host:     "localhost",
			GrpcPort: EphemeralPort(),
			Token:    uuid.New().String(),
		},
		TLS: config.TLSConfig{
			Enable: false,
		},
		Tags:          []string{"tag1", "tag2"},
		DisplayName:   "nap-monitoring-component-test",
		InstanceGroup: "instance-group1",
		Extensions: []string{
			agent_config.NginxAppProtectMonitoringExtensionPlugin,
		},
	}

	nginxAppProtectMonitoringConfig := manager.NginxAppProtectMonitoringConfig{
		CollectorBufferSize: 50,
		ProcessorBufferSize: 50,
		SyslogIP:            "127.0.0.1",
		SyslogPort:          EphemeralPort(),
		ReportInterval:      time.Minute,
		// Since the minimum report interval is one minute, MAP monitor won't have enough time within the test timeframe
		// to send the report. So always beware of the count of logged attacks and set report count accordingly
		// Count of attacks = count of files under ./testData/logs-in/
		ReportCount: 7,
	}

	// Expected common dimensions that need to be added to the generated SecurityViolationEvent
	// Values of hostname and uuid are coming from test/utils/environment(*MockEnvironment.NewHostInfo)
	expectedHostname := "test-host"
	expectedUUID := uuid.NewString()
	expectedInstanceTags := strings.Join(cfg.Tags, ",")
	expectedDisplayName := cfg.DisplayName
	expectedInstanceGroup := cfg.InstanceGroup

	ctx, cancel := context.WithCancel(context.Background())

	ingestionServer, err := mock.NewIngestionServerMock(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GrpcPort))
	assert.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		ingestionServer.Run(ctx)
	}()

	grpcDialOptions := setDialOptions(cfg)
	secureMetricsDialOpts, err := sdkGRPC.SecureDialOptions(
		cfg.TLS.Enable,
		cfg.TLS.Cert,
		cfg.TLS.Key,
		cfg.TLS.Ca,
		cfg.Server.Metrics,
		cfg.TLS.SkipVerify)
	assert.NoError(t, err)
	if err != nil {
		cancel()
		return
	}

	reporter := client.NewMetricReporterClient()
	reporter.WithServer(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GrpcPort))
	reporter.WithDialOptions(append(grpcDialOptions, secureMetricsDialOpts)...)

	err = reporter.Connect(ctx)
	assert.NoError(t, err)
	if err != nil {
		cancel()
		return
	}

	metricsSender := plugins.NewMetricsSender(reporter)

	env := tutils.NewMockEnvironment()
	env.On("NewHostInfo", testifyMock.Anything, testifyMock.Anything, testifyMock.Anything).Return(&sdkPb.HostInfo{
		Hostname: expectedHostname,
		Uuid:     expectedUUID,
	})

	napMonitoring, err := extensions.NewNAPMonitoring(env, cfg, nginxAppProtectMonitoringConfig)
	assert.NoError(t, err)

	pipe := initializeMessagePipe(t, ctx, []core.Plugin{metricsSender}, []core.ExtensionPlugin{napMonitoring})

	pipe.Process(core.NewMessage(core.RegistrationCompletedTopic, nil))

	wg.Add(1)
	go func() {
		defer wg.Done()
		pipe.Run()
	}()

	// Let monitor init
	time.Sleep(5 * time.Second)

	sysLog, err := syslog.Dial("tcp", fmt.Sprintf("%s:%d", nginxAppProtectMonitoringConfig.SyslogIP, nginxAppProtectMonitoringConfig.SyslogPort), syslog.LOG_WARNING, "napMonitoringTest")
	assert.NoError(t, err)

	files, err := os.ReadDir("./testData/logs-in/")
	assert.NoError(t, err)
	for _, file := range files {
		bInput, err := os.ReadFile(fmt.Sprintf("./testData/logs-in/%s", file.Name()))
		assert.NoError(t, err)
		input := string(bInput)
		_, err = fmt.Fprint(sysLog, input)
		assert.NoError(t, err)
	}

	// Let monitor work
	time.Sleep(5 * time.Second)

	files, err = os.ReadDir("./testData/events-out/")
	assert.NoError(t, err)
	for _, file := range files {
		fName := fmt.Sprintf("./testData/events-out/%s", file.Name())
		log.Debugf("Running Monitoring Test for %s", fName)

		bEvent, err := os.ReadFile(fName)
		assert.NoError(t, err)
		expectedEvent := &events.Event{}
		err = proto.UnmarshalString(string(bEvent), expectedEvent)
		assert.NoError(t, err)

		resultEvent, found := ingestionServer.ReceivedEvent(expectedEvent.GetSecurityViolationEvent().SupportID)
		assert.True(t, found)
		if !found {
			break
		}
		assertEqualSecurityViolationEvents(t, expectedEvent, resultEvent)
		assert.Equal(t, expectedHostname, resultEvent.GetSecurityViolationEvent().ParentHostname)
		assert.Equal(t, expectedUUID, resultEvent.GetSecurityViolationEvent().SystemID)
		assert.Equal(t, expectedInstanceTags, resultEvent.GetSecurityViolationEvent().InstanceTags)
		assert.Equal(t, expectedInstanceGroup, resultEvent.GetSecurityViolationEvent().InstanceGroup)
		assert.Equal(t, expectedDisplayName, resultEvent.GetSecurityViolationEvent().DisplayName)
	}

	err = reporter.Close()
	assert.NoError(t, err)
	cancel()
	wg.Wait()
}

func assertEqualSecurityViolationEvents(t *testing.T, expectedEvent, resultEvent *events.Event) {
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().SupportID, resultEvent.GetSecurityViolationEvent().SupportID)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().PolicyName, resultEvent.GetSecurityViolationEvent().PolicyName)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Outcome, resultEvent.GetSecurityViolationEvent().Outcome)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().OutcomeReason, resultEvent.GetSecurityViolationEvent().OutcomeReason)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Method, resultEvent.GetSecurityViolationEvent().Method)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Protocol, resultEvent.GetSecurityViolationEvent().Protocol)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().URI, resultEvent.GetSecurityViolationEvent().URI)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Request, resultEvent.GetSecurityViolationEvent().Request)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().RequestStatus, resultEvent.GetSecurityViolationEvent().RequestStatus)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().ResponseCode, resultEvent.GetSecurityViolationEvent().ResponseCode)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().VSName, resultEvent.GetSecurityViolationEvent().VSName)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().RemoteAddr, resultEvent.GetSecurityViolationEvent().RemoteAddr)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().RemotePort, resultEvent.GetSecurityViolationEvent().RemotePort)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().ServerPort, resultEvent.GetSecurityViolationEvent().ServerPort)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Violations, resultEvent.GetSecurityViolationEvent().Violations)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Severity, resultEvent.GetSecurityViolationEvent().Severity)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().ClientClass, resultEvent.GetSecurityViolationEvent().ClientClass)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().BotSignatureName, resultEvent.GetSecurityViolationEvent().BotSignatureName)
	assertEqualSecurityViolationsDetails(t, expectedEvent.GetSecurityViolationEvent().ViolationsData, resultEvent.GetSecurityViolationEvent().ViolationsData)
}

func assertEqualSecurityViolationsDetails(t *testing.T, expectedDetails, resultDetails []*events.ViolationData) {
	for i, expected := range expectedDetails {
		result := resultDetails[i]
		assert.Equal(t, expected.Name, result.Name)
		assert.Equal(t, expected.Context, result.Context)
		if expected.ContextData != nil {
			assert.Equal(t, expected.ContextData.Name, result.ContextData.Name)
			assert.Equal(t, expected.ContextData.Value, result.ContextData.Value)
		}
	}
}

func EphemeralPort() int {
	base := 32768
	port := base + rand.Intn(10000)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	for err != nil {
		port = base + rand.Intn(10000)
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	}
	_ = ln.Close()
	return port
}

func initializeMessagePipe(t *testing.T, ctx context.Context, corePlugins []core.Plugin, extensionPlugins []core.ExtensionPlugin) *core.MessagePipe {
	pipe := core.NewMessagePipe(ctx)
	err := pipe.Register(DEFAULT_PLUGIN_SIZE, corePlugins, extensionPlugins)
	assert.NoError(t, err)
	return pipe
}

func setDialOptions(loadedConfig *config.Config) []grpc.DialOption {
	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DataplaneConnectionDialOptions(loadedConfig.Server.Token, sdkGRPC.NewMessageMeta(uuid.NewString()))...)
	return grpcDialOptions
}
