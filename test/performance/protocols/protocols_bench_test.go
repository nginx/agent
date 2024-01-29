package protocols

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/nginx/agent/v3/test/performance/protocols/proto"
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type GrpcServer struct {
	toClient   chan *proto.Message
	fromClient chan *proto.Message
	proto.UnsafeMessengerServer
}

const (
	PROTOCOL = "tcp"
)

var (
	grpcServer *grpc.Server
	httpServer *http.Server
	wg         sync.WaitGroup
	protoMsg   *proto.Message
	msg        = `{
		"timestamp": "1706287128",
		"type": "Attachment",
		"tenantId": "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		"correlationId": "9e2d49c9-ada2-4ed1-b6f2-391d5cf634d9",
		"instances": [
		  {
			"instanceId": "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
			"type": "NGINX",
			"version": "nginx/1.25.3",
			"meta": {
				"prefix": "/opt/homebrew/Cellar/nginx/1.25.3",
				"sbin-path": "/opt/homebrew/Cellar/nginx/1.25.3/bin/nginx",
				"conf-path": "/opt/homebrew/etc/nginx/nginx.conf"
			}
		  }
		]
	  }`

	serverDialOptions = []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(grpc_validator.UnaryServerInterceptor()),
		grpc.StreamInterceptor(grpc_validator.StreamServerInterceptor()),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             60 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	clientDialOptions = []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor()),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                120 * time.Second,
			Timeout:             60 * time.Second,
			PermitWithoutStream: true,
		}),
	}
)

func BenchmarkREST(b *testing.B) {
	startRESTServer()
	tenantId, baseURL, httpClient := createRESTClient()

	b.Run("bench_rest", func(bb *testing.B) {
		bb.ResetTimer()
		for i := 0; i < b.N; i++ {
			runRESTClient(b, i, tenantId, baseURL, httpClient)
		}
	})
	cleanup()
}

func BenchmarkGRPC(b *testing.B) {
	serverClose := startGRPCServer()
	stream, clientClose := createGRPCClient(b)

	b.Run("bench_grpc", func(bb *testing.B) {
		bb.ResetTimer()
		for i := 0; i < b.N; i++ {
			runGRPCClient(b, i, stream)
		}
	})
	cleanup()
	defer serverClose()
	defer clientClose()
}

func getInstances(c *gin.Context) {
	protoMsg = &proto.Message{
		Timestamp:     "1706287128",
		Correlationid: "9e2d49c9-ada2-4ed1-b6f2-391d5cf634d9",
		Tenantid:      "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		Type:          proto.Message_ATTACHMENT,
		Union: &proto.Message_Attachement{
			Attachement: &proto.Attachment{
				Instances: []*proto.Instances{
					{
						Instance: &proto.Instance{
							InstanceId: "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
							Version:    "1.25.1",
							Meta: &proto.Instance_NginxMeta{
								NginxMeta: &proto.NginxMeta{
									LoadableModules: "njs",
									RunnableModules: "something",
								},
							},
						},
					},
				},
			},
		},
	}
	bytes, err := protojson.Marshal(protoMsg)
	if err != nil {
		log.Fatal(err)
	}
	c.Header("Content-Type", "application/json")
	c.Writer.Write(bytes)
}

func startRESTServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())
	gin.SetMode(gin.ReleaseMode)
	r.POST("/instances", getInstances)

	httpServer = &http.Server{
		Addr:    "localhost:8080",
		Handler: r,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
		log.Print("finished setting up server")
	}()

	log.Print("finished setting up server")
}

func startGRPCServer() func() error {
	grpcServer, listener, close := NewServer("localhost:50051")
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()
	return close
}

func runRESTClient(b *testing.B, index int, tenantId, baseURL string, client http.Client) {
	// Simulate a REST API request
	response, err := SendRESTMessage(fmt.Sprintf("%s/instances/", baseURL), msg, tenantId, &client)
	if err != nil {
		b.Fatal(err)
	}
	assert.NotNil(b, response)
}

func createRESTClient() (string, string, http.Client) {
	tenantId := "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"
	baseURL := "http://localhost:8080"
	httpClient := http.Client{
		Timeout: time.Second * 1,
	}
	return tenantId, baseURL, httpClient
}

func runGRPCClient(b *testing.B, index int, stream proto.Messenger_MessageChannelClient) {
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				errStatus, ok := status.FromError(err)
				if !ok && errStatus.Code() != codes.Unavailable && errStatus.Message() != "the client connection is closing" {
					log.Fatalf("Failed to receive a note : %v", err)
				}
			}
			log.Printf("Got message %v", in.GetAttachement())
		}
	}()
}

func createGRPCClient(b *testing.B) (proto.Messenger_MessageChannelClient, func() error) {
	client, _, close := NewClient("localhost:50051")
	ctx := context.Background()

	stream, err := client.MessageChannel(ctx)
	if err != nil {
		b.Fatal(err)
	}
	return stream, close
}

func cleanup() {
	// Handle graceful shutdown on interrupt or terminate signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		log.Printf("Received signal %v. Shutting down...\n", sig)
		shutdown()
		os.Exit(0)
	}()
}

func shutdown() {
	if httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v\n", err)
		}
	}

	if grpcServer != nil {
		grpcServer.GracefulStop()
	}

	wg.Wait()
	log.Println("All servers gracefully stopped")
}

func SendRESTMessage(url, message, tenantId string, httpClient *http.Client) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("no url specified %s", url)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(message))
	if err != nil {
		return nil, fmt.Errorf("failed to create send message request: %v", err)
	}
	req.Header.Set("tenantId", tenantId)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create send message request: %v", err)
	}
	jsonData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	newMsg := new(proto.Message)
	err = protojson.Unmarshal(jsonData, newMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal body: %v", err)
	}

	defer resp.Body.Close()

	return jsonData, err
}

func newMessageServer() *GrpcServer {
	return &GrpcServer{
		toClient:   make(chan *proto.Message, 100),
		fromClient: make(chan *proto.Message, 100),
	}
}

func NewServer(address string) (*grpc.Server, net.Listener, func() error) {
	grpcListener, grpcClose := createListener(address)
	grpcServer := grpc.NewServer(serverDialOptions...)

	proto.RegisterMessengerServer(grpcServer, newMessageServer())

	return grpcServer, grpcListener, grpcClose
}

func (grpcService *GrpcServer) MessageChannel(stream proto.Messenger_MessageChannelServer) error {
	go grpcService.recvHandle(stream)

	for {
		select {
		case out := <-grpcService.toClient:
			err := stream.Send(out)
			if err == io.EOF {
				log.Print("command channel EOF")
				return nil
			}
			if err != nil {
				log.Printf("exception sending outgoing command: %v", err)
				continue
			}
		case <-stream.Context().Done():
			log.Print("command channel complete")
			return nil
		}
	}
}

func (grpcService *GrpcServer) recvHandle(server proto.Messenger_MessageChannelServer) {
	for {
		cmd, err := server.Recv()
		if err != nil {
			// recommend handling error
			log.Printf("Error in recvHandle %v", err)
			return
		}
		grpcService.handleMessage(cmd)
		grpcService.fromClient <- cmd
	}
}

func (grpcService *GrpcServer) handleMessage(msg *proto.Message) {
	if msg != nil {
		switch msg.Type {
		case *proto.Message_ATTACHMENT.Enum():
			log.Printf("Got attachment message from Agent %v", msg)
			grpcService.toClient <- msg
		default:
			log.Printf("unhandled message: %T", msg.Type)
		}
	}
}

func createListener(address string) (listener net.Listener, close func() error) {
	listen, err := net.Listen(PROTOCOL, address)
	if err != nil {
		panic(err)
	}
	return listen, listen.Close
}

func NewClient(address string) (proto.MessengerClient, *grpc.ClientConn, func() error) {
	conn, err := grpc.Dial(address, clientDialOptions...)
	if err != nil {
		return nil, nil, nil
	}

	client := proto.NewMessengerClient(conn)

	return client, conn, conn.Close
}
