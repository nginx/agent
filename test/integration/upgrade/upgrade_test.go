// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package upgrade

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

const (
	maxFileSize    int64 = 70000000
	maxUpgradeTime       = 30 * time.Second
	agentBuildDir        = "../agent/build"
	agentConfigDir       = "/etc/nginx-agent"
	agentLogDir          = "/var/log/nginx-agent"
)

var (
	osRelease   = os.Getenv("OS_RELEASE")
	packageName = os.Getenv("PACKAGE_NAME")
)

func Test_UpgradeFromV3(t *testing.T) {
	ctx := context.Background()

	containerNetwork := utils.CreateContainerNetwork(ctx, t)
	utils.SetupMockManagementPlaneGrpc(ctx, t, containerNetwork)
	defer func(ctx context.Context) {
		err := utils.MockManagementPlaneGrpcContainer.Terminate(ctx)
		require.NoError(t, err)
	}(ctx)

	// Prepare upgrade
	testContainer, teardownTest := upgradeSetup(t, true, "default", containerNetwork)
	defer teardownTest(t)

	slog.Info("starting agent v3 upgrade tests")

	// get currently installed agent version
	oldVersion := agentVersion(ctx, t, testContainer)

	// verify agent upgrade
	verifyAgentUpgrade(ctx, t, testContainer)

	// verify version of agent
	verifyAgentVersion(ctx, t, testContainer, oldVersion)

	// Expected files to validate after upgrade
	files := []helpers.ConfigFileDescriptor{
		{
			ContainerPath: agentConfigDir + "/nginx-agent.conf",
			ExpectedPath:  "./configs/default/nginx-agent.conf",
			LogLabel:      "agent config",
		},
		{
			ContainerPath: agentConfigDir + "/my_config.yaml",
			ExpectedPath:  "./configs/default/my_config.yaml",
			LogLabel:      "otel config",
		},
	}

	// validate agent manifest file
	expected := map[string]*model.ManifestFile{
		"/etc/nginx/nginx.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/nginx.conf",
				Hash:       "XEaOA4w+aT5fmNMISPwavBroLVYlkJf9sjKFTnWkTP8=",
				Size:       1142,
				Referenced: true,
			},
		},
	}
	// Check manifest file contents
	utils.CheckManifestFile(t, testContainer, expected)

	helpers.ValidateContainerFiles(ctx, t, testContainer, files)

	// Validate agent otel conf is present
	previousOtelConf := helpers.ExtractFileFromContainer(ctx, t,
		testContainer,
		agentConfigDir+"/opentelemetry-collector-agent.yaml")
	assert.NotEmpty(t, previousOtelConf)

	slog.Info("finished agent v3 upgrade tests")
}

func Test_UpgradeWithCustomOTELConfig(t *testing.T) {
	ctx := context.Background()

	containerNetwork := utils.CreateContainerNetwork(ctx, t)
	utils.SetupMockManagementPlaneGrpc(ctx, t, containerNetwork)
	defer func(ctx context.Context) {
		err := utils.MockManagementPlaneGrpcContainer.Terminate(ctx)
		require.NoError(t, err)
	}(ctx)

	testContainer, teardownTest := upgradeSetup(t, true, "custom_otel", containerNetwork)
	defer teardownTest(t)

	slog.Info("starting agent v3 upgrade tests with custom OTEL config")

	// get currently installed agent version
	oldVersion := agentVersion(ctx, t, testContainer)

	// verify agent upgrade
	verifyAgentUpgrade(ctx, t, testContainer)

	// verify version of agent
	verifyAgentVersion(ctx, t, testContainer, oldVersion)

	// Expected files to validate after upgrade
	files := []helpers.ConfigFileDescriptor{
		{
			ContainerPath: agentConfigDir + "/nginx-agent.conf",
			ExpectedPath:  "./configs/otel/nginx-agent.conf",
			LogLabel:      "agent config",
		},
		{
			ContainerPath: agentConfigDir + "/my_config.yaml",
			ExpectedPath:  "./configs/otel/my_config.yaml",
			LogLabel:      "otel custom config",
		},
		{
			ContainerPath: agentConfigDir + "/opentelemetry-collector-agent.yaml",
			ExpectedPath:  "./configs/otel/otel-config.yaml",
			LogLabel:      "otel config",
		},
	}
	// verify agent v3 configs has not changed
	helpers.ValidateContainerFiles(ctx, t, testContainer, files)

	// Validate agent.log contains OTEL startup log
	helpers.AssertStringInContainerFile(
		ctx, t, testContainer, agentLogDir+"/agent.log", "Starting OTel collector",
	)
	helpers.AssertStringInContainerFile(
		ctx,
		t,
		testContainer,
		agentLogDir+"/agent.log",
		"Merging additional OTel config files",
	)

	// Validate agent otel log contains specific logs
	helpers.AssertStringInContainerFile(
		ctx, t, testContainer, agentLogDir+"/opentelemetry-collector-agent.log",
		"Everything is ready. Begin running and processing data.",
	)

	slog.Info("finished agent v3 upgrade tests with custom OTEL config")
}

func upgradeSetup(tb testing.TB, expectNoErrorsInLogs bool, setupType string,
	containerNetwork *testcontainers.DockerNetwork,
) (testcontainers.Container, func(tb testing.TB)) {
	tb.Helper()
	ctx := context.Background()
	var params *helpers.Parameters

	switch setupType {
	case "custom_otel":
		params = &helpers.Parameters{
			NginxConfigPath:          "./configs/nginx-oss.conf",
			NginxAgentConfigPath:     "./configs/otel/nginx-agent.conf",
			NginxAgentOTELConfigPath: "./configs/otel/my_config.yaml",
			LogMessage:               "nginx_pid",
		}
	default:
		params = &helpers.Parameters{
			NginxConfigPath:          "./configs/nginx-oss.conf",
			NginxAgentConfigPath:     "./configs/default/nginx-agent.conf",
			NginxAgentOTELConfigPath: "./configs/default/my_config.yaml",
			LogMessage:               "nginx_pid",
		}
	}

	testContainer := helpers.StartContainer(
		ctx,
		tb,
		containerNetwork,
		params,
	)

	return testContainer, func(tb testing.TB) {
		tb.Helper()
		helpers.LogAndTerminateContainers(
			ctx,
			tb,
			nil,
			testContainer,
			expectNoErrorsInLogs,
			nil,
		)
	}
}

func verifyAgentUpgrade(ctx context.Context, tb testing.TB,
	testContainer testcontainers.Container,
) {
	tb.Helper()

	// cmdOut for validating logs
	_, upgradeTime := upgradeAgent(ctx, tb, testContainer)

	assert.LessOrEqual(tb, upgradeTime, maxUpgradeTime)
	tb.Log("Upgrade time: ", upgradeTime)
}

func upgradeAgent(ctx context.Context, tb testing.TB, testContainer testcontainers.Container,
) (io.Reader, time.Duration) {
	tb.Helper()

	var upgradeCmd []string

	if strings.Contains(osRelease, "ubuntu") || strings.Contains(osRelease, "debian") {
		upgradeCmd = []string{
			"apt-get", "install", "-y", "--only-upgrade",
			"/agent/build/" + packageName + ".deb", "-o", "Dpkg::Options::=--force-confold",
		}
	} else if strings.Contains(osRelease, "alpine") {
		upgradeCmd = []string{
			"apk", "add", "--allow-untrusted", "/agent/build/" + packageName + ".apk",
		}
	} else {
		upgradeCmd = []string{"yum", "reinstall", "-y", "/agent/build/" + packageName + ".rpm"}
	}

	start := time.Now()

	exitCode, cmdOut, err := testContainer.Exec(ctx, upgradeCmd)
	require.NoError(tb, err)

	stdoutStderr, err := io.ReadAll(cmdOut)
	require.NoError(tb, err)

	output := strings.TrimSpace(string(stdoutStderr))

	require.NoError(tb, err)
	assert.Equal(tb, 0, exitCode)
	tb.Logf("Upgrade command output: %s", output)

	duration := time.Since(start)

	return cmdOut, duration
}

func verifyAgentVersion(ctx context.Context, tb testing.TB, testContainer testcontainers.Container, oldVersion string) {
	tb.Helper()

	var newVersion string

	assert.Eventually(tb, func() bool {
		newVersion = agentVersion(ctx, tb, testContainer)

		return newVersion != oldVersion
	}, maxUpgradeTime, 100*time.Millisecond, "agent version not upgraded, still %s after upgrade", oldVersion)

	tb.Logf("agent upgraded to version %s successfully", newVersion)
}

func agentVersion(ctx context.Context, tb testing.TB, testContainer testcontainers.Container) string {
	tb.Helper()

	cmd := []string{"nginx-agent", "--version"}
	exitCode, cmdOut, err := testContainer.Exec(ctx, cmd)
	require.NoError(tb, err)
	assert.Equal(tb, 0, exitCode)

	stdoutStderr, err := io.ReadAll(cmdOut)
	require.NoError(tb, err)

	output := strings.TrimSpace(string(stdoutStderr))

	return output
}
