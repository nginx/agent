/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ingester

import (
	"context"
	"sync"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/reader"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables"
	log "github.com/sirupsen/logrus"
)

//go:generate go run go.uber.org/mock/mockgen -source ingester.go -destination mocks/ingester_mock.go -package mocks -copyright_file=../../../../COPYRIGHT
const workers = 3

type StagingTable interface {
	Add(tables.FieldIterator) error
}

// Ingester is responsible for receiving frames and storing it in the staging table.
// Ingester decodes received message and implements field iterator which is responsible
// for splitting message in multiple fields.
type Ingester struct {
	stagingTable  StagingTable
	framesChannel <-chan reader.Frame
}

func NewIngester(framesChannel <-chan reader.Frame, stagingTable StagingTable) *Ingester {
	return &Ingester{
		stagingTable:  stagingTable,
		framesChannel: framesChannel,
	}
}

func (i *Ingester) Run(ctx context.Context) {
	wg := sync.WaitGroup{}

	wg.Add(workers)
	for j := 0; j < workers; j++ {
		go func() {
			defer wg.Done()
			i.parseFrames(ctx)
		}()
	}
	wg.Wait()
}

func (i *Ingester) parseFrames(ctx context.Context) {
	log.Info("Ingester worker starts processing data.")
	for {
		select {
		case frame, ok := <-i.framesChannel:
			if !ok {
				log.Info("Frames channel closed, ingester worker stops processing data.")
				return
			}
			for _, msg := range frame.Messages() {
				err := i.stagingTable.Add(newMessageFieldIterator(msg))
				if err != nil {
					log.Warnf("Fail to process incoming metric '%s': %s", string(msg), err.Error())
				}
			}
			frame.Release()
		case <-ctx.Done():
			log.Info("Context canceled, ingester worker stops processing data.")
			return
		}
	}
}
