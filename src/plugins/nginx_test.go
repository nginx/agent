package plugins

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/checksum"
	"github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"

	"github.com/nginx/agent/v2/src/core"
	loadedConfig "github.com/nginx/agent/v2/src/core/config"

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
)

func TestNginxConfigApply(t *testing.T) {
	validationTimeout = 0 * time.Millisecond
	t.Parallel()

	tests := []struct {
		config    *proto.NginxConfig
		msgTopics []string
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
				core.CommResponse,
				core.NginxConfigValidationSucceeded,
			},
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
				core.CommResponse,
				core.NginxConfigValidationSucceeded,
			},
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
				core.CommResponse,
				core.NginxConfigValidationSucceeded,
			},
		},
	}

	cmd := &proto.Command{
		Meta: grpc.NewMessageMeta(uuid.New().String()),
		Type: 1,
		Data: &proto.Command_NginxConfig{
			NginxConfig: &proto.NginxConfig{
				Action: proto.NginxConfigAction_APPLY,
				ConfigData: &proto.ConfigDescriptor{
					NginxId:  "12345",
					Checksum: "test",
				},
			},
		},
	}

	for idx, test := range tests {
		test := test
		t.Run(fmt.Sprintf("%d", idx), func(tt *testing.T) {
			tt.Parallel()
			dir := t.TempDir()
			tempConf, err := ioutil.TempFile(dir, "nginx.conf")
			assert.NoError(t, err)

			err = ioutil.WriteFile(tempConf.Name(), fourth, 0644)
			assert.NoError(t, err)

			ctx := context.TODO()

			env := tutils.GetMockEnvWithProcess()
			allowedDirectoriesMap := map[string]struct{}{dir: {}}

			config, err := sdk.NewConfigApply(tempConf.Name(), allowedDirectoriesMap)
			assert.NoError(t, err)

			binary := tutils.NewMockNginxBinary()
			binary.On("WriteConfig", mock.Anything).Return(config, nil)
			binary.On("ValidateConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			binary.On("GetNginxDetailsByID", "12345").Return(tutils.GetDetailsMap()["12345"])
			binary.On("UpdateNginxDetailsFromProcesses", env.Processes())
			binary.On("GetNginxDetailsMapFromProcesses", env.Processes()).Return((tutils.GetDetailsMap()))

			commandClient := tutils.GetMockCommandClient(test.config)

			pluginUnderTest := NewNginx(commandClient, binary, env, &loadedConfig.Config{Features: []string{loadedConfig.FeatureNginxConfig}})
			messagePipe := core.SetupMockMessagePipe(t, ctx, pluginUnderTest)

			messagePipe.Process(core.NewMessage(core.CommNginxConfig, cmd))
			messagePipe.Run()
			processedMessages := messagePipe.GetProcessedMessages()

			assert.Eventually(
				tt,
				func() bool { return len(processedMessages) != len(test.msgTopics) },
				time.Duration(15*time.Millisecond),
				3*time.Millisecond,
				fmt.Sprintf("Expected %d messages but only processed %d messages", len(test.msgTopics), len(processedMessages)),
			)
			binary.AssertExpectations(tt)
			env.AssertExpectations(tt)

			for idx, msg := range processedMessages {
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

	pluginUnderTest := NewNginx(cmdr, binary, env, &loadedConfig.Config{Features: []string{loadedConfig.FeatureNginxConfig}})
	messagePipe := core.SetupMockMessagePipe(t, context.Background(), pluginUnderTest)

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
	messagePipe := core.SetupMockMessagePipe(t, context.Background(), pluginUnderTest)

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
	messagePipe := core.SetupMockMessagePipe(t, context.Background(), pluginUnderTest)

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

	pluginUnderTest := NewNginx(cmdr, binary, env, &loadedConfig.Config{Features: []string{loadedConfig.FeatureNginxConfig}})
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

func TestNginx_completeConfigApply(t *testing.T) {
	expectedTopics := []string{
		core.NginxConfigValidationSucceeded,
		core.NginxPluginConfigured,
		core.NginxInstancesFound,
		core.CommResponse,
		core.FileWatcherEnabled,
		core.NginxReloadComplete,
		core.NginxConfigApplySucceeded,
	}

	env := tutils.GetMockEnvWithProcess()
	env.On("GetSystemUUID").Return("456")

	binary := tutils.NewMockNginxBinary()
	binary.On("uploadConfig", mock.Anything, mock.Anything).Return(nil)
	binary.On("GetNginxDetailsByID", "12345").Return(tutils.GetDetailsMap()["12345"])
	binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)
	binary.On("UpdateNginxDetailsFromProcesses", env.Processes())
	binary.On("GetNginxDetailsMapFromProcesses", env.Processes()).Return((tutils.GetDetailsMap()))
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

	pluginUnderTest := NewNginx(commandClient, binary, env, &loadedConfig.Config{Features: []string{loadedConfig.FeatureNginxConfig}})

	dir := t.TempDir()
	tempConf, err := ioutil.TempFile(dir, "nginx.conf")
	assert.NoError(t, err)
	allowedDirectoriesMap := map[string]struct{}{dir: {}}
	configApply, err := sdk.NewConfigApply(tempConf.Name(), allowedDirectoriesMap)

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

	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), pluginUnderTest)
	messagePipe.Process(core.NewMessage(core.NginxConfigValidationSucceeded, response))
	messagePipe.Run()

	processedMessages := messagePipe.GetProcessedMessages()

	assert.Eventually(
		t,
		func() bool { return len(processedMessages) == len(expectedTopics) },
		time.Duration(15*time.Millisecond),
		3*time.Millisecond,
		fmt.Sprintf("Expected %d messages but only processed %d messages", len(expectedTopics), len(processedMessages)),
	)

	for idx, msg := range processedMessages {
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
	binary.On("GetNginxDetailsMapFromProcesses", env.Processes()).Return((tutils.GetDetailsMap()))
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

	pluginUnderTest := NewNginx(commandClient, binary, env, &loadedConfig.Config{Features: []string{loadedConfig.FeatureNginxConfig}})

	dir := t.TempDir()
	tempConf, err := ioutil.TempFile(dir, "nginx.conf")
	assert.NoError(t, err)
	allowedDirectoriesMap := map[string]struct{}{dir: {}}
	configApply, err := sdk.NewConfigApply(tempConf.Name(), allowedDirectoriesMap)

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

	messagePipe := core.SetupMockMessagePipe(t, context.TODO(), pluginUnderTest)
	messagePipe.Process(core.NewMessage(core.NginxConfigValidationFailed, response))
	messagePipe.Run()

	processedMessages := messagePipe.GetProcessedMessages()

	assert.Eventually(
		t,
		func() bool { return len(processedMessages) == len(expectedTopics) },
		time.Duration(5*time.Millisecond),
		1*time.Millisecond,
		fmt.Sprintf("Expected %d messages but only processed %d messages", len(expectedTopics), len(processedMessages)),
	)

	for idx, msg := range processedMessages {
		if expectedTopics[idx] != msg.Topic() {
			t.Errorf("unexpected message topic: %s :: should have been: %s", msg.Topic(), expectedTopics[idx])
		}
	}
}
