/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/sdk/v2"
	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/checksum"
	"github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"
	sdk_zip "github.com/nginx/agent/sdk/v2/zip"
	"github.com/nginx/agent/v2/src/core"
	loadedConfig "github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/payloads"
	tutils "github.com/nginx/agent/v2/test/utils"
)

var (
	first  = []byte("nginx conf contents")
	second = []byte("")
	third  = []byte(`user       www www;  ## Default: nobody
	worker_processes  5;  ## Default: 1
	error_log  logs/error.log;
	pid        logs/nginx.pid;
	worker_rlimit_nofile 8192;

	events {
	worker_connections  4096;  ## Default: 1024
	}
	
	http {
	include    conf/mime.types;
	include    /etc/nginx/proxy.conf;
	include    /etc/nginx/fastcgi.conf;
	index    index.html index.htm index.php;

	default_type application/octet-stream;
	log_format   main '$remote_addr - $remote_user [$time_local]  $status '
		'\"$request\" $body_bytes_sent \"$http_referer\" '
		'\"$http_user_agent\" \"$http_x_forwarded_for\"';
	access_log   logs/access.log  main;
	sendfile     on;
	tcp_nopush   on;
	server_names_hash_bucket_size 128; # this seems to be required for some vhosts
	
	server { # php/fastcgi
		listen       80;
		server_name  domain1.com www.domain1.com;
		access_log   logs/domain1.access.log  main;
		root         html;

		location ~ \.php$ {
		fastcgi_pass   127.0.0.1:1025;
		}
	}`)
	fourth = []byte(`daemon            off;
	worker_processes  2;
	user              www-data;
	
	events {
		use           epoll;
		worker_connections  128;
	}
	
	error_log         logs/error.log info;
	
	http {
		server_tokens off;
		charset       utf-8;
	
		access_log    logs/access.log  combined;
	
		server {
			server_name   localhost;
			listen        127.0.0.1:80;
	
			error_page    500 502 503 504;
	
			location      / {
				root      html;
			}
		}
	}`)
	wafMetaData1 = []byte(`{
		"napVersion": "3.1088.2",
		"globalStateFileName": "",
		"globalStateFileUID": "",
		"attackSignatureRevisionTimestamp ": "2021.04.04",
		"threatCampaignRevisionTimestamp": "2021.04.16",
		"policyMetadata": [
		  {
			"name": "default-enforcemen t3",
			"uid": "d102e132-12a0-483d-8329-d365d29801e0",
			"revisionTimestamp": 1669071164488
		  },
		  {
			"name": "ignore-xss3",
			" uid": "de178e35-dc3b-40da-a53e-2bcede19e072",
			"revisionTimestamp": 1668723801915
		  }
		],
		"logProfileMetadata": [
		  {
			"nam e": "log_all",
			"uid": "ee07fd58-fbd2-4db9-a2c2-0d06dc4d4321",
			"revisionTimestamp": 1668723322517
		  }
		]
	  }`)
)

func TestNginxConfigApply(t *testing.T) {
	validationTimeout = 100 * time.Millisecond
	updatedProcesses := []core.Process{
		{Pid: 1, Name: "12345", IsMaster: true},
		{Pid: 4, ParentPid: 1, Name: "worker-4", IsMaster: false},
		{Pid: 5, ParentPid: 1, Name: "worker-5", IsMaster: false},
	}

	t.Parallel()

	tests := []struct {
		config     *proto.NginxConfig
		msgTopics  []string
		wafVersion string
	}{
		{
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_APPLY,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      first,
					Checksum:      checksum.Checksum(first),
					RootDirectory: "nginx.conf",
				},
				Zaux:         &proto.ZippedFile{},
				AccessLogs:   &proto.AccessLogs{},
				ErrorLogs:    &proto.ErrorLogs{},
				Ssl:          &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{},
			},
			msgTopics: []string{
				core.CommNginxConfig,
				core.NginxPluginConfigured,
				core.NginxInstancesFound,
				core.NginxConfigValidationPending,
				core.FileWatcherEnabled,
				core.NginxConfigValidationSucceeded,
				core.CommResponse,
				core.NginxReloadComplete,
				core.CommResponse,
				core.FileWatcherEnabled,
				core.NginxConfigApplySucceeded,
			},
			wafVersion: "",
		},
		{
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_APPLY,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      second,
					Checksum:      checksum.Checksum(second),
					RootDirectory: "nginx.conf",
				},
				Zaux:         &proto.ZippedFile{},
				AccessLogs:   &proto.AccessLogs{},
				ErrorLogs:    &proto.ErrorLogs{},
				Ssl:          &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{},
			},
			msgTopics: []string{
				core.CommNginxConfig,
				core.NginxPluginConfigured,
				core.NginxInstancesFound,
				core.NginxConfigValidationPending,
				core.FileWatcherEnabled,
				core.NginxConfigValidationSucceeded,
				core.CommResponse,
				core.NginxReloadComplete,
				core.CommResponse,
				core.FileWatcherEnabled,
				core.NginxConfigApplySucceeded,
			},
			wafVersion: "",
		},
		{
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_APPLY,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      third,
					Checksum:      checksum.Checksum(third),
					RootDirectory: "nginx.conf",
				},
				Zaux:         &proto.ZippedFile{},
				AccessLogs:   &proto.AccessLogs{},
				ErrorLogs:    &proto.ErrorLogs{},
				Ssl:          &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{},
			},
			msgTopics: []string{
				core.CommNginxConfig,
				core.NginxPluginConfigured,
				core.NginxInstancesFound,
				core.NginxConfigValidationPending,
				core.FileWatcherEnabled,
				core.NginxConfigValidationSucceeded,
				core.CommResponse,
				core.NginxReloadComplete,
				core.CommResponse,
				core.FileWatcherEnabled,
				core.NginxConfigApplySucceeded,
			},
			wafVersion: "",
		},
		{
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_APPLY,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      third,
					Checksum:      checksum.Checksum(third),
					RootDirectory: "nginx.conf",
				},
				Zaux: &proto.ZippedFile{
					Contents:      wafMetaData1,
					Checksum:      "e7658d44b84512b4047385ed1d5c842ddfae87386f0ad1abae9e3360b4d11a65",
					RootDirectory: "/etc/nms",
				},
				AccessLogs: &proto.AccessLogs{},
				ErrorLogs:  &proto.ErrorLogs{},
				Ssl:        &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{
					Directories: []*proto.Directory{
						{
							Name:        "/etc/nms",
							Permissions: "0755",
							Files: []*proto.File{
								{
									Name:        "app_protect_metadata.json",
									Permissions: "0644",
									Size_:       959,
									Contents:    wafMetaData1,
								},
							},
							Size_: 128,
						},
					},
				},
			},
			msgTopics: []string{
				core.CommNginxConfig,
				core.NginxPluginConfigured,
				core.NginxInstancesFound,
				core.NginxConfigValidationPending,
				core.FileWatcherEnabled,
				core.NginxConfigValidationSucceeded,
				core.CommResponse,
				core.NginxReloadComplete,
				core.CommResponse,
				core.FileWatcherEnabled,
				core.NginxConfigApplySucceeded,
			},
			wafVersion: "3.1088.2",
		},
		{
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_APPLY,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      third,
					Checksum:      checksum.Checksum(third),
					RootDirectory: "nginx.conf",
				},
				Zaux: &proto.ZippedFile{
					Contents:      wafMetaData1,
					Checksum:      checksum.Checksum(wafMetaData1),
					RootDirectory: "/etc/nms",
				},
				AccessLogs:   &proto.AccessLogs{},
				ErrorLogs:    &proto.ErrorLogs{},
				Ssl:          &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{},
			},
			msgTopics: []string{
				core.CommNginxConfig,
				core.NginxPluginConfigured,
				core.NginxInstancesFound,
				core.NginxConfigValidationPending,
				core.FileWatcherEnabled,
				core.NginxConfigValidationSucceeded,
				core.CommResponse,
				core.NginxReloadComplete,
				core.CommResponse,
				core.FileWatcherEnabled,
				core.NginxConfigApplySucceeded,
			},
			wafVersion: "",
		},
		{
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_FORCE,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      first,
					Checksum:      checksum.Checksum(first),
					RootDirectory: "nginx.conf",
				},
				Zaux:         &proto.ZippedFile{},
				AccessLogs:   &proto.AccessLogs{},
				ErrorLogs:    &proto.ErrorLogs{},
				Ssl:          &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{},
			},
			msgTopics: []string{
				core.CommNginxConfig,
				core.NginxPluginConfigured,
				core.NginxInstancesFound,
				core.NginxConfigValidationPending,
				core.FileWatcherEnabled,
				core.NginxConfigValidationSucceeded,
				core.CommResponse,
				core.NginxReloadComplete,
				core.CommResponse,
				core.FileWatcherEnabled,
				core.NginxConfigApplySucceeded,
			},
			wafVersion: "",
		},
		{
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_FORCE,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      first,
					Checksum:      checksum.Checksum(first),
					RootDirectory: "nginx.conf",
				},
				Zaux: &proto.ZippedFile{
					Contents:      wafMetaData1,
					Checksum:      "e7658d44b84512b4047385ed1d5c842ddfae87386f0ad1abae9e3360b4d11a65",
					RootDirectory: "/etc/nms",
				},
				AccessLogs: &proto.AccessLogs{},
				ErrorLogs:  &proto.ErrorLogs{},
				Ssl:        &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{
					Directories: []*proto.Directory{
						{
							Name:        "/etc/nms",
							Permissions: "0755",
							Files: []*proto.File{
								{
									Name:        "app_protect_metadata.json",
									Permissions: "0644",
									Size_:       959,
									Contents:    wafMetaData1,
								},
							},
							Size_: 128,
						},
					},
				},
			},
			msgTopics: []string{
				core.CommNginxConfig,
				core.NginxPluginConfigured,
				core.NginxInstancesFound,
				core.NginxConfigValidationPending,
				core.FileWatcherEnabled,
				core.NginxConfigValidationSucceeded,
				core.CommResponse,
				core.NginxReloadComplete,
				core.CommResponse,
				core.FileWatcherEnabled,
				core.NginxConfigApplySucceeded,
			},
			// mismatch, test should still pass because of the NginxConfigAction_FORCE
			wafVersion: "3.1088.1",
		},
		{
			config: &proto.NginxConfig{
				Action: proto.NginxConfigAction_APPLY,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
				Zconfig: &proto.ZippedFile{
					Contents:      first,
					Checksum:      checksum.Checksum(first),
					RootDirectory: "nginx.conf",
				},
				Zaux: &proto.ZippedFile{
					Contents:      wafMetaData1,
					Checksum:      "e7658d44b84512b4047385ed1d5c842ddfae87386f0ad1abae9e3360b4d11a65",
					RootDirectory: "/etc/nms",
				},
				AccessLogs: &proto.AccessLogs{},
				ErrorLogs:  &proto.ErrorLogs{},
				Ssl:        &proto.SslCertificates{},
				DirectoryMap: &proto.DirectoryMap{
					Directories: []*proto.Directory{
						{
							Name:        "/etc/nms",
							Permissions: "0755",
							Files: []*proto.File{
								{
									Name:        "app_protect_metadata.json",
									Permissions: "0644",
									Size_:       959,
									Contents:    wafMetaData1,
								},
							},
							Size_: 128,
						},
					},
				},
			},
			msgTopics: []string{
				core.CommNginxConfig,
				core.NginxPluginConfigured,
				core.NginxInstancesFound,
				core.NginxConfigValidationPending,
				core.CommResponse,
			},
			// mismatch, should fail on preflight because of the NginxConfigAction_APPLY
			wafVersion: "3.1088.1",
		},
	}

	for idx, test := range tests {
		test := test
		cmd := &proto.Command{
			Meta: grpc.NewMessageMeta(uuid.New().String()),
			Type: 1,
			Data: &proto.Command_NginxConfig{
				NginxConfig: &proto.NginxConfig{
					Action: test.config.GetAction(),
					ConfigData: &proto.ConfigDescriptor{
						NginxId:  "12345",
						Checksum: "test",
					},
				},
			},
		}

		t.Run(fmt.Sprintf("%d", idx), func(tt *testing.T) {
			dir := t.TempDir()
			var auxPath string
			tempConf, err := os.CreateTemp(dir, "nginx.conf")
			assert.NoError(t, err)

			err = os.WriteFile(tempConf.Name(), fourth, 0644)
			assert.NoError(t, err)

			if (test.config.GetZaux() != &proto.ZippedFile{} && len(test.config.GetZaux().GetContents()) > 0) {
				auxDir := t.TempDir()
				auxMainFile := fmt.Sprintf("%s/app_protect_metadata.json", auxDir)
				err := os.WriteFile(auxMainFile, wafMetaData1, 0644)
				assert.NoError(t, err)
				auxPath = auxMainFile

				writer, err := sdk_zip.NewWriter(auxDir)
				assert.NoError(t, err)

				for _, directory := range test.config.GetDirectoryMap().GetDirectories() {
					for _, file := range directory.GetFiles() {
						reader := bytes.NewReader(file.GetContents())
						err = writer.Add(fmt.Sprintf("%s/%s", directory.GetName(), file.GetName()), 0644, reader)
						assert.NoError(t, err)
					}
				}

				zipFile, err := writer.Proto()
				assert.NoError(t, err)
				assert.NotEmpty(t, zipFile.Contents)

				test.config.Zaux = zipFile
			}

			ctx := context.TODO()

			env := tutils.GetMockEnvWithProcess()
			allowedDirectoriesMap := map[string]struct{}{dir: {}}

			config, err := sdk.NewConfigApply(tempConf.Name(), allowedDirectoriesMap)
			assert.NoError(t, err)

			binary := tutils.NewMockNginxBinary()
			binary.On("WriteConfig", mock.Anything).Return(config, nil)
			binary.On("ValidateConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(test.config, nil)
			binary.On("GetNginxDetailsByID", "12345").Return(tutils.GetDetailsMap()["12345"])
			binary.On("UpdateNginxDetailsFromProcesses", env.Processes())
			binary.On("GetNginxDetailsMapFromProcesses", env.Processes()).Return(tutils.GetDetailsMap())
			binary.On("Reload", mock.Anything, mock.Anything).Return(nil)
			binary.On("GetErrorLogs").Return(make(map[string]string))

			commandClient := tutils.GetMockCommandClient(test.config)
			conf := &loadedConfig.Config{
				Server: loadedConfig.Server{
					Host:     "127.0.0.1",
					GrpcPort: 9092,
				},
				Nginx: loadedConfig.Nginx{
					ConfigReloadMonitoringPeriod: 5 * time.Second,
				},
				Features:   []string{agent_config.FeatureNginxConfig},
				Extensions: []string{agent_config.NginxAppProtectExtensionPlugin},
			}

			pluginUnderTest := NewNginx(commandClient, binary, env, conf)
			if (test.config.GetZaux() != &proto.ZippedFile{} && len(test.config.GetZaux().GetContents()) > 0) {
				pluginUnderTest.nginxAppProtectSoftwareDetails = &proto.AppProtectWAFDetails{
					PrecompiledPublication: true,
					WafLocation:            auxPath,
					WafVersion:             test.wafVersion,
				}
			}

			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

			messagePipe.Process(core.NewMessage(core.CommNginxConfig, cmd))

			go messagePipe.Run()

			time.Sleep(500 * time.Millisecond)
			pluginUnderTest.syncProcessInfo(updatedProcesses)

			assert.Eventually(
				tt,
				func() bool { return len(messagePipe.GetProcessedMessages()) == len(test.msgTopics) },
				time.Duration(5*time.Second),
				3*time.Millisecond,
			)

			for idx, msg := range messagePipe.GetProcessedMessages() {
				t.Logf("%v", msg.Topic())
				if test.msgTopics[idx] != msg.Topic() {
					tt.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), test.msgTopics[idx])
				}
				// Verify that the NginxConfig Command and NginxConfigResponse Command meta messageIds match
				if msg.Topic() == core.CommResponse {
					messageIdReceived := msg.Data().(*proto.Command).Meta.MessageId
					if messageIdReceived != cmd.Meta.MessageId {
						tt.Errorf("unexpected MessageId: %s :: should have been: %s", messageIdReceived, cmd.Meta.MessageId)
					}
				}
			}
		})
	}
}

func TestUploadConfigs(t *testing.T) {
	config := &proto.NginxConfig{
		Action: proto.NginxConfigAction_APPLY,
		ConfigData: &proto.ConfigDescriptor{
			NginxId:  "12345",
			Checksum: "test",
		},
		Zconfig: &proto.ZippedFile{
			Contents:      third,
			Checksum:      checksum.Checksum(third),
			RootDirectory: "nginx-agent.conf",
		},
		Zaux:         &proto.ZippedFile{},
		AccessLogs:   &proto.AccessLogs{},
		ErrorLogs:    &proto.ErrorLogs{},
		Ssl:          &proto.SslCertificates{},
		DirectoryMap: &proto.DirectoryMap{},
	}

	msgTopics := []string{
		core.NginxPluginConfigured,
		core.NginxInstancesFound,
		core.DataplaneChanged,
		core.NginxPluginConfigured,
		core.NginxInstancesFound,
	}

	env := tutils.GetMockEnvWithProcess()

	binary := tutils.NewMockNginxBinary()
	binary.On("GetNginxDetailsByID", "12345").Return(tutils.GetDetailsMap()["12345"])
	binary.On("ReadConfig", "/var/conf", "12345", "12345678").Return(config, nil)
	binary.On("UpdateNginxDetailsFromProcesses", mock.Anything)
	binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return(tutils.GetDetailsMap())

	cmdr := tutils.NewMockCommandClient()
	cmdr.On("Upload", mock.Anything, mock.Anything).Return(nil)

	conf := &loadedConfig.Config{Server: loadedConfig.Server{Host: "127.0.0.1", GrpcPort: 9092}, Features: []string{agent_config.FeatureNginxConfig}}

	pluginUnderTest := NewNginx(cmdr, binary, env, conf)
	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

	pluginUnderTest.Init(messagePipe)
	messagePipe.Process(core.NewMessage(core.DataplaneChanged, nil))
	messagePipe.Run()

	binary.AssertExpectations(t)
	cmdr.AssertExpectations(t)
	env.AssertExpectations(t)

	core.ValidateMessages(t, messagePipe, msgTopics)
}

func TestDisableUploadConfigs(t *testing.T) {
	msgTopics := []string{
		core.NginxPluginConfigured,
		core.NginxInstancesFound,
		core.DataplaneChanged,
		core.NginxPluginConfigured,
		core.NginxInstancesFound,
	}

	env := tutils.GetMockEnvWithProcess()

	binary := tutils.NewMockNginxBinary()
	binary.On("UpdateNginxDetailsFromProcesses", mock.Anything)
	binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return(tutils.GetDetailsMap())

	cmdr := tutils.NewMockCommandClient()

	pluginUnderTest := NewNginx(cmdr, binary, env, &loadedConfig.Config{})
	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

	pluginUnderTest.Init(messagePipe)
	messagePipe.Process(core.NewMessage(core.DataplaneChanged, nil))
	messagePipe.Run()

	binary.AssertExpectations(t)
	env.AssertExpectations(t)

	core.ValidateMessages(t, messagePipe, msgTopics)
}

func TestNginxDetailProcUpdate(t *testing.T) {
	foundMessage := false
	env := tutils.GetMockEnvWithProcess()

	binary := tutils.NewMockNginxBinary()
	binary.On("GetNginxDetailsMapFromProcesses", mock.Anything).Return(tutils.GetDetailsMap())
	binary.On("UpdateNginxDetailsFromProcesses", tutils.GetProcesses())

	cmdr := tutils.NewMockCommandClient()

	pluginUnderTest := NewNginx(cmdr, binary, env, &loadedConfig.Config{})
	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})

	pluginUnderTest.Init(messagePipe)
	messagePipe.Process(core.NewMessage(core.NginxDetailProcUpdate, tutils.GetProcesses()))
	messagePipe.Run()

	binary.AssertExpectations(t)
	cmdr.AssertExpectations(t)
	env.AssertExpectations(t)

	processedMessages := messagePipe.GetProcessedMessages()

	for _, msg := range processedMessages {
		if msg.Topic() == core.NginxDetailProcUpdate {
			messageReceived := msg.Data().([]core.Process)
			assert.Equal(t, tutils.GetProcesses(), messageReceived)
			foundMessage = true
		}
	}
	assert.Len(t, processedMessages, 5)
	assert.True(t, foundMessage)
}

func TestNginx_Process_NginxConfigUpload(t *testing.T) {
	configDesc := &proto.ConfigDescriptor{
		SystemId: "12345678",
		NginxId:  "12345",
	}
	config := &proto.NginxConfig{
		Action: proto.NginxConfigAction_APPLY,
		ConfigData: &proto.ConfigDescriptor{
			NginxId:  "12345",
			Checksum: "test",
		},
		Zconfig: &proto.ZippedFile{
			Contents:      third,
			Checksum:      checksum.Checksum(third),
			RootDirectory: "nginx-agent.conf",
		},
	}
	cmdr := tutils.NewMockCommandClient()
	cmdr.On("Upload", mock.Anything, mock.Anything).Return(nil)

	binary := tutils.NewMockNginxBinary()
	binary.On("GetNginxDetailsByID", "12345").Return(tutils.GetDetailsMap()["12345"])
	binary.On("ReadConfig", "/var/conf", "12345", "12345678").Return(config, nil)

	env := tutils.GetMockEnvWithProcess()
	conf := &loadedConfig.Config{Server: loadedConfig.Server{Host: "127.0.0.1", GrpcPort: 9092}, Features: []string{agent_config.FeatureNginxConfig}}

	pluginUnderTest := NewNginx(cmdr, binary, env, conf)
	pluginUnderTest.Process(core.NewMessage(core.NginxConfigUpload, configDesc))

	binary.AssertExpectations(t)
	cmdr.AssertExpectations(t)
	env.AssertExpectations(t)

	pluginUnderTest.Close()
}

func TestNginx_Subscriptions(t *testing.T) {
	subs := []string{
		core.CommNginxConfig,
		core.NginxConfigUpload,
		core.NginxDetailProcUpdate,
		core.DataplaneChanged,
		core.DataplaneSoftwareDetailsUpdated,
		core.AgentConfigChanged,
		core.EnableExtension,
		core.NginxConfigValidationPending,
		core.NginxConfigValidationSucceeded,
		core.NginxConfigValidationFailed,
	}
	pluginUnderTest := NewNginx(nil, nil, tutils.GetMockEnvWithProcess(), &loadedConfig.Config{})

	assert.Equal(t, subs, pluginUnderTest.Subscriptions())
}

func TestNginx_Info(t *testing.T) {
	pluginUnderTest := NewNginx(nil, nil, tutils.GetMockEnvWithProcess(), &loadedConfig.Config{})

	assert.Equal(t, "NginxBinary", pluginUnderTest.Info().Name())
}

func TestNginx_validateConfig(t *testing.T) {
	tests := []struct {
		name             string
		validationResult error
		expectedTopic    string
		expectedError    error
	}{
		{
			name:             "successful validation",
			validationResult: nil,
			expectedTopic:    core.NginxConfigValidationSucceeded,
		},
		{
			name:             "failed validation",
			validationResult: errors.New("failure"),
			expectedTopic:    core.NginxConfigValidationFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {

			env := tutils.GetMockEnvWithProcess()
			binary := tutils.NewMockNginxBinary()
			binary.On("ValidateConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(test.validationResult)
			binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)
			binary.On("GetNginxDetailsMapFromProcesses", env.Processes()).Return(tutils.GetDetailsMap())
			binary.On("UpdateNginxDetailsFromProcesses", env.Processes())
			conf := &loadedConfig.Config{Server: loadedConfig.Server{Host: "127.0.0.1", GrpcPort: 9092}, Features: []string{agent_config.FeatureNginxConfig}}

			pluginUnderTest := NewNginx(&tutils.MockCommandClient{}, binary, env, conf)

			messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})
			messagePipe.Run()

			pluginUnderTest.validateConfig(&proto.NginxDetails{}, "123", &proto.NginxConfig{}, &sdk.ConfigApply{})

			assert.Eventually(
				t,
				func() bool { return len(messagePipe.GetMessages()) == 1 },
				time.Duration(2*time.Second),
				3*time.Millisecond,
			)

			assert.Equal(t, test.expectedTopic, messagePipe.GetMessages()[0].Topic())
			assert.Equal(t, "123", messagePipe.GetMessages()[0].Data().(*NginxConfigValidationResponse).correlationId)
			if test.validationResult == nil {
				assert.Nil(t, messagePipe.GetMessages()[0].Data().(*NginxConfigValidationResponse).err)
			} else {
				assert.NotNil(t, messagePipe.GetMessages()[0].Data().(*NginxConfigValidationResponse).err)
			}
			assert.Greater(t, messagePipe.GetMessages()[0].Data().(*NginxConfigValidationResponse).elapsedTime, 0*time.Second)
		})
	}
}

func TestNginx_completeConfigApply(t *testing.T) {
	expectedTopics := []string{
		core.NginxConfigValidationSucceeded,
		core.NginxPluginConfigured,
		core.NginxInstancesFound,
		core.NginxReloadComplete,
		core.CommResponse,
		core.FileWatcherEnabled,
		core.NginxConfigApplySucceeded,
	}

	env := tutils.GetMockEnvWithProcess()
	env.On("GetSystemUUID").Return("456")

	updatedProcesses := []core.Process{
		{Pid: 1, Name: "12345", IsMaster: true},
		{Pid: 4, ParentPid: 1, Name: "worker-4", IsMaster: false},
		{Pid: 5, ParentPid: 1, Name: "worker-5", IsMaster: false},
	}
	binary := tutils.NewMockNginxBinary()
	binary.On("uploadConfig", mock.Anything, mock.Anything).Return(nil)
	binary.On("GetNginxDetailsByID", "12345").Return(tutils.GetDetailsMap()["12345"])
	binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)

	binary.On("UpdateNginxDetailsFromProcesses", env.Processes()).Once()
	binary.On("GetNginxDetailsMapFromProcesses", env.Processes()).Return(tutils.GetDetailsMap()).Once()

	binary.On("Reload", mock.Anything, mock.Anything)
	binary.On("GetErrorLogs").Return(make(map[string]string))

	commandClient := tutils.GetMockCommandClient(
		&proto.NginxConfig{
			Action: proto.NginxConfigAction_APPLY,
			ConfigData: &proto.ConfigDescriptor{
				NginxId:  "12345",
				Checksum: "2314365",
			},
			Zconfig: &proto.ZippedFile{
				Contents:      first,
				Checksum:      checksum.Checksum(first),
				RootDirectory: "nginx.conf",
			},
			Zaux:         &proto.ZippedFile{},
			AccessLogs:   &proto.AccessLogs{},
			ErrorLogs:    &proto.ErrorLogs{},
			Ssl:          &proto.SslCertificates{},
			DirectoryMap: &proto.DirectoryMap{},
		},
	)

	conf := &loadedConfig.Config{
		Server: loadedConfig.Server{
			Host:     "127.0.0.1",
			GrpcPort: 9092,
		},
		Features: []string{agent_config.FeatureNginxConfig},
		Nginx: loadedConfig.Nginx{
			ConfigReloadMonitoringPeriod: 5 * time.Second,
		},
	}

	pluginUnderTest := NewNginx(commandClient, binary, env, conf)

	dir := t.TempDir()
	tempConf, err := os.CreateTemp(dir, "nginx.conf")
	assert.NoError(t, err)
	allowedDirectoriesMap := map[string]struct{}{dir: {}}
	configApply, err := sdk.NewConfigApply(tempConf.Name(), allowedDirectoriesMap)
	assert.NoError(t, err)

	response := &NginxConfigValidationResponse{
		err:           nil,
		correlationId: "123",
		nginxDetails: &proto.NginxDetails{
			NginxId:     "12345",
			ProcessId:   "123456",
			ProcessPath: "/var/test/",
		},
		config: &proto.NginxConfig{
			Action: proto.NginxConfigAction_APPLY,
			ConfigData: &proto.ConfigDescriptor{
				SystemId: "456",
				NginxId:  "12345",
				Checksum: "2314365",
			},
		},
		configApply: configApply,
	}

	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})
	messagePipe.Process(core.NewMessage(core.NginxConfigValidationSucceeded, response))
	go messagePipe.Run()

	time.Sleep(1 * time.Second)
	pluginUnderTest.syncProcessInfo(updatedProcesses)

	assert.Eventually(
		t,
		func() bool { return len(messagePipe.GetProcessedMessages()) == len(expectedTopics) },
		time.Duration(10*time.Second),
		3*time.Millisecond,
	)

	for idx, msg := range messagePipe.GetProcessedMessages() {
		if expectedTopics[idx] != msg.Topic() {
			t.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), expectedTopics[idx])
		}
	}
}

func TestNginx_rollbackConfigApply(t *testing.T) {
	expectedTopics := []string{
		core.NginxConfigValidationFailed,
		core.NginxPluginConfigured,
		core.NginxInstancesFound,
		core.ConfigRollbackResponse,
		core.NginxConfigApplyFailed,
		core.FileWatcherEnabled,
	}

	env := tutils.GetMockEnvWithProcess()
	env.On("GetSystemUUID").Return("456")

	binary := tutils.NewMockNginxBinary()
	binary.On("uploadConfig", mock.Anything, mock.Anything).Return(nil)
	binary.On("GetNginxDetailsByID", "12345").Return(tutils.GetDetailsMap()["12345"])
	binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)
	binary.On("UpdateNginxDetailsFromProcesses", env.Processes())
	binary.On("GetNginxDetailsMapFromProcesses", env.Processes()).Return(tutils.GetDetailsMap())
	binary.On("Reload", mock.Anything, mock.Anything)

	commandClient := tutils.GetMockCommandClient(
		&proto.NginxConfig{
			Action: proto.NginxConfigAction_APPLY,
			ConfigData: &proto.ConfigDescriptor{
				NginxId:  "12345",
				Checksum: "2314365",
			},
			Zconfig: &proto.ZippedFile{
				Contents:      first,
				Checksum:      checksum.Checksum(first),
				RootDirectory: "nginx.conf",
			},
			Zaux:         &proto.ZippedFile{},
			AccessLogs:   &proto.AccessLogs{},
			ErrorLogs:    &proto.ErrorLogs{},
			Ssl:          &proto.SslCertificates{},
			DirectoryMap: &proto.DirectoryMap{},
		},
	)

	conf := &loadedConfig.Config{Server: loadedConfig.Server{Host: "127.0.0.1", GrpcPort: 9092}, Features: []string{agent_config.FeatureNginxConfig}}

	pluginUnderTest := NewNginx(commandClient, binary, env, conf)

	dir := t.TempDir()
	tempConf, err := os.CreateTemp(dir, "nginx.conf")
	assert.NoError(t, err)
	allowedDirectoriesMap := map[string]struct{}{dir: {}}
	configApply, err := sdk.NewConfigApply(tempConf.Name(), allowedDirectoriesMap)
	assert.NoError(t, err)

	response := &NginxConfigValidationResponse{
		err:           errors.New("Failure"),
		correlationId: "123",
		nginxDetails: &proto.NginxDetails{
			NginxId:     "12345",
			ProcessId:   "123456",
			ProcessPath: "/var/test/",
		},
		config: &proto.NginxConfig{
			Action: proto.NginxConfigAction_APPLY,
			ConfigData: &proto.ConfigDescriptor{
				SystemId: "456",
				NginxId:  "12345",
				Checksum: "2314365",
			},
		},
		configApply: configApply,
	}

	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})
	messagePipe.Process(core.NewMessage(core.NginxConfigValidationFailed, response))
	messagePipe.Run()

	assert.Eventually(
		t,
		func() bool { return len(messagePipe.GetProcessedMessages()) == len(expectedTopics) },
		time.Duration(2*time.Second),
		1*time.Millisecond,
	)

	for idx, msg := range messagePipe.GetProcessedMessages() {
		if expectedTopics[idx] != msg.Topic() {
			t.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), expectedTopics[idx])
		}
	}
}

func TestBlock_ConfigApply(t *testing.T) {
	commandClient := tutils.GetMockCommandClient(tutils.GetNginxConfig(first))

	env := tutils.GetMockEnvWithProcess()
	binary := tutils.NewMockNginxBinary()
	binary.On("UpdateNginxDetailsFromProcesses", env.Processes())
	binary.On("GetNginxDetailsMapFromProcesses", env.Processes()).Return(tutils.GetDetailsMap())
	binary.On("Reload", mock.Anything, mock.Anything).Return(nil)

	config := tutils.GetMockAgentConfig()
	pluginUnderTest := NewNginx(commandClient, binary, env, config)

	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})
	messagePipe.Process(
		core.NewMessage(
			core.DataplaneSoftwareDetailsUpdated,
			payloads.NewDataplaneSoftwareDetailsUpdate(
				agent_config.NginxAppProtectExtensionPlugin,
				&proto.DataplaneSoftwareDetails{
					Data: testNAPDetailsActive,
				},
			),
		),
	)
	messagePipe.Run()

	assert.Equal(t, testNAPDetailsActive.AppProtectWafDetails.WafVersion, pluginUnderTest.nginxAppProtectSoftwareDetails.WafVersion)
}

func TestNginx_monitor(t *testing.T) {
	tmpDir := t.TempDir()
	errorLogFileName := path.Join(tmpDir, "/error.log")
	errorLogFile, err := os.Create(errorLogFileName)

	defer func() {
		err := errorLogFile.Close()
		require.NoError(t, err, "Error closing error log file")
		os.Remove(errorLogFile.Name())
	}()

	require.NoError(t, err, "Error creating error log")
	commandClient := tutils.GetMockCommandClient(&proto.NginxConfig{})

	env := tutils.GetMockEnvWithProcess()
	binary := tutils.NewMockNginxBinary()
	binary.On("GetErrorLogs").Return(make(map[string]string)).Once()
	binary.On("GetErrorLogs").Return(map[string]string{errorLogFileName: errorLogFileName}).Once()

	config := tutils.GetMockAgentConfig()
	config.Nginx.ConfigReloadMonitoringPeriod = 10 * time.Second
	pluginUnderTest := NewNginx(commandClient, binary, env, config)

	// Validate that errors in the logs returned
	go func() {
		errorFound := pluginUnderTest.monitor(pluginUnderTest.getNginxProccessInfo())
		assert.NoError(t, errorFound)
	}()

	time.Sleep(1 * time.Second)

	errorsChannel := make(chan error, 1)

	// Validate that errors in the logs returned
	go func() {
		errorFound := pluginUnderTest.monitor(pluginUnderTest.getNginxProccessInfo())
		errorsChannel <- errorFound
	}()
	time.Sleep(1 * time.Second)

	_, err = errorLogFile.WriteString("2023/03/14 14:16:23 [emerg] 3871#3871: bind() to 0.0.0.0:8081 failed (98: Address already in use)")
	require.NoError(t, err, "Error writing data to error log file")

	for {
		select {
		case x := <-errorsChannel:
			assert.Equal(t, "2023/03/14 14:16:23 [emerg] 3871#3871: bind() to 0.0.0.0:8081 failed (98: Address already in use)", x.Error())
			return
		case <-time.After((config.Nginx.ConfigReloadMonitoringPeriod * 2) * time.Second):
			assert.Fail(t, "Expected error to be reported")
			return
		}
	}
}

func TestNginx_monitorLog(t *testing.T) {
	tmpDir := t.TempDir()
	errorLogFileName := path.Join(tmpDir, "/error.log")
	errorLogFile, err := os.Create(errorLogFileName)
	errorLogs := map[string]string{errorLogFileName: errorLogFileName}

	defer func() {
		err := errorLogFile.Close()
		require.NoError(t, err, "Error closing error log file")
		os.Remove(errorLogFile.Name())
	}()

	require.NoError(t, err, "Error creating error log")
	commandClient := tutils.GetMockCommandClient(&proto.NginxConfig{})

	env := tutils.GetMockEnvWithProcess()
	binary := tutils.NewMockNginxBinary()
	binary.On("GetErrorLogs").Return(errorLogs)

	config := tutils.GetMockAgentConfig()
	config.Nginx.ConfigReloadMonitoringPeriod = 10 * time.Second
	pluginUnderTest := NewNginx(commandClient, binary, env, config)
	errorsChannel := make(chan string, 1)

	pluginUnderTest.monitorLogs(errorLogs, errorsChannel)

	// Validate that errors in the logs returned
	go func() {
		pluginUnderTest.monitorLogs(errorLogs, errorsChannel)
	}()

	time.Sleep(config.Nginx.ConfigReloadMonitoringPeriod / 2)

	_, err = errorLogFile.WriteString("2023/03/14 14:16:23 [emerg] 3871#3871: bind() to 0.0.0.0:8081 failed (98: Address already in use)")
	require.NoError(t, err, "Error writing data to error log file")

	for {
		select {
		case x := <-errorsChannel:
			assert.Equal(t, "2023/03/14 14:16:23 [emerg] 3871#3871: bind() to 0.0.0.0:8081 failed (98: Address already in use)", x)
			return
		case <-time.After((config.Nginx.ConfigReloadMonitoringPeriod * 2) * time.Second):
			assert.Fail(t, "Expected error to be reported")
			return
		}
	}
}
