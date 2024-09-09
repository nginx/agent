// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestInput_emit(t *testing.T) {
	input := Input{
		fileConsumer: &fileconsumer.Manager{},
		toBody: func(token []byte) any {
			grok, err := NewCompiledGrok(accessLogPattern, zap.L())
			if err != nil {
				t.Errorf("Failed to create new grok, %v", err)
				return nil
			}
			mappedResults := grok.ParseString(string(token))

			item, newNginxAccessItemError := newNginxAccessItem(mappedResults)
			if newNginxAccessItemError != nil {
				t.Errorf("Failed to cast grok map to access item, %v", newNginxAccessItemError)
				return nil
			}

			return item
		},
	}

	token := []byte(accessLogLine)

	err := input.emit(context.Background(), token, map[string]any{"attribute1": "test"})
	require.NoError(t, err)

	// nil token check
	err = input.emit(context.Background(), nil, map[string]any{"attribute1": "test"})
	require.NoError(t, err)
}
