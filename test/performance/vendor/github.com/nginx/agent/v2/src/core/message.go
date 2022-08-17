package core

import (
	"strings"
)

type Payload interface {
}

type Message struct {
	topic *string
	data  *Payload
}

func NewMessage(topic string, data Payload) *Message {
	message := new(Message)
	message.topic = &topic
	message.data = &data
	return message
}

func (m *Message) Match(topic string) bool {
	return strings.HasPrefix(*m.topic, topic)
}

func (m *Message) Exact(topic string) bool {
	return *m.topic == topic
}

func (m *Message) Topic() string {
	return *m.topic
}

func (m *Message) Data() Payload {
	return *m.data
}
