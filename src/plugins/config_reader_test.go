/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
)

var fields = []string{
	"server.host",
	"server.grpcport",
	"log.level",
	"log.path",
	"nginx.basic_status_url",
	"nginx.bin_path",
	"nginx.plus_api_url",
	"dataplane.status.poll_interval",
	"tls.ca",
	"tls.cert",
	"tls.enable",
	"tls.key",
	"metrics.poll_interval",
	"metrics.bulk_size",
}

func TestConfigReader(t *testing.T) {
	tests := []struct {
		fileName string
		expected map[string]string
	}{
		{
			fileName: "testdata/configs/nginx-agent.conf",
			expected: map[string]string{
				"server.host":                    "127.0.0.1",
				"server.grpcport":                "10001",
				"server.command":                 "test-server-commands",
				"server.metrics":                 "test-server-metrics",
				"log.level":                      "info",
				"log.path":                       "/var/log/nginx-agent/log.txt",
				"nginx.basic_status_url":         "http://127.0.0.1:80/nginx_status",
				"nginx.bin_path":                 "/usr/sbin/nginx",
				"nginx.plus_api_url":             "http://127.0.0.1:8080/api",
				"dataplane.status.poll_interval": "1000ms",
				"tls.ca":                         "/etc/ssl/nginx-agent/ca.pem",
				"tls.cert":                       "/etc/ssl/nginx-agent/agent.crt",
				"tls.enable":                     "false",
				"tls.key":                        "/etc/ssl/nginx-agent/agent.key",
				"metrics.poll_interval":          "1000ms",
				"metrics.bulk_size":              "20",
			},
		},
		{
			fileName: "testdata/configs/missing_fields.conf",
			expected: map[string]string{
				"server.host":                    "127.0.0.1",
				"server.grpcport":                "10000",
				"server.command":                 "dataplane-manager",
				"server.metrics":                 "agent-ingest",
				"log.level":                      "info",
				"log.path":                       "/var/log/nginx-agent/log.txt",
				"nginx.basic_status_url":         "http://127.0.0.1:80/nginx_status",
				"nginx.bin_path":                 "/usr/sbin/nginx",
				"nginx.plus_api_url":             "http://127.0.0.1:8080/api",
				"dataplane.status.poll_interval": "1000ms",
				"tls.ca":                         "/etc/ssl/nginx-agent/ca.pem",
				"tls.cert":                       "/etc/ssl/nginx-agent/agent.crt",
				"tls.enable":                     "false",
				"metrics.bulk_size":              "20",
			},
		},
		{
			fileName: "testdata/configs/empty_config.conf",
			expected: map[string]string{
				"server.host":                    "",
				"server.grpcport":                "",
				"server.self":                    "",
				"log.level":                      "",
				"log.path":                       "",
				"nginx.basic_status_url":         "",
				"nginx.bin_path":                 "",
				"nginx.plus_api_url":             "",
				"dataplane.status.poll_interval": "",
				"tls.ca":                         "",
				"tls.cert":                       "",
				"tls.enable":                     "",
				"tls.key":                        "",
				"metrics.poll_interval":          "",
				"metrics.bulk_size":              "",
			},
		},
		{
			fileName: "",
			expected: map[string]string{},
		},
	}

	var msg *core.Message
	for _, test := range tests {
		config := NewConfigReader(&config.Config{ClientID: "12345"})
		messagePipe := core.NewMockMessagePipe(context.Background())
		err := messagePipe.Register(10, []core.Plugin{config}, []core.ExtensionPlugin{})
		assert.NoError(t, err)

		config.Init(messagePipe)
		for _, i := range fields {
			msg = core.NewMessage("configs.agent.", i)
			config.Process(msg)
		}
		for _, m := range messagePipe.GetProcessedMessages() {
			assert.EqualValues(t, test.expected[m.Topic()], m.Data())
		}
		messagePipe.ClearMessages()
	}
}

func TestUpdateAgentConfig(t *testing.T) {
	// Get the current config so we can correctly set a few test case variables
	_, _, _, err := tutils.CreateTestAgentConfigEnv()
	if err != nil {
		t.Fatalf(err.Error())
	}
	curConf, err := config.GetConfig("12345")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	testCases := []struct {
		testName    string
		cmd         *proto.Command
		expConfTags []string
		updatedTags bool
		msgTopics   []string
	}{
		{
			testName: "NoTagsToUpdate",
			cmd: &proto.Command{
				Data: &proto.Command_AgentConfig{
					AgentConfig: &proto.AgentConfig{
						Details: &proto.AgentDetails{
							Tags:     curConf.Tags,
							Features: curConf.Features,
						},
					},
				},
			},
			expConfTags: curConf.Tags,
			updatedTags: false,
			msgTopics:   []string{},
		},
		{
			testName: "UpdatedTagsAndExtensions",
			cmd: &proto.Command{
				Data: &proto.Command_AgentConfig{
					AgentConfig: &proto.AgentConfig{
						Details: &proto.AgentDetails{
							Tags:       []string{"new-tag1:one", "new-tag2:two"},
							Extensions: []string{"advanced-metrics", "nginx-app-protect"},
						},
					},
				},
			},
			expConfTags: []string{"new-tag1:one", "new-tag2:two"},
			updatedTags: true,
			msgTopics: []string{
				core.AgentConfigChanged,
				core.EnableExtension,
				core.EnableExtension,
			},
		},
		{
			testName: "RemoveAllTags",
			cmd: &proto.Command{
				Data: &proto.Command_AgentConfig{
					AgentConfig: &proto.AgentConfig{
						Details: &proto.AgentDetails{
							Tags: []string{},
						},
					},
				},
			},
			expConfTags: []string{},
			updatedTags: true,
			msgTopics: []string{
				core.AgentConfigChanged,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Create an agent config and initialize Viper config properties
			// based off of it, clean up when done.
			_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
			if err != nil {
				t.Fatalf(err.Error())
			}
			defer cleanupFunc()

			conf, err := config.GetConfig("12345")
			assert.NoError(t, err)
			// Setup config reader
			configReader := NewConfigReader(conf)
			messagePipe := core.SetupMockMessagePipe(t, context.Background(), []core.Plugin{configReader}, []core.ExtensionPlugin{})

			configReader.Init(messagePipe)

			// Create message that should trigger an update agent config call
			msg := core.NewMessage(core.AgentConfig, tc.cmd)
			configReader.Process(msg)

			// Get updated config
			updatedConf, err := config.GetConfig("12345")
			assert.Nil(t, err)

			// Sort tags before asserting
			sort.Strings(tc.expConfTags)
			sort.Strings(updatedConf.Tags)
			equalTags := reflect.DeepEqual(tc.expConfTags, updatedConf.Tags)

			// Check equality of tags
			assert.Equal(t, equalTags, true)

			// Check that the proper messages were sent through the message pipe
			messages := messagePipe.GetMessages()
			if len(messages) != len(tc.msgTopics) {
				t.Fatalf("expected %d messages, received %d", len(tc.msgTopics), len(messages))
			}
			for idx, msg := range messages {
				if tc.msgTopics[idx] != msg.Topic() {
					t.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), tc.msgTopics[idx])
				}
			}
		})
	}
}
