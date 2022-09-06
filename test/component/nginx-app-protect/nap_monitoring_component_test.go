package nginx_app_protect

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"io/ioutil"
	"log/syslog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	events "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/manager"
)

func TestNAPMonitoring(t *testing.T) {
	cfg := &config.Config{
		Server: config.Server{
			Host:     "localhost",
			GrpcPort: 8443,
		},
		TLS: config.TLSConfig{
			Enable: false,
		},
		NAPMonitoring: config.NAPMonitoring{
			CollectorBufferSize: 50,
			ProcessorBufferSize: 50,
			SyslogIP:            "0.0.0.0",
			SyslogPort:          1234,
		},
	}

	sem, err := manager.NewSecurityEventManager(cfg)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go sem.Run(ctx)

	// Let monitor init
	time.Sleep(3 * time.Second)

	sysLog, err := syslog.Dial("tcp", "localhost:1234", syslog.LOG_WARNING, "napMonitoringTest")
	assert.NoError(t, err)

	ingestionServer, err := NewIngestionServerTest(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.GrpcPort))
	assert.NoError(t, err)

	go ingestionServer.Run(ctx)

	files, err := ioutil.ReadDir("./testData/logs-in/")
	assert.NoError(t, err)
	for _, file := range files {
		bInput, err := ioutil.ReadFile(fmt.Sprintf("./testData/logs-in/%s", file.Name()))
		assert.NoError(t, err)
		input := string(bInput)
		_, err = fmt.Fprintf(sysLog, input)
		assert.NoError(t, err)
	}

	// Let monitor work
	time.Sleep(5 * time.Second)

	files, err = ioutil.ReadDir("./testData/events-out/")
	assert.NoError(t, err)
	for _, file := range files {
		bEvent, err := ioutil.ReadFile(fmt.Sprintf("./testData/events-out/%s", file.Name()))
		assert.NoError(t, err)
		expectedEvent := &events.Event{}
		err = proto.Unmarshal(bEvent, expectedEvent)
		assert.NoError(t, err)

		resultEvent, found := ingestionServer.ReceivedEvent(expectedEvent.GetSecurityViolationEvent().SupportID)
		assert.True(t, found)
		assertEqualSecurityViolationEvents(t, expectedEvent, resultEvent)
	}
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
}
