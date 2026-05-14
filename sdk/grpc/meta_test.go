/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package grpc

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetMetaForTest() {
	meta.ClientId = ""
	meta.CloudAccountId = ""
}

func TestInitMeta_PopulatesGlobal(t *testing.T) {
	resetMetaForTest()
	t.Cleanup(resetMetaForTest)

	InitMeta("client-1", "cloud-1")

	assert.Equal(t, "client-1", meta.ClientId)
	assert.Equal(t, "cloud-1", meta.CloudAccountId)
}

func TestInitMeta_OverwritesPreviousValues(t *testing.T) {
	resetMetaForTest()
	t.Cleanup(resetMetaForTest)

	InitMeta("first", "cloud-first")
	InitMeta("second", "cloud-second")

	assert.Equal(t, "second", meta.ClientId)
	assert.Equal(t, "cloud-second", meta.CloudAccountId)
}

func TestNewMessageMeta_UsesGlobalMeta(t *testing.T) {
	resetMetaForTest()
	t.Cleanup(resetMetaForTest)

	InitMeta("global-client", "global-cloud")
	m := NewMessageMeta("msg-123")

	require.NotNil(t, m)
	assert.Equal(t, "msg-123", m.MessageId)
	assert.Equal(t, "global-client", m.ClientId)
	assert.Equal(t, "global-cloud", m.CloudAccountId)
	require.NotNil(t, m.Timestamp)
}

func TestNewMessageMeta_TimestampIsRecent(t *testing.T) {
	resetMetaForTest()
	t.Cleanup(resetMetaForTest)

	before := time.Now().Add(-1 * time.Second)
	m := NewMessageMeta("msg-time")
	after := time.Now().Add(1 * time.Second)

	require.NotNil(t, m.Timestamp)
	ts := time.Unix(m.Timestamp.Seconds, int64(m.Timestamp.Nanos))
	assert.True(t, ts.After(before), "timestamp should be after %v, got %v", before, ts)
	assert.True(t, ts.Before(after), "timestamp should be before %v, got %v", after, ts)
}

func TestNewMessageMeta_BeforeInitMeta(t *testing.T) {
	resetMetaForTest()
	t.Cleanup(resetMetaForTest)

	m := NewMessageMeta("msg-no-init")
	require.NotNil(t, m)
	assert.Empty(t, m.ClientId)
	assert.Empty(t, m.CloudAccountId)
	assert.Equal(t, "msg-no-init", m.MessageId)
}

func TestNewMeta_SetsAllFields(t *testing.T) {
	m := NewMeta("client-x", "msg-x", "cloud-x")

	require.NotNil(t, m)
	assert.Equal(t, "client-x", m.ClientId)
	assert.Equal(t, "msg-x", m.MessageId)
	assert.Equal(t, "cloud-x", m.CloudAccountId)
	require.NotNil(t, m.Timestamp)
}

func TestNewMeta_DoesNotMutateGlobal(t *testing.T) {
	resetMetaForTest()
	t.Cleanup(resetMetaForTest)

	InitMeta("global", "global-cloud")
	_ = NewMeta("other-client", "msg", "other-cloud")

	assert.Equal(t, "global", meta.ClientId)
	assert.Equal(t, "global-cloud", meta.CloudAccountId)
}

func TestNewMeta_EmptyArgs(t *testing.T) {
	m := NewMeta("", "", "")
	require.NotNil(t, m)
	assert.Empty(t, m.ClientId)
	assert.Empty(t, m.MessageId)
	assert.Empty(t, m.CloudAccountId)
	assert.NotNil(t, m.Timestamp)
}

func TestNewMessageMeta_ConcurrentReads(t *testing.T) {
	resetMetaForTest()
	t.Cleanup(resetMetaForTest)

	InitMeta("concurrent-client", "concurrent-cloud")

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			m := NewMessageMeta("msg")
			assert.Equal(t, "concurrent-client", m.ClientId)
			assert.Equal(t, "concurrent-cloud", m.CloudAccountId)
		}()
	}
	wg.Wait()
}
