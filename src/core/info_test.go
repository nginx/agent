package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfo(t *testing.T) {
	name := "plugin"
	version := "v0.0.1"
	info := NewInfo(name, version)

	assert.Equal(t, name, info.Name())
	assert.Equal(t, version, info.Version())
}
