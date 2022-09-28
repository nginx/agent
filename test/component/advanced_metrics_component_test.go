package advanced_metrics_component_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"net"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	advanced_metrics "github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/advanced-metrics"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/publisher"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/schema"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/sample"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	aggregatedDimension = "AGGR"

	frameSeparator        = ';'
	messageFieldSeparator = ' '

	dim1    = "dim1"
	dim2    = "dim2"
	metric1 = "metric1"
	metric2 = "metric2"

	aggregationPeriod time.Duration = time.Millisecond * 10
	publishingPeriod  time.Duration = aggregationPeriod * 10

	defaultCardinality = 10
)

var (
	defaultAggregatorConfig = advanced_metrics.AggregatorConfig{
		AggregationPeriod: aggregationPeriod,
		PublishingPeriod:  publishingPeriod,
	}

	defaultTableSizesLimits = advanced_metrics.TableSizesLimits{
		StagingTableMaxSize:    10,
		StagingTableThreshold:  10,
		PriorityTableMaxSize:   10,
		PriorityTableThreshold: 10,
	}
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

// TestAdvancedMetricsSimple this is simple test which presents basic logic of advanced_metrics module
func TestAdvancedMetricsSimple(t *testing.T) {
	value, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	socketLocation := fmt.Sprintf("/tmp/advanced_metrics_test_%d.sr", value)

	// Definition of the schema with two dimensions and two metrics
	// It is important to note that schema defines the layout of the message which
	// Advanced Metrics will support.
	builder := schema.NewSchemaBuilder().
		NewDimension(dim1, defaultCardinality).
		NewDimension(dim2, defaultCardinality).
		NewMetric(metric1).
		NewMetric(metric2)
	s, err := builder.Build()
	assert.NoError(t, err)

	cfg := advanced_metrics.Config{
		Address:          socketLocation,
		AggregatorConfig: defaultAggregatorConfig,
		TableSizesLimits: defaultTableSizesLimits,
	}

	ctx, cancel := context.WithCancel(context.Background())
	advanced_metrics, err := advanced_metrics.NewAdvancedMetrics(cfg, s)
	assert.NoError(t, err)
	wg := start(t, ctx, advanced_metrics, socketLocation)

	// This is definition of the message which contains two separate samples of the metrics.
	// Each message contain 4 fields and the order of the field is strictly related to defined above schema.
	message1 := [][]byte{
		toMessage("dim1Val1", "dim2Val1", 1, 2),
		toMessage("dim1Val1", "dim2Val1", 1, 3),
	}

	assertMessageSent(t, socketLocation, message1)

	// Receive expected list of metrics accumulated during publishing window.
	// Here only one MetricSet is expected as two samples send has the same value for the all dimensions so
	// dimension set for each sample is the same and Advanced Metrics module is aggregating all metrics which belongs to the same dimensions set.
	assertReceiveMetrics(t, advanced_metrics.OutChannel(), []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val1"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 2, Last: 1,
						Min: 1, Max: 1,
						Sum: 2,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 2, Last: 3,
						Min: 2, Max: 3,
						Sum: 5,
					},
				},
			},
		},
	})

	cancel()
	wg.Wait()
}

func TestAdvancedMetricsIsAbleToCollectMetricsFromMultipleAggregationPeriods(t *testing.T) {
	value, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	socketLocation := fmt.Sprintf("/tmp/advanced_metrics_test_%d.sr", value)

	builder := schema.NewSchemaBuilder().
		NewDimension(dim1, defaultCardinality).
		NewDimension(dim2, defaultCardinality).
		NewMetric(metric1).
		NewMetric(metric2)
	s, err := builder.Build()
	assert.NoError(t, err)

	cfg := advanced_metrics.Config{
		Address:          socketLocation,
		AggregatorConfig: defaultAggregatorConfig,
		TableSizesLimits: defaultTableSizesLimits,
	}

	ctx, cancel := context.WithCancel(context.Background())
	advanced_metrics, err := advanced_metrics.NewAdvancedMetrics(cfg, s)
	assert.NoError(t, err)
	wg := start(t, ctx, advanced_metrics, socketLocation)

	message1 := [][]byte{
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val2", "dim2Val2", 1, 1),
	}
	message2 := [][]byte{
		toMessage("dim1Val1", "dim2Val1", 2, 2),
		toMessage("dim1Val2", "dim2Val2", 3, 3),
	}

	assertMessageSent(t, socketLocation, message1)
	<-time.After(aggregationPeriod)
	assertMessageSent(t, socketLocation, message2)
	<-time.After(aggregationPeriod)
	assertMessageSent(t, socketLocation, message1)
	<-time.After(aggregationPeriod)
	assertMessageSent(t, socketLocation, message2)
	<-time.After(aggregationPeriod)

	assertReceiveMetrics(t, advanced_metrics.OutChannel(), []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val1"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 4, Last: 2,
						Min: 1, Max: 2,
						Sum: 6,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 4, Last: 2,
						Min: 1, Max: 2,
						Sum: 6,
					},
				},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val2"},
				{Name: dim2, Value: "dim2Val2"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 4, Last: 3,
						Min: 1, Max: 3,
						Sum: 8,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 4, Last: 3,
						Min: 1, Max: 3,
						Sum: 8,
					},
				},
			},
		},
	})

	cancel()
	wg.Wait()
}

func TestAdvancedMetricsIsAbleToCollectMetricsFromSinglePublishPeriod(t *testing.T) {
	value, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	socketLocation := fmt.Sprintf("/tmp/advanced_metrics_test_%d.sr", value)

	builder := schema.NewSchemaBuilder().
		NewDimension(dim1, defaultCardinality).
		NewDimension(dim2, defaultCardinality).
		NewMetric(metric1).
		NewMetric(metric2)
	s, err := builder.Build()
	assert.NoError(t, err)

	cfg := advanced_metrics.Config{
		Address:          socketLocation,
		AggregatorConfig: defaultAggregatorConfig,
		TableSizesLimits: defaultTableSizesLimits,
	}

	ctx, cancel := context.WithCancel(context.Background())
	advanced_metrics, err := advanced_metrics.NewAdvancedMetrics(cfg, s)
	assert.NoError(t, err)
	wg := start(t, ctx, advanced_metrics, socketLocation)

	message1 := [][]byte{
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val2", "dim2Val2", 1, 1),
		toMessage("dim1Val1", "dim2Val1", 2, 2),
		toMessage("dim1Val2", "dim2Val2", 3, 3),
	}

	assertMessageSent(t, socketLocation, message1)
	assertReceiveMetrics(t, advanced_metrics.OutChannel(), []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val1"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 2, Last: 2,
						Min: 1, Max: 2,
						Sum: 3,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 2, Last: 2,
						Min: 1, Max: 2,
						Sum: 3,
					},
				},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val2"},
				{Name: dim2, Value: "dim2Val2"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 2, Last: 3,
						Min: 1, Max: 3,
						Sum: 4,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 2, Last: 3,
						Min: 1, Max: 3,
						Sum: 4,
					},
				},
			},
		},
	})

	cancel()
	wg.Wait()
}

func TestAdvancedMetricsIsAbleToCollectMetricsFromMultiplePublishPeriod(t *testing.T) {
	value, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	socketLocation := fmt.Sprintf("/tmp/advanced_metrics_test_%d.sr", value)

	builder := schema.NewSchemaBuilder().
		NewDimension(dim1, defaultCardinality).
		NewDimension(dim2, defaultCardinality).
		NewMetric(metric1).
		NewMetric(metric2)
	s, err := builder.Build()
	assert.NoError(t, err)

	cfg := advanced_metrics.Config{
		Address:          socketLocation,
		AggregatorConfig: defaultAggregatorConfig,
		TableSizesLimits: defaultTableSizesLimits,
	}

	ctx, cancel := context.WithCancel(context.Background())
	advanced_metrics, err := advanced_metrics.NewAdvancedMetrics(cfg, s)
	assert.NoError(t, err)
	wg := start(t, ctx, advanced_metrics, socketLocation)

	message1 := [][]byte{
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val2", "dim2Val2", 1, 1),
		toMessage("dim1Val1", "dim2Val1", 2, 2),
		toMessage("dim1Val2", "dim2Val2", 3, 3),
	}

	assertMessageSent(t, socketLocation, message1)
	assertReceiveMetrics(t, advanced_metrics.OutChannel(), []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val1"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 2, Last: 2,
						Min: 1, Max: 2,
						Sum: 3,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 2, Last: 2,
						Min: 1, Max: 2,
						Sum: 3,
					},
				},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val2"},
				{Name: dim2, Value: "dim2Val2"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 2, Last: 3,
						Min: 1, Max: 3,
						Sum: 4,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 2, Last: 3,
						Min: 1, Max: 3,
						Sum: 4,
					},
				},
			},
		},
	})

	message2 := [][]byte{
		toMessage("dim1Val2", "dim2Val1", 1, 1),
		toMessage("dim1Val3", "dim2Val2", 1, 1),
		toMessage("dim1Val2", "dim2Val1", 2, 2),
		toMessage("dim1Val3", "dim2Val2", 4, 4),
	}

	assertMessageSent(t, socketLocation, message2)
	assertReceiveMetrics(t, advanced_metrics.OutChannel(), []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val2"},
				{Name: dim2, Value: "dim2Val1"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 2, Last: 2,
						Min: 1, Max: 2,
						Sum: 3,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 2, Last: 2,
						Min: 1, Max: 2,
						Sum: 3,
					},
				},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val3"},
				{Name: dim2, Value: "dim2Val2"},
			},
			Metrics: []publisher.Metric{
				{
					Name: metric1,
					Values: sample.Metric{
						Count: 2, Last: 4,
						Min: 1, Max: 4,
						Sum: 5,
					},
				},
				{
					Name: metric2,
					Values: sample.Metric{
						Count: 2, Last: 4,
						Min: 1, Max: 4,
						Sum: 5,
					},
				},
			},
		},
	})

	cancel()
	wg.Wait()
}

func TestAdvancedMetricsIsAbleToCollapseDimensionLookupTable(t *testing.T) {
	value, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	socketLocation := fmt.Sprintf("/tmp/advanced_metrics_test_%d.sr", value)

	builder := schema.NewSchemaBuilder().
		NewDimension(dim1, defaultCardinality).
		NewDimension(dim2, 4).
		NewMetric(metric1).
		NewMetric(metric2)
	s, err := builder.Build()
	assert.NoError(t, err)

	cfg := advanced_metrics.Config{
		Address:          socketLocation,
		AggregatorConfig: defaultAggregatorConfig,
		TableSizesLimits: defaultTableSizesLimits,
	}

	ctx, cancel := context.WithCancel(context.Background())
	advanced_metrics, err := advanced_metrics.NewAdvancedMetrics(cfg, s)
	assert.NoError(t, err)
	wg := start(t, ctx, advanced_metrics, socketLocation)

	message1 := [][]byte{
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val1", "dim2Val2", 1, 1),
		toMessage("dim1Val1", "dim2Val3", 1, 1),
		toMessage("dim1Val1", "dim2Val4", 1, 1),
	}

	singleMetric := sample.Metric{
		Count: 1, Last: 1,
		Min: 1, Max: 1,
		Sum: 1,
	}

	twoSingleMetricAggregated := sample.Metric{
		Count: 2, Last: 1,
		Min: 1, Max: 1,
		Sum: 2,
	}

	assertMessageSent(t, socketLocation, message1)
	assertReceiveMetrics(t, advanced_metrics.OutChannel(), []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val1"},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: singleMetric},
				{Name: metric2, Values: singleMetric},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val2"},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: singleMetric},
				{Name: metric2, Values: singleMetric},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: aggregatedDimension},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: twoSingleMetricAggregated},
				{Name: metric2, Values: twoSingleMetricAggregated},
			},
		},
	})

	cancel()
	wg.Wait()
}

func TestAdvancedMetricsIsAbleToCollapseSamplesInStagingTable(t *testing.T) {
	value, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	socketLocation := fmt.Sprintf("/tmp/advanced_metrics_test_%d.sr", value)

	builder := schema.NewSchemaBuilder().
		NewDimension(dim1, defaultCardinality).
		NewDimension(dim2, defaultCardinality, schema.WithCollapsingLevel(1)).
		NewMetric(metric1).
		NewMetric(metric2)
	s, err := builder.Build()
	assert.NoError(t, err)

	cfg := advanced_metrics.Config{
		Address:          socketLocation,
		AggregatorConfig: defaultAggregatorConfig,
		TableSizesLimits: advanced_metrics.TableSizesLimits{
			StagingTableMaxSize:    2,
			StagingTableThreshold:  1,
			PriorityTableMaxSize:   10,
			PriorityTableThreshold: 10,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	advanced_metrics, err := advanced_metrics.NewAdvancedMetrics(cfg, s)
	assert.NoError(t, err)
	wg := start(t, ctx, advanced_metrics, socketLocation)

	message1 := [][]byte{
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val1", "dim2Val2", 1, 1),
		toMessage("dim1Val1", "dim2Val3", 1, 1),
		toMessage("dim1Val1", "dim2Val4", 1, 1),
	}

	singleMetric := sample.Metric{
		Count: 1, Last: 1,
		Min: 1, Max: 1,
		Sum: 1,
	}

	twoSingleMetricAggregated := sample.Metric{
		Count: 2, Last: 1,
		Min: 1, Max: 1,
		Sum: 2,
	}

	assertMessageSent(t, socketLocation, message1)
	assertReceiveMetrics(t, advanced_metrics.OutChannel(), []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val1"},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: singleMetric},
				{Name: metric2, Values: singleMetric},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val2"},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: singleMetric},
				{Name: metric2, Values: singleMetric},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: aggregatedDimension},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: twoSingleMetricAggregated},
				{Name: metric2, Values: twoSingleMetricAggregated},
			},
		},
	})

	cancel()
	wg.Wait()
}

func TestAdvancedMetricsIsAbleToCollapseSamplesInPriorityTable(t *testing.T) {
	value, _ := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	socketLocation := fmt.Sprintf("/tmp/advanced_metrics_test_%d.sr", value)

	builder := schema.NewSchemaBuilder().
		NewDimension(dim1, defaultCardinality).
		NewDimension(dim2, defaultCardinality, schema.WithCollapsingLevel(1)).
		NewMetric(metric1).
		NewMetric(metric2)
	s, err := builder.Build()
	assert.NoError(t, err)

	cfg := advanced_metrics.Config{
		Address:          socketLocation,
		AggregatorConfig: defaultAggregatorConfig,
		TableSizesLimits: advanced_metrics.TableSizesLimits{
			StagingTableMaxSize:    10,
			StagingTableThreshold:  10,
			PriorityTableMaxSize:   3,
			PriorityTableThreshold: 2,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	advanced_metrics, err := advanced_metrics.NewAdvancedMetrics(cfg, s)
	assert.NoError(t, err)
	wg := start(t, ctx, advanced_metrics, socketLocation)

	message1 := [][]byte{
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val1", "dim2Val1", 1, 1),
		toMessage("dim1Val1", "dim2Val2", 1, 1),
		toMessage("dim1Val1", "dim2Val2", 1, 1),
		toMessage("dim1Val1", "dim2Val2", 1, 1),
		toMessage("dim1Val1", "dim2Val3", 1, 1),
		toMessage("dim1Val1", "dim2Val4", 1, 1),
	}

	fourSingleMetric := sample.Metric{
		Count: 4, Last: 1,
		Min: 1, Max: 1,
		Sum: 4,
	}

	threeSingleMetric := sample.Metric{
		Count: 3, Last: 1,
		Min: 1, Max: 1,
		Sum: 3,
	}

	twoSingleMetricAggregated := sample.Metric{
		Count: 2, Last: 1,
		Min: 1, Max: 1,
		Sum: 2,
	}

	assertMessageSent(t, socketLocation, message1)
	assertReceiveMetrics(t, advanced_metrics.OutChannel(), []*publisher.MetricSet{
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val1"},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: fourSingleMetric},
				{Name: metric2, Values: fourSingleMetric},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: "dim2Val2"},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: threeSingleMetric},
				{Name: metric2, Values: threeSingleMetric},
			},
		},
		{
			Dimensions: []publisher.Dimension{
				{Name: dim1, Value: "dim1Val1"},
				{Name: dim2, Value: aggregatedDimension},
			},
			Metrics: []publisher.Metric{
				{Name: metric1, Values: twoSingleMetricAggregated},
				{Name: metric2, Values: twoSingleMetricAggregated},
			},
		},
	})

	cancel()
	wg.Wait()
}

func assertReceiveMetrics(t *testing.T, outChannel chan []*publisher.MetricSet, expectedMetrics []*publisher.MetricSet) []*publisher.MetricSet {
	receivedMessages := make([]*publisher.MetricSet, 0)
	assert.Eventually(t, func() bool {
	r_loop:
		for {
			select {
			case f := <-outChannel:
				receivedMessages = append(receivedMessages, f...)
			default:
				break r_loop
			}
		}
		return len(expectedMetrics) == len(receivedMessages)
	}, time.Second, time.Microsecond*10)

	assert.ElementsMatch(t, expectedMetrics, receivedMessages)

	return receivedMessages
}

func start(t *testing.T, ctx context.Context, advanced_metrics *advanced_metrics.AdvancedMetrics, socketLocation string) *sync.WaitGroup {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := advanced_metrics.Run(ctx)
		assert.NoError(t, err)
	}()

	assert.Eventually(t, func() bool {
		_, err := os.Stat(socketLocation)
		return err == nil
	}, time.Second*2, time.Microsecond*1, "fail to change socket file permission")

	return &wg
}

func toMessage(data ...interface{}) []byte {
	msg := make([]byte, 0)
	for i, d := range data {
		if i != 0 {
			msg = append(msg, messageFieldSeparator)
		}
		if d == nil {
			continue
		}
		switch val := d.(type) {
		case string:
			msg = append(msg, '"')
			msg = append(msg, []byte(val)...)
			msg = append(msg, '"')
		case int:
			msg = append(msg, []byte(strconv.Itoa(val))...)
		}
	}
	return msg
}

func assertMessageSent(t *testing.T, addr string, dataToSend [][]byte) {
	numberOfRetries := 3
	conn, err := net.Dial("unix", addr)
	
	// if net.Dial fails connection will be retried 3 times
	for i := 0; i <= numberOfRetries && err != nil; i++ {
		time.Sleep(5 * time.Millisecond)
		conn, err = net.Dial("unix", addr)
	}

	assert.NoError(t, err)

	for _, data := range dataToSend {
		n, err := conn.Write(data)
		assert.NoError(t, err)
		assert.Equal(t, n, len(data))

		n, err = conn.Write([]byte{frameSeparator})
		assert.NoError(t, err)
		assert.Equal(t, n, 1)
	}

	conn.Close()
}
