package performance

import (
	"context"
	"fmt"
	"os"

	"net"
	"sync"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	"github.com/nginx/agent/sdk/v2/client"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"
	f5_nginx_agent_sdk_events "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/logger"
	"github.com/nginx/agent/v2/src/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var (
	addr        = "127.0.0.1:90"
	network     = "tcp"
	messageChan = make(chan *proto.MetricsReport)
	eventChan   = make(chan *f5_nginx_agent_sdk_events.EventReport)
)

func BenchmarkMetrics(b *testing.B) {
	startFakeServer()
	time.Sleep(5 * time.Second)
	startNginxAgent(b)
	metricReport := <-messageChan
	require.NotNil(b, metricReport, "Metric report should not be nil")
	// Start timer after first report is received
	b.ResetTimer()
	b.Run("metrics", func(bb *testing.B) {
		bb.ReportAllocs()
		for n := 0; n < b.N; n++ {
			metricReport := <-messageChan
			require.NotNil(bb, metricReport, "Metric report should not be nil")
		}
	})
}

type MetricsServer struct {
	messageChan chan *proto.MetricsReport
	sync.RWMutex
	metricHandler *metricHandler
}

func NewMetricsServer() *MetricsServer {
	return &MetricsServer{
		messageChan: make(chan *proto.MetricsReport),
	}
}

type metricHandlerFunc func(proto.MetricsService_StreamServer, *sync.WaitGroup)
type eventReportHandlerFunc func(proto.MetricsService_StreamEventsServer, *sync.WaitGroup)
type metricHandler struct {
	msgCount               atomic.Int64
	handleCount            atomic.Int64
	metricHandlerFunc      metricHandlerFunc
	eventReportHandlerFunc eventReportHandlerFunc
}

func (m *MetricsServer) Stream(stream proto.MetricsService_StreamServer) error {
	wg := &sync.WaitGroup{}
	h := m.ensureHandler()
	wg.Add(1)
	hf := h.metricHandlerFunc
	if hf == nil {
		hf = h.metricsHandle
	}
	go hf(stream, wg)
	wg.Wait()
	return nil
}

func (m *MetricsServer) StreamEvents(stream proto.MetricsService_StreamEventsServer) error {
	wg := &sync.WaitGroup{}
	h := m.ensureHandler()
	wg.Add(1)
	hf := h.eventReportHandlerFunc
	if hf == nil {
		hf = h.eventReportHandle
	}
	go hf(stream, wg)
	wg.Wait()
	return nil
}

func (m *MetricsServer) ensureHandler() *metricHandler {
	m.RLock()
	if m.metricHandler == nil {
		m.RUnlock()
		m.Lock()
		defer m.Unlock()
		m.metricHandler = &metricHandler{
			msgCount: atomic.Int64{},
		}
		return m.metricHandler
	}
	defer m.RUnlock()
	return m.metricHandler
}

func (h *metricHandler) metricsHandle(server proto.MetricsService_StreamServer, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	h.handleCount.Inc()
	for {
		metricsReport, err := server.Recv()
		if err != nil {
			return
		}
		messageChan <- metricsReport
		h.msgCount.Inc()
	}
}

func (h *metricHandler) eventReportHandle(server proto.MetricsService_StreamEventsServer, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	for {
		eventReport, err := server.Recv()
		if err != nil {
			return
		}
		eventChan <- eventReport
	}
}

type handlerFunc func(proto.Commander_CommandChannelServer, *sync.WaitGroup)

type handler struct {
	msgCount    atomic.Int64
	handleCount atomic.Int64
	handleFunc  handlerFunc
}

type cmdService struct {
	sync.RWMutex
	handler *handler
}

func (c *cmdService) CommandChannel(server proto.Commander_CommandChannelServer) error {
	wg := &sync.WaitGroup{}
	h := c.ensureHandler()
	wg.Add(1)
	hf := h.handleFunc
	if hf == nil {
		hf = h.handle
	}
	go hf(server, wg)
	wg.Wait()
	return nil
}

func (c *cmdService) Download(request *proto.DownloadRequest, server proto.Commander_DownloadServer) error {
	panic("Not Implemented")
}

func (c *cmdService) Upload(server proto.Commander_UploadServer) error {
	_, err := server.Recv()
	server.SendAndClose(
		&proto.UploadStatus{
			Meta: &proto.Metadata{
				Timestamp: types.TimestampNow(),
				ClientId:  "1",
				MessageId: "1",
			},
			Status: proto.UploadStatus_OK,
			Reason: "",
		})

	return err
}

func (c *cmdService) ensureHandler() *handler {
	c.RLock()
	if c.handler == nil {
		c.RUnlock()
		c.Lock()
		defer c.Unlock()
		c.handler = &handler{
			msgCount: atomic.Int64{},
		}
		return c.handler
	}
	defer c.RUnlock()
	return c.handler
}

func (h *handler) handle(server proto.Commander_CommandChannelServer, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()
	h.handleCount.Inc()
	for {
		_, err := server.Recv()
		if err != nil {
			fmt.Printf("Command Error: %v\n", err)
			return
		}
		h.msgCount.Inc()
	}
}

func startFakeServer() *MetricsServer {
	lis, err := net.Listen(network, addr)
	if err != nil {
		fmt.Printf("failed to initialize listener: %v", err)
	}

	enforcement := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second,
		PermitWithoutStream: true,
	}

	srvOptions := []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(enforcement),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge:      10 * time.Second,
			MaxConnectionAgeGrace: 300 * time.Second,
		}),
	}

	grpcServer := grpc.NewServer(srvOptions...)

	metricsServer := NewMetricsServer()
	proto.RegisterMetricsServiceServer(grpcServer, metricsServer)

	commandServer := &cmdService{}
	proto.RegisterCommanderServer(grpcServer, commandServer)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			fmt.Print(err)
		}
	}()

	return metricsServer
}

func startNginxAgent(b *testing.B) {
	mu := &sync.Mutex{}
	ctx := context.Background()
	env := &core.EnvironmentType{}
	configPath, _ := config.RegisterConfigFile(
		"../testdata/configs/agent-dynamic.conf",
		"nginx-agent-metrics.conf",
		"../testdata/configs/",
	)
	config.Viper.Set(config.ConfigPathKey, configPath)

	loadedConfig, _ := config.GetConfig(env.GetSystemUUID())
	logger.SetLogLevel("error")
	os.Create("/var/log/nginx-agent.log")
	logger.SetLogFile("/var/log/nginx-agent.log")
	binary := core.NewNginxBinary(env, loadedConfig)
	sdkGRPC.InitMeta(loadedConfig.ClientID, loadedConfig.CloudAccountID)

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DataplaneConnectionDialOptions(loadedConfig.Server.Token, sdkGRPC.NewMessageMeta(uuid.New().String()))...)

	secureMetricsDialOpts, err := sdkGRPC.SecureDialOptions(
		loadedConfig.TLS.Enable,
		loadedConfig.TLS.Cert,
		loadedConfig.TLS.Key,
		loadedConfig.TLS.Ca,
		loadedConfig.Server.Metrics,
		loadedConfig.TLS.SkipVerify)
	if err != nil {
		fmt.Printf("Failed to load secure metric gRPC dial options: %v", err)
	}

	secureCmdDialOpts, err := sdkGRPC.SecureDialOptions(
		loadedConfig.TLS.Enable,
		loadedConfig.TLS.Cert,
		loadedConfig.TLS.Key,
		loadedConfig.TLS.Ca,
		loadedConfig.Server.Command,
		loadedConfig.TLS.SkipVerify)
	if err != nil {
		fmt.Printf("Failed to load secure command gRPC dial options: %v", err)
	}

	controller := client.NewClientController()
	controller.WithContext(ctx)

	commander := client.NewCommanderClient()
	commander.WithServer(loadedConfig.Server.Target)
	commander.WithDialOptions(append(grpcDialOptions, secureCmdDialOpts)...)

	reporter := client.NewMetricReporterClient()
	reporter.WithServer(loadedConfig.Server.Target)
	reporter.WithDialOptions(append(grpcDialOptions, secureMetricsDialOpts)...)

	controller.WithClient(commander)
	controller.WithClient(reporter)
	if err := controller.Connect(); err != nil {
		fmt.Printf("Unable to connect to control plane: %v", err)
		return
	}
	var corePlugins []core.Plugin

	corePlugins = append(corePlugins,
		plugins.NewConfigReader(loadedConfig),
		plugins.NewNginx(commander, binary, env, &config.Config{}),
		plugins.NewCommander(commander, loadedConfig),
		plugins.NewMetricsSender(reporter),
		plugins.NewOneTimeRegistration(loadedConfig, binary, env, sdkGRPC.NewMessageMeta(uuid.New().String()), "1.0.0"),
		plugins.NewMetrics(loadedConfig, env, binary),
		plugins.NewMetricsThrottle(loadedConfig, env),
		plugins.NewDataPlaneStatus(loadedConfig, sdkGRPC.NewMessageMeta(uuid.New().String()), binary, env, "1.0.0"),
	)

	messagePipe := core.NewMessagePipe(ctx)
	err = messagePipe.Register(100, corePlugins, []core.ExtensionPlugin{})
	assert.NoError(b, err)

	go func() {
		mu.Lock()
		defer mu.Unlock()
		messagePipe.Run()
	}()

}
