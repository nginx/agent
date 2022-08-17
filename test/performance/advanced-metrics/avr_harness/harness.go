package avr_harness

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/kelseyhightower/envconfig"
	nats_server "github.com/nats-io/nats-server/v2/server"
	nats_test "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	metrics "github.com/nginx/agent/sdk/v2/proto"
)

const (
	aggregatedValue = "Aggregated"
)

var (
	messagesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "avr_processed_total",
		Help: "The total number of processed messages",
	})
	metricsProcessedOnOutput = promauto.NewCounter(prometheus.CounterOpts{
		Name: "avr_processed_dimension_set",
		Help: "The total number of processed entities",
	})
	aggregatedDimensionValuesProcessedOnOutput = promauto.NewCounter(prometheus.CounterOpts{
		Name: "avr_discovered_aggregated_dimension_values",
		Help: "Number of processed dimensions with AGGR value",
	})
)

const (
	NatsSubj = "controller.avrd.metrics"
)

type Config struct {
	NATSPort  int    `envconfig:"nats_port" default:"4222"`
	AvrSocket string `envconfig:"avr_socket" default:"/tmp/bench.sock"`
	PromPort  int    `envconfig:"prometheus_port" default:"2112"`
}

func main() {
	mu := &sync.Mutex{}
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		log.Fatalf("cannot process configuration: %s", err.Error())
	}

	go func() {
		mu.Lock()
		defer mu.Unlock()
		fmt.Println("exposing metrics on port", cfg.PromPort)
		http.Handle("/metrics", promhttp.Handler())
		log.Println(http.ListenAndServe(fmt.Sprintf(":%v", cfg.PromPort), nil))
	}()

	opts := nats_test.DefaultTestOptions
	opts.Port = cfg.NATSPort
	opts.Host = "0.0.0.0"
	nServ := nats_test.RunServer(&opts)

	h := Harness{
		natsServer: nServ,
		natsSubs:   map[string]*NatsConn{},
	}
	h.SetupNATSSubscriber(NatsSubj, fmt.Sprintf("nats://%v:%v", opts.Host, opts.Port))

	wg := sync.WaitGroup{}
	wg.Add(1)

	fmt.Println("Nats Subscriber OK")
	for _, s := range h.natsSubs {
		go func() {
			for {
				msg := <-s.Stats
				messagesProcessed.Inc()
				if msg.Stats == nil {
					continue
				}

				metricsProcessedOnOutput.Add(float64(len(msg.Stats.Metrics)))
				for _, d := range msg.Stats.Dimensions {
					if d.Value == aggregatedValue {
						aggregatedDimensionValuesProcessedOnOutput.Inc()
					}
				}
			}
		}()
	}
	wg.Wait()
}

type Harness struct {
	natsServer *nats_server.Server
	natsSubs   map[string]*NatsConn
}

// Stats is a container, so we can pass through the channel
type Stats struct {
	Stats   *metrics.StatsEntity
	Subject *string
	Size    *int
}

type NatsConn struct {
	Nats  *nats.Conn
	Stats chan *Stats
}

func (h *Harness) SetupNATSSubscriber(subj string, addr string) *NatsConn {
	if n, ok := h.natsSubs[subj]; ok {
		return n
	}
	nc, err := nats.Connect(addr)
	if err != nil {
		log.Fatalf("NATS client connect failed(err:%v)", err)
	}
	nc.SetDisconnectErrHandler(natsDisconnectHandler)
	out := make(chan *Stats)
	_, err = nc.Subscribe(subj, func(m *nats.Msg) {
		size := len(m.Data)
		avrstat := &metrics.StatsEntity{}
		err := proto.Unmarshal(m.Data, avrstat)
		if err != nil {
			log.Fatalf("Error proto.Unmarshal: %v, Payload (%d bytes): %q", err, size, string(m.Data))
		} else {
			out <- &Stats{Stats: avrstat, Subject: &subj, Size: &size}
		}
	})
	if err != nil {
		log.Fatalf("NATS client subscription failed(err:%v)", err)
	}
	n := &NatsConn{
		Nats: nc, Stats: out,
	}
	h.natsSubs[subj] = n
	return n
}

func natsDisconnectHandler(conn *nats.Conn, err error) {
	log.Printf("NATS client disconnected(err:%v)!!!", err)
}
