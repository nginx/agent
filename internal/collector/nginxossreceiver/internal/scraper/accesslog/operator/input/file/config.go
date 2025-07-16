// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/trivago/grok"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/model"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/fileconsumer"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
)

const operatorType = "access_log_file_input"

// Config is the configuration of a file input operator
type Config struct {
	helper.InputConfig  `mapstructure:",squash"`
	AccessLogFormat     string `mapstructure:"access_log_format"`
	fileconsumer.Config `mapstructure:",squash"`
}

func init() {
	operator.Register(operatorType, func() operator.Builder { return NewConfig() })
}

// NewConfig creates a new input config with default values
func NewConfig() *Config {
	return NewConfigWithID(operatorType)
}

// NewConfigWithID creates a new input config with default values
func NewConfigWithID(operatorID string) *Config {
	return &Config{
		InputConfig: helper.NewInputConfig(operatorID, operatorType),
		Config:      *fileconsumer.NewConfig(),
	}
}

// Build will build a file input operator from the supplied configuration
// nolint: ireturn
func (c Config) Build(set component.TelemetrySettings) (operator.Operator, error) {
	logger := set.Logger

	inputOperator, err := c.InputConfig.Build(set)
	if err != nil {
		return nil, err
	}

	compiledGrok, err := NewCompiledGrok(c.AccessLogFormat, logger)
	if err != nil {
		return nil, fmt.Errorf("grok init: %w", err)
	}

	toBody := grokParseFunction(logger, compiledGrok)

	input := &Input{
		InputOperator: inputOperator,
		toBody:        toBody,
	}

	input.fileConsumer, err = c.Config.Build(set, input.emit)
	if err != nil {
		return nil, err
	}

	return input, nil
}

func newNginxAccessItem(mappedResults map[string]string) (*model.NginxAccessItem, error) {
	res := &model.NginxAccessItem{}
	if err := mapstructure.Decode(mappedResults, res); err != nil {
		return nil, err
	}

	return res, nil
}

func grokParseFunction(logger *zap.Logger, compiledGrok *grok.CompiledGrok) toBodyFunc {
	return func(token []byte) any {
		return convertBytesToNginxAccessItem(logger, compiledGrok, token)
	}
}

func copyFunction(logger *zap.Logger, compiledGrok *grok.CompiledGrok) toBodyFunc {
	return func(token []byte) any {
		copied := make([]byte, len(token))
		copy(copied, token)

		return convertBytesToNginxAccessItem(logger, compiledGrok, copied)
	}
}

func convertBytesToNginxAccessItem(
	logger *zap.Logger,
	compiledGrok *grok.CompiledGrok,
	input []byte,
) *model.NginxAccessItem {
	mappedResults := compiledGrok.ParseString(string(input))

	item, newNginxAccessItemError := newNginxAccessItem(mappedResults)
	if newNginxAccessItemError != nil {
		logger.Error("Failed to cast grok map to access item", zap.Error(newNginxAccessItemError))
		return nil
	}

	return item
}
