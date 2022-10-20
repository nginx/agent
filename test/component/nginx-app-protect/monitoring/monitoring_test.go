package monitoring

import (
	"context"
	"fmt"
	"log/syslog"
	"math/rand"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/nginx/agent/sdk/v2/client"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	events "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/plugins"
	"github.com/nginx/agent/v2/test/component/nginx-app-protect/monitoring/mock"
)

const (
	DEFAULT_PLUGIN_SIZE = 100
)

func TestNAPMonitoring(t *testing.T) {
	cfg := &config.Config{
		Server: config.Server{
			Host:     "localhost",
			GrpcPort: EphemeralPort(),
			Token:    uuid.New().String(),
		},
		TLS: config.TLSConfig{
			Enable: false,
		},
		NAPMonitoring: config.NAPMonitoring{
			CollectorBufferSize: 50,
			ProcessorBufferSize: 50,
			SyslogIP:            "127.0.0.1",
			SyslogPort:          EphemeralPort(),
			ReportInterval:      time.Minute,
			// Since the minimum report interval is on minute, MAP monitor won't have enough time within the test timeframe
			// to send the report. So always beware of the count of logged attacks and set report count accordingly
			// Count of attacks = count of files under ./testData/logs-in/
			ReportCount: 7,
		},
	}

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

	comms := plugins.NewComms(reporter)

	napMonitoring, err := plugins.NewNAPMonitoring(cfg)
	assert.NoError(t, err)

	pipe := initializeMessagePipe(t, ctx, []core.Plugin{comms, napMonitoring})

	pipe.Process(core.NewMessage(core.RegistrationCompletedTopic, nil))

	wg.Add(1)
	go func() {
		defer wg.Done()
		pipe.Run()
	}()

	// Let monitor init
	time.Sleep(5 * time.Second)

	sysLog, err := syslog.Dial("tcp", fmt.Sprintf("%s:%d", cfg.NAPMonitoring.SyslogIP, cfg.NAPMonitoring.SyslogPort), syslog.LOG_WARNING, "napMonitoringTest")
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
		bEvent, err := os.ReadFile(fmt.Sprintf("./testData/events-out/%s", file.Name()))
		assert.NoError(t, err)
		expectedEvent := &events.Event{}
		err = proto.Unmarshal(bEvent, expectedEvent)
		assert.NoError(t, err)

		resultEvent, found := ingestionServer.ReceivedEvent(expectedEvent.GetSecurityViolationEvent().SupportID)
		assert.True(t, found)
		if !found {
			break
		}
		assertEqualSecurityViolationEvents(t, expectedEvent, resultEvent)
	}

	err = reporter.Close()
	assert.NoError(t, err)
	cancel()
	wg.Wait()
}

func assertEqualSecurityViolationEvents(t *testing.T, expectedEvent, resultEvent *events.Event) {
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().PolicyName, resultEvent.GetSecurityViolationEvent().PolicyName)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Outcome, resultEvent.GetSecurityViolationEvent().Outcome)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().OutcomeReason, resultEvent.GetSecurityViolationEvent().OutcomeReason)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Method, resultEvent.GetSecurityViolationEvent().Method)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Protocol, resultEvent.GetSecurityViolationEvent().Protocol)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().URI, resultEvent.GetSecurityViolationEvent().URI)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().Request, resultEvent.GetSecurityViolationEvent().Request)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().RequestStatus, resultEvent.GetSecurityViolationEvent().RequestStatus)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().ResponseCode, resultEvent.GetSecurityViolationEvent().ResponseCode)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().UnitHostname, resultEvent.GetSecurityViolationEvent().UnitHostname)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().VSName, resultEvent.GetSecurityViolationEvent().VSName)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().IPClient, resultEvent.GetSecurityViolationEvent().IPClient)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().DestinationPort, resultEvent.GetSecurityViolationEvent().DestinationPort)
	assert.Equal(t, expectedEvent.GetSecurityViolationEvent().SourcePort, resultEvent.GetSecurityViolationEvent().SourcePort)
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

func initializeMessagePipe(t *testing.T, ctx context.Context, corePlugins []core.Plugin) *core.MessagePipe {
	pipe := core.NewMessagePipe(ctx)
	err := pipe.Register(DEFAULT_PLUGIN_SIZE, corePlugins...)
	assert.NoError(t, err)
	return pipe
}

func setDialOptions(loadedConfig *config.Config) []grpc.DialOption {
	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DataplaneConnectionDialOptions(loadedConfig.Server.Token, sdkGRPC.NewMessageMeta(uuid.NewString()))...)
	return grpcDialOptions
}

/*
	ne := &pb.Event{
		Data: &pb.Event_SecurityViolationEvent{
			SecurityViolationEvent: &pb.SecurityViolationEvent{
				PolicyName:       "extract from input log",
				SupportID:        "extract from input log",
				Outcome:          "extract from input log",
				OutcomeReason:    "extract from input log",
				Method:           "extract from input log",
				Protocol:         "extract from input log",
				URI:              "extract from input log",
				Request:          "extract from input log",
				RequestStatus:    "extract from input log",
				ResponseCode:     "extract from input log",
				UnitHostname:     "extract from input log",
				VSName:           "extract from input log",
				IPClient:         "extract from input log",
				DestinationPort:  "extract from input log",
				SourcePort:       "extract from input log",
				Violations:       "extract from input log",
				ClientClass:      "extract from input log",
				Severity:         "extract from input log",
				BotSignatureName: "extract from input log",
				ViolationsData:   "extract from input log",
				ViolationsData:   []*pb.ViolationData{
					{
						Name: "extract from input log",
						Context: "extract from input log",
						ContextData: &pb.ContextData{
							Name:                 "extract from input log",
							Value:                "extract from input log",
						},
					},
				},
			},
		},
	}
	bEvent, err := proto.Marshal(ne)
	if err != nil {
		t.Fatalf("Error while marshaling event: %v", err)
	}
	if err := ioutil.WriteFile(tc.testFile+".out", bEvent, 0644); err != nil {
		log.Fatalln("Failed to write event:", err)
	}
*/
