/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"
)

const (
	withWithPrefix        = "with-"
	withModuleSuffix      = "module"
	defaultNginxOssPrefix = "/usr/local/nginx"
)

var (
	logMutex    sync.Mutex
	unpackMutex sync.Mutex
	re          = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+)`)
	plusre      = regexp.MustCompile(`(?P<name>\S+)/(?P<version>\S+).\((?P<plus>\S+plus\S+)\)`)
)

type NginxBinary interface {
	Start(nginxId, bin string) error
	Stop(processId, bin string) error
	Reload(processId, bin string) error
	ValidateConfig(processId, bin, configLocation string, config *proto.NginxConfig, configApply *sdk.ConfigApply) error
	GetNginxDetailsFromProcess(nginxProcess Process) *proto.NginxDetails
	GetNginxDetailsByID(nginxID string) *proto.NginxDetails
	GetNginxIDForProcess(nginxProcess Process) string
	GetNginxDetailsMapFromProcesses(nginxProcesses []Process) map[string]*proto.NginxDetails
	UpdateNginxDetailsFromProcesses(nginxProcesses []Process)
	WriteConfig(config *proto.NginxConfig) (*sdk.ConfigApply, error)
	ReadConfig(path, nginxId, systemId string) (*proto.NginxConfig, error)
	UpdateLogs(existingLogs map[string]string, newLogs map[string]string) bool
	GetAccessLogs() map[string]string
	GetErrorLogs() map[string]string
	GetChildProcesses() map[string][]*proto.NginxDetails
}

type NginxBinaryType struct {
	detailsMapMutex   sync.Mutex
	workersMapMutex   sync.Mutex
	env               Environment
	config            *config.Config
	nginxDetailsMap   map[string]*proto.NginxDetails
	nginxWorkersMap   map[string][]*proto.NginxDetails
	nginxInfoMap      map[string]*nginxInfo
	accessLogs        map[string]string
	errorLogs         map[string]string
	accessLogsUpdated bool
	errorLogsUpdated  bool
}

type nginxInfo struct {
	version         string
	mtime           time.Time
	plusver         string
	source          string
	prefix          string
	confPath        string
	logPath         string
	errorPath       string
	ssl             []string
	cfgf            map[string]interface{}
	configureArgs   []string
	loadableModules []string
	modulesPath     string
}

func NewNginxBinary(env Environment, config *config.Config) *NginxBinaryType {
	return &NginxBinaryType{
		env:          env,
		nginxInfoMap: make(map[string]*nginxInfo),
		accessLogs:   make(map[string]string),
		errorLogs:    make(map[string]string),
		config:       config,
	}
}

func (n *NginxBinaryType) GetNginxDetailsMapFromProcesses(nginxProcesses []Process) map[string]*proto.NginxDetails {
	n.detailsMapMutex.Lock()
	defer n.detailsMapMutex.Unlock()
	return n.nginxDetailsMap
}

func (n *NginxBinaryType) UpdateNginxDetailsFromProcesses(nginxProcesses []Process) {
	n.detailsMapMutex.Lock()
	defer n.detailsMapMutex.Unlock()
	n.nginxDetailsMap = map[string]*proto.NginxDetails{}

	n.workersMapMutex.Lock()
	defer n.workersMapMutex.Unlock()
	n.nginxWorkersMap = map[string][]*proto.NginxDetails{}

	for _, process := range nginxProcesses {
		nginxDetails := n.GetNginxDetailsFromProcess(process)
		if process.IsMaster {
			n.nginxDetailsMap[nginxDetails.GetNginxId()] = nginxDetails
		} else {
			n.nginxWorkersMap[nginxDetails.GetNginxId()] = append(n.nginxWorkersMap[nginxDetails.GetNginxId()], nginxDetails)
		}
	}
}

func (n *NginxBinaryType) GetChildProcesses() map[string][]*proto.NginxDetails {
	n.workersMapMutex.Lock()
	defer n.workersMapMutex.Unlock()
	return n.nginxWorkersMap
}

func (n *NginxBinaryType) GetNginxIDForProcess(nginxProcess Process) string {
	defaulted := n.sanitizeProcessPath(&nginxProcess)
	info := n.getNginxInfoFrom(nginxProcess.Path)

	// reset the process path from the default to what NGINX tells us
	if defaulted &&
		info.cfgf["sbin-path"] != nil &&
		nginxProcess.Path != info.cfgf["sbin-path"] {
		nginxProcess.Path = info.cfgf["sbin-path"].(string)
	}

	return n.getNginxIDFromProcessInfo(nginxProcess, info)
}

func (n *NginxBinaryType) getNginxIDFromProcessInfo(nginxProcess Process, info *nginxInfo) string {
	return GenerateNginxID("%s_%s_%s", nginxProcess.Path, info.confPath, info.prefix)
}

func (n *NginxBinaryType) GetNginxDetailsByID(nginxID string) *proto.NginxDetails {
	n.detailsMapMutex.Lock()
	defer n.detailsMapMutex.Unlock()
	return n.nginxDetailsMap[nginxID]
}

func (n *NginxBinaryType) sanitizeProcessPath(nginxProcess *Process) bool {
	defaulted := false
	if nginxProcess.Path == "" {
		nginxProcess.Path = defaultToNginxCommandForProcessPath()
		defaulted = true
	}
	if strings.Contains(nginxProcess.Path, execDeleted) {
		log.Debugf("nginx was upgraded (process), using new info")
		nginxProcess.Path = sanitizeExecDeletedPath(nginxProcess.Path)
	}
	return defaulted
}

func (n *NginxBinaryType) GetNginxDetailsFromProcess(nginxProcess Process) *proto.NginxDetails {
	defaulted := n.sanitizeProcessPath(&nginxProcess)
	info := n.getNginxInfoFrom(nginxProcess.Path)

	// reset the process path from the default to what NGINX tells us
	if defaulted &&
		info.cfgf["sbin-path"] != nil &&
		nginxProcess.Path != info.cfgf["sbin-path"] {
		nginxProcess.Path = info.cfgf["sbin-path"].(string)
	}

	nginxID := n.getNginxIDFromProcessInfo(nginxProcess, info)
	log.Tracef("NGINX %s %s %s %v nginxID=%s conf=%s", info.plusver, info.source, info.ssl, info.cfgf, nginxID, info.confPath)

	nginxDetailsFacade := &proto.NginxDetails{
		NginxId:         nginxID,
		Version:         info.version,
		ConfPath:        info.confPath,
		ProcessId:       fmt.Sprintf("%d", nginxProcess.Pid),
		ProcessPath:     nginxProcess.Path,
		StartTime:       nginxProcess.CreateTime,
		BuiltFromSource: false,
		LoadableModules: info.loadableModules,
		RuntimeModules:  runtimeFromConfigure(info.configureArgs),
		Plus:            buildPlus(info.plusver),
		Ssl:             buildSsl(info.ssl, info.source),
		ConfigureArgs:   info.configureArgs,
	}

	if path := getConfPathFromCommand(nginxProcess.Command); path != "" {
		log.Tracef("Custom conf path set: %v", path)
		nginxDetailsFacade.ConfPath = path
	}

	url, err := sdk.GetStatusApiInfo(nginxDetailsFacade.ConfPath)
	if err != nil {
		log.Tracef("Unable to get status api from the configuration: NGINX metrics will be unavailable for this system. please configure a status API to get NGINX metrics: %v", err)
	}
	nginxDetailsFacade.StatusUrl = url

	return nginxDetailsFacade
}

func defaultToNginxCommandForProcessPath() string {
	log.Debug("Defaulting to NGINX on path")

	// LookPath figures out the full path of the binary using the $PATH
	// command is not portable
	path, err := exec.LookPath("nginx")
	if err != nil {
		log.Warnf("Unable to find the correct NGINX binary in $PATH: %v", err)
		return ""
	}
	return path
}

// Start starts NGINX.
func (n *NginxBinaryType) Start(nginxId, bin string) error {
	log.Infof("Starting NGINX Id: %s Bin: %s", nginxId, bin)

	_, err := runCmd(bin)
	if err != nil {
		log.Errorf("Starting NGINX caused error: %v", err)
	} else {
		log.Infof("NGINX Id: %s Started", nginxId)
	}

	return err
}

// Reload NGINX.
func (n *NginxBinaryType) Reload(processId, bin string) error {
	log.Infof("Reloading NGINX: %s PID: %s", bin, processId)
	intProcess, err := strconv.Atoi(processId)
	if err != nil {
		log.Errorf("Reloading NGINX caused error when trying to determine process id: %v", err)
		return err
	}

	err = syscall.Kill(intProcess, syscall.SIGHUP)
	if err != nil {
		log.Errorf("Reloading NGINX caused error: %v", err)
	} else {
		log.Infof("NGINX with process Id: %s reloaded", processId)
	}
	return err
}

// ValidateConfig tests the config with nginx -t -c configLocation.
func (n *NginxBinaryType) ValidateConfig(processId, bin, configLocation string, config *proto.NginxConfig, configApply *sdk.ConfigApply) error {
	log.Debugf("Validating config, %s for nginx process, %s", configLocation, processId)
	response, err := runCmd(bin, "-t", "-c", configLocation)
	if err != nil {
		confFiles, auxFiles, getNginxConfigFilesErr := sdk.GetNginxConfigFiles(config)
		if getNginxConfigFilesErr == nil {
			n.writeBackup(config, confFiles, auxFiles)
		}
		return fmt.Errorf("error running nginx -t -c %v:\n%s", configLocation, response)
	}

	log.Infof("Config validated:\n%s", response)

	return nil
}

// Stop stops an instance of NGINX.
func (n *NginxBinaryType) Stop(processId, bin string) error {
	log.Info("Stopping NGINX")

	_, err := runCmd(bin, "-s", "stop")
	if err != nil {
		log.Errorf("Stopping NGINX caused error: %v", err)
	} else {
		log.Infof("NGINX with process Id: %s stopped", processId)
	}

	return err
}

func ensureFilesAllowed(files []*proto.File, allowList map[string]struct{}, path string) error {
	for _, file := range files {
		filename := file.Name
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(path, filename)
		}
		log.Tracef("checking file %s is allowed", filename)
		if !allowedFile(filename, allowList) {
			return fmt.Errorf("the file %s is outside the allowed directory list", filename)
		}
	}
	return nil
}

func hasConfPath(files []*proto.File, confPath string) bool {
	confDir := filepath.Dir(confPath)
	for _, file := range files {
		filename := file.Name
		if !filepath.IsAbs(filename) {
			filename = filepath.Join(confDir, filename)
		}
		if filename == confPath {
			return true
		}
	}
	return false
}

func (n *NginxBinaryType) WriteConfig(config *proto.NginxConfig) (*sdk.ConfigApply, error) {
	if log.IsLevelEnabled(log.TraceLevel) {
		jsonConfig, err := json.Marshal(config)
		if err != nil {
			log.Tracef("Writing raw config: %+v", config)
		} else {
			log.Tracef("Writing JSON config: %+v", string(jsonConfig))
		}
	}

	details, ok := n.nginxDetailsMap[config.ConfigData.NginxId]
	if !ok || details == nil {
		return nil, fmt.Errorf("NGINX instance %s not found", config.ConfigData.NginxId)
	}

	systemNginxConfig, err := sdk.GetNginxConfig(
		details.ConfPath,
		config.ConfigData.NginxId,
		config.ConfigData.SystemId,
		n.config.AllowedDirectoriesMap,
	)
	if err != nil {
		return nil, err
	}

	if !allowedFile(filepath.Dir(details.ConfPath), n.config.AllowedDirectoriesMap) {
		return nil, fmt.Errorf("config directory %s not allowed", filepath.Dir(details.ConfPath))
	}

	confFiles, auxFiles, err := sdk.GetNginxConfigFiles(config)
	if err != nil {
		return nil, err
	}

	// Ensure this config request does not remove the process config
	if !hasConfPath(confFiles, details.ConfPath) {
		return nil, fmt.Errorf("should not delete %s", details.ConfPath)
	}

	// Ensure all config files are within the allowed list directories.
	confDir := filepath.Dir(details.ConfPath)
	if err := ensureFilesAllowed(confFiles, n.config.AllowedDirectoriesMap, confDir); err != nil {
		return nil, err
	}

	// Ensure all aux files are within the allowed list directories.
	if err := ensureFilesAllowed(auxFiles, n.config.AllowedDirectoriesMap, config.GetZaux().GetRootDirectory()); err != nil {
		return nil, err
	}

	unpackMutex.Lock()
	defer unpackMutex.Unlock()

	log.Info("Updating NGINX config")
	var configApply *sdk.ConfigApply
	configApply, err = sdk.NewConfigApply(details.ConfPath, n.config.AllowedDirectoriesMap)
	if err != nil {
		log.Warnf("config_apply error: %s", err)
		return nil, err
	}

	// TODO: return to Control Plane that there was a rollback
	err = n.env.WriteFiles(configApply, confFiles, filepath.Dir(details.ConfPath), n.config.AllowedDirectoriesMap)
	if err != nil {
		log.Warnf("configuration write failed: %s", err)
		n.writeBackup(config, confFiles, auxFiles)
		return configApply, err
	}

	if len(auxFiles) > 0 {
		auxPath := config.GetZaux().GetRootDirectory()
		err = n.env.WriteFiles(configApply, auxFiles, auxPath, n.config.AllowedDirectoriesMap)
		if err != nil {
			log.Warnf("Auxiliary files write failed: %s", err)
			return configApply, err
		}
	}

	filesToDelete, ok := generateDeleteFromDirectoryMap(config.DirectoryMap, n.config.AllowedDirectoriesMap)
	if ok {
		log.Debugf("use explicit set action for delete files %s", filesToDelete)
	} else {
		// Delete files that are not in the directory map
		filesToDelete = getDirectoryMapDiff(systemNginxConfig.DirectoryMap.Directories, config.DirectoryMap.Directories)
	}

	fileDeleted := make(map[string]struct{})
	for _, file := range filesToDelete {
		log.Infof("Deleting file: %s", file)
		if _, ok = fileDeleted[file]; ok {
			continue
		}

		if found, foundErr := FileExists(file); !found {
			if foundErr == nil {
				log.Debugf("skip delete for non-existing file: %s", file)
				continue
			}
			// possible perm deny, depends on platform
			log.Warnf("file exists returned for %s: %s", file, foundErr)
			return configApply, foundErr
		}
		if err = configApply.MarkAndSave(file); err != nil {
			return configApply, err
		}
		if err = os.Remove(file); err != nil {
			return configApply, err
		}
		fileDeleted[file] = struct{}{}
	}

	return configApply, nil
}

// generateDeleteFromDirectoryMap return a list of delete files from the directory map where Action File_delete is set.
// This supports incremental upgrade if the files in the DirectoryMap doesn't have any action set to a non-default value,
// in which the return bool will be false, to indicate explicit action is not set in the provided DirectoryMap.
func generateDeleteFromDirectoryMap(
	directoryMap *proto.DirectoryMap,
	allowedDirectory map[string]struct{},
) ([]string, bool) {
	actionIsSet := false
	if directoryMap == nil {
		return nil, actionIsSet
	}
	deleteFiles := make([]string, 0)
	for _, dir := range directoryMap.Directories {
		for _, f := range dir.Files {
			if f.Action == proto.File_unset {
				continue
			}
			actionIsSet = true
			if f.Action != proto.File_delete {
				continue
			}
			path := filepath.Join(dir.Name, f.Name)
			if !filepath.IsAbs(path) {
				// can't assume relative path
				continue
			}
			if !allowedFile(path, allowedDirectory) {
				continue
			}
			deleteFiles = append(deleteFiles, path)
		}
	}
	return deleteFiles, actionIsSet
}

func (n *NginxBinaryType) ReadConfig(confFile, nginxId, systemId string) (*proto.NginxConfig, error) {
	configPayload, err := sdk.GetNginxConfig(confFile, nginxId, systemId, n.config.AllowedDirectoriesMap)
	if err != nil {
		return nil, err
	}

	// get access logs list for analysis
	accessLogs := AccessLogs(configPayload)
	// get error logs list for analysis
	errorLogs := ErrorLogs(configPayload)

	logMutex.Lock()
	defer logMutex.Unlock()

	n.accessLogsUpdated = n.UpdateLogs(n.accessLogs, accessLogs)
	n.errorLogsUpdated = n.UpdateLogs(n.errorLogs, errorLogs)

	return configPayload, nil
}

func (n *NginxBinaryType) GetAccessLogs() map[string]string {
	logMutex.Lock()
	defer logMutex.Unlock()
	return n.accessLogs
}

func (n *NginxBinaryType) GetErrorLogs() map[string]string {
	logMutex.Lock()
	defer logMutex.Unlock()
	return n.errorLogs
}

// SkipLog checks if a logfile should be omitted from analysis
func (n *NginxBinaryType) SkipLog(filename string) bool {
	if n.config != nil {
		for _, filter := range strings.Split(n.config.Nginx.ExcludeLogs, ",") {
			ok, err := filepath.Match(filter, filename)
			if err != nil {
				log.Error("invalid path spec for excluding access_log: ", filter)
			} else if ok {
				log.Debugf("excluding access log %q as specified by filter: %q", filename, filter)
				return true
			}
		}
	}
	return false
}

func (n *NginxBinaryType) writeBackup(config *proto.NginxConfig, confFiles []*proto.File, auxFiles []*proto.File) {
	if n.config.Nginx.Debug {
		allowedDirs := map[string]struct{}{"/tmp": {}}
		path := filepath.Join("/tmp", strconv.FormatInt(time.Now().Unix(), 10))

		configApply, err := sdk.NewConfigApply("/tmp", n.config.AllowedDirectoriesMap)
		if err != nil {
			log.Warnf("config_apply error: %s", err)
			return
		}

		log.Tracef("Writing failed configuration to %s", path)

		confFilesCopy := deepCopyWithNewPath(confFiles, config.Zconfig.RootDirectory, path)

		err = n.env.WriteFiles(configApply, confFilesCopy, path, allowedDirs)
		if err != nil {
			log.Warnf("Error writing to config %s", err)
		}

		auxFilesCopy := deepCopyWithNewPath(auxFiles, config.Zconfig.RootDirectory, path)

		err = n.env.WriteFiles(configApply, auxFilesCopy, path, allowedDirs)
		if err != nil {
			log.Warnf("Error writing to aux %s", err)
		}

		if err = configApply.Complete(); err != nil {
			log.Errorf("Backup config complete failed: %v", err)
		}
	}
}

func deepCopyWithNewPath(files []*proto.File, oldPath, newPath string) []*proto.File {
	filesCopy := make([]*proto.File, len(files))
	for index, file := range files {
		filesCopy[index] = &proto.File{
			Name:                 strings.ReplaceAll(file.Name, oldPath, newPath),
			Lines:                file.Lines,
			Mtime:                file.Mtime,
			Permissions:          file.Permissions,
			Size_:                file.Size_,
			Contents:             file.Contents,
			XXX_NoUnkeyedLiteral: file.XXX_NoUnkeyedLiteral,
			XXX_unrecognized:     file.XXX_unrecognized,
			XXX_sizecache:        file.XXX_sizecache,
		}
	}
	return filesCopy
}

func getConfPathFromCommand(command string) string {
	commands := strings.Split(command, " ")

	for i, command := range commands {
		if command == "-c" {
			if i < len(commands)-1 {
				return commands[i+1]
			}
		}
	}
	return ""
}

func parseConfigureArguments(line string) (result map[string]interface{}, flags []string) {
	// need to check for empty strings
	flags = strings.Split(line[len("configure arguments:"):], " --")
	result = map[string]interface{}{}
	for _, flag := range flags {
		vals := strings.Split(flag, "=")
		switch len(vals) {
		case 1:
			if vals[0] != "" {
				result[vals[0]] = true
			}
		case 2:
			result[vals[0]] = vals[1]
		default:
			break
		}
	}
	return result, flags
}

func (n *NginxBinaryType) getNginxInfoFrom(ngxExe string) *nginxInfo {
	if ngxExe == "" {
		return &nginxInfo{}
	}
	if strings.Contains(ngxExe, execDeleted) {
		log.Infof("nginx was upgraded, using new info")
		ngxExe = sanitizeExecDeletedPath(ngxExe)
	}
	if info, ok := n.nginxInfoMap[ngxExe]; ok {
		stat, err := os.Stat(ngxExe)
		if err == nil && stat.ModTime().Equal(info.mtime) {
			return info
		}
	}
	outbuf, err := runCmd(ngxExe, "-V")
	if err != nil {
		log.Errorf("nginx -V failed (%s): %v", outbuf.String(), err)
		return &nginxInfo{}
	}

	info := n.getNginxInfoFromBuffer(ngxExe, outbuf)
	n.nginxInfoMap[ngxExe] = info
	return info
}

const (
	execDeleted = "(deleted)"
)

func sanitizeExecDeletedPath(exe string) string {
	firstSpace := strings.Index(exe, execDeleted)
	if firstSpace != -1 {
		return strings.TrimSpace(exe[0:firstSpace])
	}
	return strings.TrimSpace(exe)
}

// getNginxInfoFromBuffer -
func (n *NginxBinaryType) getNginxInfoFromBuffer(exePath string, buffer *bytes.Buffer) *nginxInfo {
	info := &nginxInfo{}
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "nginx version"):
			info.version, info.plusver = parseNginxVersion(line)
		case strings.HasPrefix(line, "configure arguments"):
			info.cfgf, info.configureArgs = parseConfigureArguments(line)
		case strings.HasPrefix(line, "built by"):
			info.source = line
		case strings.HasPrefix(line, "built with"):
			l := strings.ReplaceAll(line, "built with ", "")
			sslInfo := strings.SplitN(l, " ", 3)
			for i := range sslInfo {
				sslInfo[i] = strings.TrimSpace(sslInfo[i])
			}
			info.ssl = sslInfo
		}
	}

	if info.cfgf["modules-path"] != nil {
		info.modulesPath = info.cfgf["modules-path"].(string)
		if info.modulesPath != "" {
			info.loadableModules, _ = n.parseModulePath(info.modulesPath)
		}
	}

	if info.cfgf["prefix"] == nil {
		info.prefix = defaultNginxOssPrefix
	} else {
		info.prefix = info.cfgf["prefix"].(string)
	}

	// conf path default value but -c overrides it elsewhere
	if info.cfgf["conf-path"] != nil {
		info.confPath = info.cfgf["conf-path"].(string)
	} else {
		// if conf-path is not specified, assume nginx is built from source and that there is a config file in the config directory
		info.confPath = path.Join(info.prefix, "/conf/nginx.conf")
	}

	if info.cfgf["http-log-path"] != nil {
		info.logPath = info.cfgf["http-log-path"].(string)
	}

	if info.cfgf["error-log-path"] != nil {
		info.errorPath = info.cfgf["error-log-path"].(string)
	}
	stat, err := os.Stat(exePath)
	if err == nil {
		info.mtime = stat.ModTime()
	}
	return info
}

func (n *NginxBinaryType) parseModulePath(dir string) ([]string, error) {
	result, err := n.env.ReadDirectory(dir, ".so")
	if err != nil {
		log.Errorf("Unable to parse module path %v", err)
		return nil, err
	}
	return result, nil
}

func (n *NginxBinaryType) UpdateLogs(existingLogs map[string]string, newLogs map[string]string) bool {
	logUpdated := false

	for logFile, logFormat := range newLogs {
		if !(strings.HasPrefix(logFile, "syslog:") || n.SkipLog(logFile)) {
			if _, found := existingLogs[logFile]; !found || existingLogs[logFile] != logFormat {
				logUpdated = true
			}
			existingLogs[logFile] = logFormat
		}
	}

	// delete old logs
	for logFile := range existingLogs {
		if _, found := newLogs[logFile]; !found {
			delete(existingLogs, logFile)
			logUpdated = true
		}
	}

	return logUpdated
}

func parseNginxVersion(line string) (version, plusVersion string) {
	matches := re.FindStringSubmatch(line)
	plusmatches := plusre.FindStringSubmatch(line)

	if len(plusmatches) > 0 {
		subNames := plusre.SubexpNames()
		for i, v := range plusmatches {
			switch subNames[i] {
			case "plus":
				plusVersion = v
			case "version":
				version = v
			}
		}
		return version, plusVersion
	}

	if len(matches) > 0 {
		for i, key := range re.SubexpNames() {
			val := matches[i]
			if key == "version" {
				version = val
			}
		}
	}

	return version, plusVersion
}

func buildSsl(ssl []string, source string) *proto.NginxSslMetaData {
	var nginxSslType proto.NginxSslMetaData_NginxSslType
	if strings.HasPrefix(source, "built") {
		nginxSslType = proto.NginxSslMetaData_BUILT
	} else {
		nginxSslType = proto.NginxSslMetaData_RUN
	}

	return &proto.NginxSslMetaData{
		SslType: nginxSslType,
		Details: ssl,
	}
}

func buildPlus(plusver string) *proto.NginxPlusMetaData {
	plus := false
	if plusver != "" {
		plus = true
	}
	return &proto.NginxPlusMetaData{
		Enabled: plus,
		Release: plusver,
	}
}

// runtimeFromConfigure parse and return the runtime modules from `nginx -V` configured args
// these are usually in the form of "with-X_module", so we just look for "with" prefix, and "module" suffix.
func runtimeFromConfigure(configure []string) []string {
	pkgs := make([]string, 0)
	for _, arg := range configure {
		if i := strings.Index(arg, withWithPrefix); i > -1 && strings.HasSuffix(arg, withModuleSuffix) {
			pkgs = append(pkgs, arg[i+len(withWithPrefix):])
		}
	}

	return pkgs
}

// AccessLogs returns a list of access logs in the config
func AccessLogs(p *proto.NginxConfig) map[string]string {
	var found = make(map[string]string)
	for _, accessLog := range p.GetAccessLogs().GetAccessLog() {
		// check if the access log is readable or not
		if accessLog.GetReadable() && accessLog.GetName() != "off" {
			name := strings.Split(accessLog.GetName(), " ")[0]
			format := accessLog.GetFormat()
			found[name] = format
		} else {
			log.Warnf("NGINX Access log %s is not readable or is disabled. Please make it readable and enabled in order for NGINX metrics to be collected.", accessLog.GetName())
		}
	}

	return found
}

// ErrorLogs returns a list of error logs in the config
func ErrorLogs(p *proto.NginxConfig) map[string]string {
	var found = make(map[string]string)
	for _, errorLog := range p.GetErrorLogs().GetErrorLog() {
		// check if the error log is readable or not
		if errorLog.GetReadable() {
			name := strings.Split(errorLog.GetName(), " ")[0]
			// In the future, different error log formats will be supported
			found[name] = ""
		} else {
			log.Warnf("NGINX Error log %s is not readable or is disabled. Please make it readable and enabled in order for NGINX metrics to be collected.", errorLog.GetName())
		}
	}

	return found
}

// Returns a list of files that are in the currentDirectoryMap but not in the incomingDirectoryMap
func getDirectoryMapDiff(currentDirectoryMap []*proto.Directory, incomingDirectoryMap []*proto.Directory) []string {
	diff := []string{}

	incomingMap := make(map[string]struct{})
	for _, incomingDirectory := range incomingDirectoryMap {
		for _, incomingFile := range incomingDirectory.Files {
			filePath := incomingFile.Name
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(incomingDirectory.Name, filePath)
			}
			incomingMap[filePath] = struct{}{}
		}
	}

	for _, currentDirectory := range currentDirectoryMap {
		for _, currentFile := range currentDirectory.Files {
			filePath := currentFile.Name
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(currentDirectory.Name, currentFile.Name)
			}
			if _, ok := incomingMap[filePath]; !ok {
				diff = append(diff, filePath)
			}
		}
	}

	return diff
}
