package advanced_metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	UniqueDimensionPercentage int `envconfig:"unique_dimension_percentage" default:"10"`
	DimensionSize             int `envconfig:"dimension_size" default:"5"`
	MetricsPerMinute          int `envconfig:"metrics_per_minute" default:"1200"`

	Duration     time.Duration `envconfig:"duration" default:"10m"`
	AvrSocket    string        `envconfig:"avr_socket" default:"/tmp/bench.sock"`
	RandomSocket bool          `envconfig:"random_socket" default:"false"`
	PromPort     int           `envconfig:"prometheus_port" default:"2113"`
	SimpleTest   bool          `envconfig:"simple_benchmark" default:"false"`
}

func main() {
	mu := &sync.Mutex{}
	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		log.Fatalf("cannot process configuration: %s", err.Error())
	}
	if cfg.RandomSocket {
		cfg.AvrSocket = fmt.Sprintf("/tmp/sock%v", time.Now().Unix())
	}

	generatorOutputMetricsNumber := promauto.NewCounter(prometheus.CounterOpts{
		Name: "generator_output_metrics",
		Help: "Number of metrics outputted by the generator",
	})
	retriesNumber := promauto.NewCounter(prometheus.CounterOpts{
		Name: "generator_retries",
		Help: "Number of send retries the generator went through",
	})
	failedRetriesNumber := promauto.NewCounter(prometheus.CounterOpts{
		Name: "generator_retries_failed",
		Help: "Number of send retries the generator went through and failed",
	})

	go func() {
		mu.Lock()
		defer mu.Unlock()
		fmt.Println("exposing metrics on port", cfg.PromPort)
		http.Handle("/metrics", promhttp.Handler())
		log.Println(http.ListenAndServe(fmt.Sprintf(":%v", cfg.PromPort), nil))
	}()

	opts := []metric_gen.Option{
		metric_gen.WithSimpleSet(cfg.SimpleTest),
	}
	gen, err := metric_gen.NewGenerator(metric_gen.Config{
		UniquePercent:       cfg.UniqueDimensionPercentage,
		DimensionSize:       cfg.DimensionSize,
		MetricSetsPerMinute: cfg.MetricsPerMinute,
	}, opts...)
	if err != nil {
		log.Fatal(err)
	}

	outputChan := make(chan *metric_gen.Message, 100)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := gen.Generate(ctx, outputChan)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// wait 3 seconds to make sure socket exists
	time.Sleep(time.Second * 3)

	pusher := NewPusher(cfg.AvrSocket)
	err = pusher.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Close()

	ticker := time.NewTicker(cfg.Duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cancel()
			fmt.Println("Waiting 30 seconds to report the rest of metrics")
			time.Sleep(time.Second * 30)
			os.Exit(0)
		case msg := <-outputChan:
			sentBytes, err := pusher.PushToSocket([]byte(msg.Message), 0, time.Millisecond*333)
			if err != nil {
				retriesNumber.Add(1)
				_, err2 := pusher.PushToSocket([]byte(msg.Message), sentBytes, time.Second*2)
				if err2 != nil {
					now := time.Now()
					fmt.Printf("%s - Pusher error on retry: %s\n", now.Format("15:04:05"), err2.Error())
					failedRetriesNumber.Add(1)
					continue
				}
			}

			generatorOutputMetricsNumber.Add(float64(msg.MetricSets))
		}
	}
}
