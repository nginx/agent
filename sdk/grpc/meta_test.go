/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package grpc

import (
	"testing"
	"time"

	sdk "github.com/nginx/agent/sdk/v2/proto"
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

func TestMetaVariants(t *testing.T) {
	tests := []struct {
		name          string
		setup         func()
		metaFunc      func() *sdk.Metadata
		wantClientId  string
		wantMessageId string
		wantCloudId   string
	}{
		{
			name: "NewMessageMeta uses global meta",
			setup: func() {
				resetMetaForTest()
				InitMeta("global-client", "global-cloud")
			},
			metaFunc: func() *sdk.Metadata {
				return NewMessageMeta("msg-123")
			},
			wantClientId:  "global-client",
			wantMessageId: "msg-123",
			wantCloudId:   "global-cloud",
		},
		{
			name:  "NewMeta sets all fields",
			setup: func() {},
			metaFunc: func() *sdk.Metadata {
				return NewMeta("client-x", "msg-x", "cloud-x")
			},
			wantClientId:  "client-x",
			wantMessageId: "msg-x",
			wantCloudId:   "cloud-x",
		},
		{
			name:  "NewMeta empty args",
			setup: func() {},
			metaFunc: func() *sdk.Metadata {
				return NewMeta("", "", "")
			},
			wantClientId:  "",
			wantMessageId: "",
			wantCloudId:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMetaForTest()
			t.Cleanup(resetMetaForTest)
			tt.setup()
			m := tt.metaFunc()
			require.NotNil(t, m, "meta should not be nil for test %q", tt.name)
			assert.Equal(t, tt.wantClientId, m.ClientId, "Test %q failed", tt.name)
			assert.Equal(t, tt.wantMessageId, m.MessageId, "Test %q failed", tt.name)
			assert.Equal(t, tt.wantCloudId, m.CloudAccountId, "Test %q failed", tt.name)
			assert.NotNil(t, m.Timestamp, "Test %q failed", tt.name)
		})
	}
}
