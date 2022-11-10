package processor

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	pb "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
)

const (
	// in seconds
	eventWaitTimeout = 5
	numWorkers       = 4
)

func TestNAPProcess(t *testing.T) {
	testCases := []struct {
		testName   string
		testFile   string
		expected   *pb.Event
		isNegative bool
		fileExists bool
	}{
		{
			testName: "PassedEvent",
			testFile: "./testdata/expanded_nap_waf.log.txt",
			expected: nil,
		},
		// XML Parsing
		{
			testName: "violation name parsing",
			testFile: "./testdata/xml_violation_name.log.txt",
			expected: nil,
		},
		{
			testName: "parameter data parsing",
			testFile: "./testdata/xml_parameter_data.log.txt",
			expected: nil,
		},
		{
			testName: "parameter data parsing with empty context key",
			testFile: "./testdata/xml_parameter_data_empty_context.log.txt",
			expected: nil,
		},
		{
			testName: "parameter data parsing as param_data",
			testFile: "./testdata/xml_parameter_data_as_param_data.log.txt",
			expected: nil,
		},
		{
			testName: "header data parsing",
			testFile: "./testdata/xml_header_data.log.txt",
			expected: nil,
		},
		{
			testName: "signature data parsing",
			testFile: "./testdata/xml_signature_data.log.txt",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			collect := make(chan *monitoring.RawLog, 2)
			processed := make(chan *pb.Event, 2)

			log := logrus.New()
			log.SetLevel(logrus.DebugLevel)

			p, err := GetClient(&Config{
				Logger:  log.WithField("extension", "test"),
				Workers: numWorkers,
			})
			if err != nil {
				t.Fatalf("Could not get a Processor Client: %s", err)
			}

			wg := &sync.WaitGroup{}

			wg.Add(1)
			// Start Processor
			go p.Process(ctx, wg, collect, processed)

			// Briefly sleep so map can be reconciled before event is collected
			// and processed
			if !tc.fileExists {
				time.Sleep(2 * time.Second)
			}

			input, err := os.ReadFile(tc.testFile)
			if err != nil {
				t.Fatalf("Error while reading the logfile %s: %v", tc.testFile, err)
			}

			collect <- &monitoring.RawLog{Origin: monitoring.NAP, Logline: string(input)}

			select {
			case event := <-processed:
				t.Logf("Got event: %v", event)
			case <-time.After(eventWaitTimeout * time.Second):
				// for negative test, there should not be an event generated.
				if !tc.isNegative {
					t.Error("Should receive security violation event, and should not be timeout.")
				}
			}
		})
	}
}
