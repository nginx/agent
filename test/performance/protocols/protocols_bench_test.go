package protocols

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	// "net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nginx/agent/v3/test/performance/protocols/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
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
)

func BenchmarkREST(b *testing.B) {
	startRESTServer()
	tenantId, baseURL, httpClient := createClient()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		runRESTClient(b, tenantId, baseURL, httpClient)
	}
	cleanup()
}

func BenchmarkGRPC(b *testing.B) {
	startGRPCServer()
	time.Sleep(1 * time.Second)

	b.ResetTimer()
	runGRPCClient(b)
	cleanup()
}

func getInstances(c *gin.Context) {
	protoMsg = &proto.Message{
		Timestamp:     "1706287128",
		Correlationid: "9e2d49c9-ada2-4ed1-b6f2-391d5cf634d9",
		Tenantid:      "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
		Type:          proto.Message_ATTACHMENT,
		Union:         &proto.Message_Attachement{
			Attachement: &proto.Attachment{
				Instances: []*proto.Instances{
					{
						Instance: &proto.Instance{
							InstanceId:    "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c",
							Version:       "1.25.1",
							Meta:          &proto.Instance_NginxMeta{
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
	r := gin.Default()
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
	}()
}

func startGRPCServer() {
	grpcServer, listener, close := NewServer("localhost:50051")
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal(err)
		}

		close()
	}()
}

func runRESTClient(b *testing.B,  tenantId, baseURL string, client http.Client) {
	// Simulate a REST API request
	response, err := SendRESTMessage(fmt.Sprintf("%s/instances/", baseURL), msg, tenantId, &client)
	if err != nil {
		b.Fatal(err)
	}
	assert.NotNil(b, response)
}

func createClient() (string, string, http.Client) {
	tenantId := "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"
	baseURL := "http://localhost:8080"
	httpClient := http.Client{
		Timeout: time.Second * 10,
	}
	return tenantId, baseURL, httpClient
}

func runGRPCClient(b *testing.B) {
	client, _, close := NewClient("localhost:50051")
	ctx := context.TODO()
	for i := 0; i < b.N; i++ {
		stream, err := client.MessageChannel(ctx)
		if err != nil {
			b.Fatal(err)
		}

		// make(chan struct{})
		go func() {
			for {
				in, err := stream.Recv()
				if err == io.EOF {
					close()
					return
				}
				if err != nil {
					log.Fatalf("Failed to receive a note : %v", err)
				}
				log.Printf("Got message %v", in.GetAttachement())
			}
		}()

		// stream.CloseSend()
		// <-waitc
	}
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
