/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"strings"
)

type Payload interface{}

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
