/*
 * Copyright (C) F5 Inc. 2022
 * All rights reserved.
 *
 * No part of the software may be reproduced or transmitted in any
 * form or by any means, electronic or mechanical, for any purpose,
 * without express written permission of F5 Inc.
 */

package processor

import (
	"context"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	nap_monitoring "github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"

	"github.com/sirupsen/logrus"

	pb "github.com/nginx/agent/sdk/v2/proto/events"
)

const (
	// in seconds
	eventWaitTimeout = 5

	numWorkers = 4

	mockSigDBFile   = "./testdata/mock-sigs"
	nonexistentFile = "NonexistentFile"
)

func TestNAPWAFProcess(t *testing.T) {
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
			// Event will not be generated because violationRating = 0, so it will be ignored as a low-risk event.
			isNegative: false,
			fileExists: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			
			sigDBFile := mockSigDBFile
			if !tc.fileExists {
				sigDBFile = nonexistentFile
			}

			collect := make(chan *nap_monitoring.RawLog, 2)

			processed := make(chan *pb.Event, 2)

			log := logrus.New()
			log.SetLevel(logrus.DebugLevel)

			p, err := GetClient(&Config{
				Logger:                log.WithField("extension", "test"),
				Workers:               numWorkers,
				SigDBFile:             sigDBFile,
				SigDBFilePollInterval: 1,
			})
			if err != nil {
				t.Fatalf("Could not get a Processor Client: %s", err)
			}

			// This is to simulate the behavior of the necessary DB file
			// not existing prior to getting the processor client, but knowing
			// that later at some point the DB file will exist and we'll
			// populate the sigIdToSigName map with it in the background.
			if !tc.fileExists {
				p.sigDBFile = mockSigDBFile
			}

			// make sure the tests actually wait on wg and
			// verify that they are closed on cancel()
			wg := &sync.WaitGroup{}

			wg.Add(1)
			// Start Processor
			go p.Process(ctx, wg, collect, processed)

			// Briefly sleep so map can be reconciled before event is collected
			// and processed
			if !tc.fileExists {
				time.Sleep(2 * time.Second)
			}

			input, err := ioutil.ReadFile(tc.testFile)
			if err != nil {
				t.Fatalf("Error while reading the logfile %s: %v", tc.testFile, err)
			}

			collect <- &nap_monitoring.RawLog{Origin: nap_monitoring.NAPWAF, Logline: string(input)}

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
