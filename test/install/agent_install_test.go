package install

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	AGENT_PACKAGE_FILE = os.Getenv("AGENT_PACKAGE")
	maxFileSize        = int64(20000000)
	maxInstallTime     = 30 * time.Second
	expectedLogMsg     = LogMessages()
	expectedAgentDirs  = AgentDirectories()
	systemDetails      = SystemData()
)

/*
	Test Agent Install and Uninstall.
	Verifies that agent installs with correct output and files.
	Verifies that agent uninstalls and removes all the files.
*/
func TestAgentManualInstallUninstall(t *testing.T) {

	//Set up assertions
	checkAgentInstall := assert.New(t)

	//Check the agent tarball is present
	file, err := os.Stat(AGENT_PACKAGE_FILE)
	if err != nil {
		t.Errorf("Error accessing agent tarball at location: " + AGENT_PACKAGE_FILE)
	}

	//Install Agent and record installation time/install output
	installTime, agentLog := installAgent(AGENT_PACKAGE_FILE, t)

	//Check the file size is less than or equal 20MB
	assert.LessOrEqual(t, file.Size(), maxFileSize)

	//Check the install time under 30s
	assert.LessOrEqual(t, installTime, float64(maxInstallTime))

	//Check install output
	checkAgentInstall.Contains(agentLog, expectedLogMsg["InstallFoundNginxAgent"])
	checkAgentInstall.Contains(agentLog, expectedLogMsg["InstallAgentToRunAs"])
	checkAgentInstall.Contains(agentLog, expectedLogMsg["InstallCreateSystemFile"])
	checkAgentInstall.Contains(agentLog, expectedLogMsg["InstallAgentSuccess"])
	checkAgentInstall.Contains(agentLog, expectedLogMsg["InstallAgentStartCmd"])

	//Check nginx-agent config is created.
	_, agentConfigErr := os.Stat(expectedAgentDirs["AgentConfigFile"])
	checkAgentInstall.Nil(agentConfigErr)

	//Check nginx-agent system unit file is created.
	_, agentServiceFile := os.Stat(expectedAgentDirs["AgentSystemFile"])
	checkAgentInstall.Nil(agentServiceFile)

	//Uninstall the agent package
	uninstallLog := uninstallAgent("nginx-agent", t)

	//Check uninstall output
	checkAgentInstall.Contains(uninstallLog, expectedLogMsg["UninstallAgent"])
	checkAgentInstall.Contains(uninstallLog, expectedLogMsg["UninstallAgentStopService"])
	checkAgentInstall.Contains(uninstallLog, expectedLogMsg["UninstallAgentPurgingFiles"])

	//Check nginx-agent config is removed.
	_, deletedConfigErr := os.Stat(expectedAgentDirs["AgentConfigFile"])
	checkAgentInstall.NotNil(deletedConfigErr)

	//Check nginx-agent system unit file is removed.
	_, deletedServiceFileError := os.Stat(expectedAgentDirs["AgentSystemFile"])
	checkAgentInstall.NotNil(deletedServiceFileError)
}

//Installs the agent returning total install time and install output
func installAgent(agentPackage string, verify *testing.T) (float64, string) {

	//Get OS to create install cmd
	installCmd := createInstallCommand(verify)

	//Start install timer
	start := time.Now()

	//Start agent installation and capture install output
	cmd := exec.Command(installCmd[0], installCmd[1], installCmd[2], agentPackage)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		verify.Errorf("Error with installation: " + err.Error())
		verify.FailNow()
	}

	end := time.Now()
	elapsed := end.Sub(start)

	return float64(elapsed), string(stdoutStderr)
}

//Uninstall the agent returning output
func uninstallAgent(agentPackage string, verify *testing.T) string {

	//Get OS to create uninstall cmd
	uninstallCmd := createUninstallCommand(verify)

	//Start agent uninstall and capture uninstall output
	cmd := exec.Command(uninstallCmd[0], uninstallCmd[1], uninstallCmd[2], uninstallCmd[3], agentPackage)

	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		verify.Errorf("Error with uninstall: " + err.Error())
		verify.FailNow()
	}

	return string(stdoutStderr)
}

//Creates install command based on OS
func createInstallCommand(t *testing.T) []string {

	//Check OS release file exists first to determine OS
	_, err := os.Stat(systemDetails["OSReleaseFile"])
	if err != nil {
		t.Errorf("Error accessing os-release file " + err.Error())
	}
	content, _ := ioutil.ReadFile(systemDetails["OSReleaseFile"])
	os := string(content)
	if strings.Contains(os, "UBUNTU") || strings.Contains(os, "Debian") {
		return []string{"sudo", "apt", "install"}
	} else {
		return []string{"sudo", "yum", "install"}
	}
}

//Creates uninstall command based on OS
func createUninstallCommand(t *testing.T) []string {

	//Check OS release file exists first to determine OS
	_, err := os.Stat(systemDetails["OSReleaseFile"])
	if err != nil {
		t.Errorf("Error accessing os-release file " + err.Error())
	}
	content, _ := ioutil.ReadFile(systemDetails["OSReleaseFile"])
	os := string(content)
	if strings.Contains(os, "UBUNTU") || strings.Contains(os, "Debian") {
		return []string{"sudo", "apt", "purge", "-y"}
	} else {
		return []string{"sudo", "yum", "remove", "-y"}
	}
}

func LogMessages() map[string]string {
	return map[string]string{
		"NginxVersion":               "NGINX Agent v",
		"HeartbeatTopic":             "topic=comms.heartbeat",
		"RegSuccess":                 "msg=\"OneTimeRegistration completed\"",
		"HandshakeDone":              "topic=signal.handshake.done",
		"ConnectionStatus":           "agent_connect_response:<agent_config:<details:<> configs:<configs:<> > > status:<statusCode:CONNECT_OK > >",
		"InstallFoundNginxAgent":     "Found nginx-agent /usr/bin/nginx-agent",
		"InstallAgentToRunAs":        "nginx-agent will be configured to run as same user",
		"InstallCreateSystemFile":    "Creating directory /etc/systemd/system",
		"InstallAgentSuccess":        "NGINX Agent package has been successfully installed.",
		"InstallAgentStartCmd":       "sudo systemctl start nginx-agent",
		"UninstallAgent":             "Removing nginx-agent",
		"UninstallAgentStopService":  "Stop and disable nginx-agent service",
		"UninstallAgentPurgingFiles": "Purging configuration files for nginx-agent",
	}
}

func AgentDirectories() map[string]string {
	return map[string]string{
		"AgentConfigFile": "/etc/nginx-agent/nginx-agent.conf",
		"AgentInstallLog": "/tmp/agent-install.log",
		"AgentSystemFile": "/etc/systemd/system/multi-user.target.wants/nginx-agent.service",
	}
}

func SystemData() map[string]string {
	return map[string]string{
		"OSReleaseFile": "/etc/os-release",
	}
}
