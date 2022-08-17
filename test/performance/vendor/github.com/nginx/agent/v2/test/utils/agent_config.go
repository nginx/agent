package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/spf13/viper"

	"github.com/nginx/agent/v2/src/core/config"
	sysutils "github.com/nginx/agent/v2/test/utils/system"
	log "github.com/sirupsen/logrus"
)

const (
	tempAgentConfFileName        = "temp-nginx-agent.conf"
	tempDynamicAgentConfFileName = "temp-agent-dynamic.conf"
	DisplayNameKey               = "display_name"
	DefaultTestDisplayName       = "nginx-agent-repo"
)

var (
	// Get variables for parent directories
	_, absFilePath, _, _ = runtime.Caller(0)
	absUtilsDirPath      = filepath.Dir(absFilePath)
	absTestDirPath       = filepath.Dir(absUtilsDirPath)
	delFunc              = func() {}

	// Absolute paths to test files
	testAgentConfPath        = absTestDirPath + "/testdata/configs/nginx-agent.conf"
	testAgentDynamicConfPath = absTestDirPath + "/testdata/configs/agent-dynamic.conf"
)

func GetMockAgentConfig() *config.Config {
	return &config.Config{
		ClientID:   "12345",
		Tags:       InitialConfTags,
		ConfigDirs: "/testDirs",
		AgentMetrics: config.AgentMetrics{
			BulkSize:           1,
			ReportInterval:     5,
			CollectionInterval: 1,
			Mode:               "aggregated",
		},
	}
}

// GetTestAgentConfigPath gets the absolute path to the test agent config
func GetTestAgentConfigPath() string {
	return testAgentConfPath
}

// CreateTestAgentConfigEnv creates an agent config file named "temp-nginx-agent.conf"
// and a dynamic config named "temp-agent-dynamic.conf" meant for testing in the current
// working directory. Additionally, a Viper config is created that has its variables set
// based off the created conf files ("temp-nginx-agent.conf" and "temp-agent-dynamic.conf").
// It returns the name of the config ("nginx-agent.conf"), the name of the of the dynamic
// config ("temp-agent-dynamic.conf"), and a function to call that deletes the both of the
// files that were created.
func CreateTestAgentConfigEnv() (string, string, func(), error) {
	wg := &sync.WaitGroup{}
	err := make(chan error, 1)

	wg.Add(1)
	go func() {
		err <- setupRoutine(wg)
	}()
	wg.Wait()

	if err := <-err; err != nil {
		return "", "", nil, err
	}

	return tempAgentConfFileName, tempDynamicAgentConfFileName, delFunc, nil
}

func setupRoutine(wg *sync.WaitGroup) error {
	// Setup Viper and Config variables
	// Register the temp agent config that was created
	// Set viper config properties from created test config

	// Create a temp agent conf and dynamic agent conf in the current directory
	// for calling tests to utilize
	confDeleteFunc, err := sysutils.CopyFile(testAgentConfPath, tempAgentConfFileName)
	if err != nil {
		err = confDeleteFunc()
		if err != nil {
			log.Errorf("error occurred deleting configuration: %v", err)
		}
		return fmt.Errorf("error copying file %s to destination %s", testAgentConfPath, tempAgentConfFileName)
	}

	dynamicConfDeleteFunc, err := sysutils.CopyFile(testAgentDynamicConfPath, tempDynamicAgentConfFileName)
	if err != nil {
		delFunc()
		return fmt.Errorf("error copying file %s to destination %s", testAgentConfPath, tempAgentConfFileName)
	}

	// Create the delete func that is responsible for cleaning up both temp files
	// that are created
	delFunc = func() {
		err = confDeleteFunc()
		if err != nil {
			log.Errorf("error occurred deleting configuration: %v", err)
		}
		err = dynamicConfDeleteFunc()
		if err != nil {
			log.Errorf("error occurred deleting dynamic configuration: %v", err)
		}
	}

	// Set the Viper values and variables
	viper.Reset()
	os.Clearenv()
	config.ROOT_COMMAND.ResetFlags()
	config.ROOT_COMMAND.ResetCommands()
	config.Viper = viper.NewWithOptions(viper.KeyDelimiter(config.KeyDelimiter))
	config.SetDefaults()
	config.RegisterFlags()

	// Get current directory to allow
	curDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cfg, err := config.RegisterConfigFile(fmt.Sprintf("%s/%s", curDir, tempDynamicAgentConfFileName), tempAgentConfFileName, []string{"."}...)
	if err != nil {
		delFunc()
		return fmt.Errorf("failed to register config file (%s) - %v", tempAgentConfFileName, err)
	}

	err = config.LoadPropertiesFromFile(cfg)
	if err != nil {
		delFunc()
		return fmt.Errorf("failed to load properties from config file (%s) - %v", tempAgentConfFileName, err)
	}

	config.Viper.Set(config.ConfigPathKey, cfg)
	wg.Done()
	return nil
}
