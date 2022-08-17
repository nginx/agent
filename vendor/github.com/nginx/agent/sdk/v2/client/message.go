package client

import (
	"github.com/nginx/agent/sdk/v2/proto"
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

type msg struct {
	msgType MsgClassification
	cmd     *proto.Command
	metric  *proto.MetricsReport
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
	}

	return nil
}
