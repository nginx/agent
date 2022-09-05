package reader

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source reader.go -destination mocks/reader_mock.go -package mocks
//go:generate mockgen -destination mocks/net_mocks.go -build_flags=--mod=mod -package mocks net Listener,Conn

const (
	networkType = "unix"
)

type ListenerConfig interface {
	Listen(ctx context.Context, network, address string) (net.Listener, error)
}

type NewWorkerConstructor = func(connection net.Conn, frameChannel chan Frame) Worker

type Worker interface {
	Run(ctx context.Context) error
}

// Frame represents single frame received from clients.
// For more details read `frame` documentations.
type Frame interface {
	Messages() [][]byte
	Release()
}

// Reader exposes unix socket and reads the messages send by clients and forwards it further to the 'Frame' channel.
//
// Reader implements very simple separator based protocol in order to receive multiple messages on single UNIX stream connection.
// Protocol is specified as series of following:
// <message_data><separator>
//
// Where:
// - <separator> is message separator character: `;`
// - <message_data> is any arbitrary data forming single message with following restrictions: it could not contain <separator> and
// 					maximal message size is 64kB - 1(for separator)
//
type Reader struct {
	listenerConfig ListenerConfig
	listener       net.Listener

	address string

	workersWaitGroup sync.WaitGroup
	newWorker        NewWorkerConstructor

	frameChannel chan Frame
}

func NewReader(address string) *Reader {
	frameChannel := make(chan Frame)
	return newReader(address, &net.ListenConfig{}, frameChannel, func(connection net.Conn, frameChannel chan Frame) Worker { return newWorker(connection, frameChannel) })
}

func newReader(address string, listenerConfig ListenerConfig, frameChannel chan Frame, newWorker NewWorkerConstructor) *Reader {
	return &Reader{
		listenerConfig: listenerConfig,

		address: address,

		newWorker: newWorker,

		frameChannel: frameChannel,
	}
}

func (r *Reader) Run(ctx context.Context) error {
	if networkType == "unix" {
		err := r.checkSocketAndCleanup()
		if err != nil {
			log.Warnf("Unable to cleanup orphaned unix socket, please remove manually before restarting. \nDetails:\n%v", err)
			return err
		}
	}
	listener, err := r.listenerConfig.Listen(ctx, networkType, r.address)
	if err != nil {
		return fmt.Errorf("failed to start advanced metrics listener")
	}
	r.listener = listener
	log.Debug("advanced metrics reader started listening")

	acceptLoopGroup, ctx := errgroup.WithContext(ctx)
	acceptLoopGroup.Go(func() error {
		return r.acceptLoop(ctx)
	})
	acceptLoopGroup.Go(func() error {
		return r.closeListener(ctx)
	})

	err = acceptLoopGroup.Wait()
	r.workersWaitGroup.Wait()
	close(r.frameChannel)

	return err
}

func (r *Reader) OutChannel() chan Frame {
	return r.frameChannel
}

func (r *Reader) closeListener(ctx context.Context) error {
	<-ctx.Done()
	err := r.listener.Close()
	if err != nil {
		return fmt.Errorf("fail to close listener: %w", err)
	}
	return nil
}

func (r *Reader) acceptLoop(ctx context.Context) error {
	id := 0
	for {
		connection, err := r.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("fail to accept new connection: %w", err)
		}
		r.runWorker(ctx, connection, id)
		id++
	}
}

func (r *Reader) runWorker(ctx context.Context, connection net.Conn, id int) {
	log.Debugf("New connection accepted, starting new reader worker ID: %d", id)

	worker := r.newWorker(connection, r.frameChannel)
	r.workersWaitGroup.Add(1)
	go func() {
		err := worker.Run(ctx)
		if err != nil {
			log.Error("Reader worker failed")
		}
		r.workersWaitGroup.Done()
		log.Debugf("Reader worker stopped ID: %d", id)
	}()
}

func (r *Reader) checkSocketAndCleanup() error {
	log.Debugf("Checking availability of unix socket: %s", r.address)

	if _, err := os.Stat(r.address); err == nil {
		err = os.Remove(r.address)
		if err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		return nil
	} else {
		return err
	}
	return nil
}
