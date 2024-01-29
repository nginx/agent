package protocols

import (
	"io"
	"log"
	"net"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/nginx/agent/v3/test/performance/protocols/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
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
