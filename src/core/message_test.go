package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	message := NewMessage("test.topic.one", "payload")

	assert.Equal(t, "test.topic.one", message.Topic())
	assert.Equal(t, "payload", message.Data())

	assert.True(t, message.Exact("test.topic.one"))
	assert.False(t, message.Exact("test.topic.two"))

	assert.True(t, message.Match(""))
	assert.True(t, message.Match("test."))
	assert.True(t, message.Match("test.top"))
	assert.True(t, message.Match("test.topic."))

	assert.False(t, message.Match("a-test."))
	assert.False(t, message.Match("test.topics."))
	assert.False(t, message.Match("test.topic.sub"))
}
