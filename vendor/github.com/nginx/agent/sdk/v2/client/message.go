/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"github.com/nginx/agent/sdk/v2/proto"
	models "github.com/nginx/agent/sdk/v2/proto/events"
)

func MessageFromCommand(cmd *proto.Command) Message {
	return &msg{
		msgType: MsgClassificationCommand,
		cmd:     cmd,
	}
}

func MessageFromMetrics(metric *proto.MetricsReport) Message {
	return &msg{
		msgType: MsgClassificationMetric,
		metric:  metric,
	}
}

func MessageFromEvents(event *models.EventReport) Message {
	return &msg{
		msgType: MsgClassificationEvent,
		event:   event,
	}
}

type msg struct {
	msgType MsgClassification
	cmd     *proto.Command
	metric  *proto.MetricsReport
	event   *models.EventReport
}

func (m *msg) Meta() *proto.Metadata {
	switch m.msgType {
	case MsgClassificationCommand:
		return m.cmd.GetMeta()
	case MsgClassificationMetric:
		return m.metric.GetMeta()
	}

	return nil
}

func (m *msg) Data() interface{} {
	switch m.msgType {
	case MsgClassificationCommand:
		return m.cmd.GetData()
	case MsgClassificationMetric:
		return m.metric.GetData()
	}

	return nil
}

func (m *msg) Type() MsgType {
	switch m.msgType {
	case MsgClassificationCommand:
		return m.cmd.GetType()
	case MsgClassificationMetric:
		return m.metric.GetType()
	}

	return nil
}

func (m *msg) Classification() MsgClassification {
	return m.msgType
}

func (m *msg) Raw() interface{} {
	switch m.msgType {
	case MsgClassificationCommand:
		return m.cmd
	case MsgClassificationMetric:
		return m.metric
	case MsgClassificationEvent:
		return m.event
	}

	return nil
}
