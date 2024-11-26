// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"fmt"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer/emit"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
)

type toBodyFunc func([]byte) any

// Input is an operator that monitors files for entries
type Input struct {
	fileConsumer *fileconsumer.Manager
	toBody       toBodyFunc
	helper.InputOperator
}

// Start will start the file monitoring process
func (i *Input) Start(persister operator.Persister) error {
	return i.fileConsumer.Start(persister)
}

// Stop will stop the file monitoring process
func (i *Input) Stop() error {
	return i.fileConsumer.Stop()
}

func (i *Input) emit(ctx context.Context, token emit.Token) error {
	if len(token.Body) == 0 {
		return nil
	}

	ent, err := i.NewEntry(i.toBody(token.Body))
	if err != nil {
		return fmt.Errorf("create entry: %w", err)
	}

	for k, v := range token.Attributes {
		if setError := ent.Set(entry.NewAttributeField(k), v); setError != nil {
			i.Logger().Error("Set attribute", zap.Error(setError))
		}
	}

	return i.Write(ctx, ent)
}
