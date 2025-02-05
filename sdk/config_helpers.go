/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sdk

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nginx/agent/sdk/v2/backoff"
	filesSDK "github.com/nginx/agent/sdk/v2/files"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/sdk/v2/zip"

	crossplane "github.com/nginxinc/nginx-go-crossplane"
	log "github.com/sirupsen/logrus"
)

const (
	plusAPIDirective          = "api"
	stubStatusAPIDirective    = "stub_status"
	apiFormat                 = "http://%s%s"
	predefinedAccessLogFormat = "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
	httpClientTimeout         = 1 * time.Second
)

var readLock = sync.Mutex{}

type DirectoryMap struct {
	paths map[string]*proto.Directory
}

func newDirectoryMap() *DirectoryMap {
	return &DirectoryMap{make(map[string]*proto.Directory)}
}

func (dm DirectoryMap) addDirectory(dir string) error {
	_, ok := dm.paths[dir]
	if !ok {
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("configs: could not read dir info(%s): %s", dir, err)
		}

		directory := &proto.Directory{
			Name:        dir,
			Mtime:       filesSDK.TimeConvert(info.ModTime()),
			Permissions: filesSDK.GetPermissions(info.Mode()),
			Size_:       info.Size(),
			Files:       make([]*proto.File, 0),
		}

		dm.paths[dir] = directory
	}
	return nil
}

func (dm DirectoryMap) appendFile(dir string, info fs.FileInfo) error {
	lineCount, err := filesSDK.GetLineCount(filepath.Join(dir, info.Name()))
	if err != nil {
		log.Debugf("Failed to get line count: %v", err)
	}

	fileProto := &proto.File{
		Name:        info.Name(),
		Lines:       int32(lineCount),
		Mtime:       filesSDK.TimeConvert(info.ModTime()),
		Permissions: filesSDK.GetPermissions(info.Mode()),
		Size_:       info.Size(),
	}

	return dm.appendFileWithProto(dir, fileProto)
}

func (dm DirectoryMap) appendFileWithProto(dir string, fileProto *proto.File) error {
	_, ok := dm.paths[dir]
	if !ok {
		err := dm.addDirectory(dir)
		if err != nil {
			return err
		}
	}

	dm.paths[dir].Files = append(dm.paths[dir].Files, fileProto)

	return nil
}

// GetNginxConfigWithIgnoreDirectives parse the configFile into proto.NginxConfig payload, using the provided nginxID, and systemID for
// ConfigDescriptor in the NginxConfig. The allowedDirectories is used to allowlist the directories we include
// in the aux payload.
func GetNginxConfigWithIgnoreDirectives(
	confFile,
	nginxId,
	systemId string,
	allowedDirectories map[string]struct{},
	ignoreDirectives []string,
) (*proto.NginxConfig, error) {
	readLock.Lock()
	lua := crossplane.Lua{}
	payload, err := crossplane.Parse(confFile,
		&crossplane.ParseOptions{
			IgnoreDirectives:   ignoreDirectives,
			SingleFile:         false,
			StopParsingOnError: true,
			LexOptions: crossplane.LexOptions{
				Lexers: []crossplane.RegisterLexer{lua.RegisterLexer()},
			},
		},
	)
	if err != nil {
		readLock.Unlock()
		return nil, fmt.Errorf("error reading config from %s, error: %s", confFile, err)
	}

	nginxConfig := &proto.NginxConfig{
		Action: proto.NginxConfigAction_RETURN,
		ConfigData: &proto.ConfigDescriptor{
			NginxId:  nginxId,
			SystemId: systemId,
		},
		Zconfig:      nil,
		Zaux:         nil,
		AccessLogs:   &proto.AccessLogs{AccessLog: make([]*proto.AccessLog, 0)},
		ErrorLogs:    &proto.ErrorLogs{ErrorLog: make([]*proto.ErrorLog, 0)},
		Ssl:          &proto.SslCertificates{SslCerts: make([]*proto.SslCertificate, 0)},
		DirectoryMap: &proto.DirectoryMap{Directories: make([]*proto.Directory, 0)},
	}

	err = updateNginxConfigFromPayload(confFile, payload, nginxConfig, allowedDirectories)
	if err != nil {
		readLock.Unlock()
		return nil, fmt.Errorf("error assemble payload from %s, error: %s", confFile, err)
	}

	readLock.Unlock()
	return nginxConfig, nil
}

// to ignore directives use GetNginxConfigWithIgnoreDirectives()
func GetNginxConfig(
	confFile,
	nginxId,
	systemId string,
	allowedDirectories map[string]struct{},
) (*proto.NginxConfig, error) {
	return GetNginxConfigWithIgnoreDirectives(confFile, nginxId, systemId, allowedDirectories, []string{})
}

// updateNginxConfigFromPayload updates config files from payload.
func updateNginxConfigFromPayload(
	confFile string,
	payload *crossplane.Payload,
	nginxConfig *proto.NginxConfig,
	allowedDirectories map[string]struct{},
) error {
	conf, err := zip.NewWriter(filepath.Dir(confFile))
	if err != nil {
		return fmt.Errorf("configs: could not create zip writer: %s", err)
	}
	aux, err := zip.NewWriter(filepath.Dir(confFile))
	if err != nil {
		return fmt.Errorf("configs: could not create auxillary zip writer: %s", err)
	}

	// cache the directory map, so we can look up using the base
	directoryMap := newDirectoryMap()
	formatMap := map[string]string{}                // map of accessLog/errorLog formats
	seen := make(map[string]struct{})               // local cache of seen files
	seenCerts := make(map[string]*x509.Certificate) // local cache of seen certs

	// Add files to the zipped config in a consistent order.
	if err = conf.AddFile(payload.Config[0].File); err != nil {
		return fmt.Errorf("configs: could not add conf(%s): %v", payload.Config[0].File, err)
	}

	rest := make([]crossplane.Config, len(payload.Config[1:]))
	copy(rest, payload.Config[1:])
	sort.Slice(rest, func(i, j int) bool {
		return rest[i].File < rest[j].File
	})
	for _, xpConf := range rest {
		if err = conf.AddFile(xpConf.File); err != nil {
			return fmt.Errorf("configs could not add conf file to archive: %s", err)
		}
	}

	// all files in the payload are config files
	var info fs.FileInfo
	for _, xpConf := range payload.Config {
		base := filepath.Dir(xpConf.File)

		info, err = os.Stat(xpConf.File)
		if err != nil {
			return fmt.Errorf("configs: could not read file info(%s): %s", xpConf.File, err)
		}

		if err = directoryMap.appendFile(base, info); err != nil {
			return err
		}

		err = updateNginxConfigFileConfig(xpConf, nginxConfig, filepath.Dir(confFile), aux, formatMap, seen, allowedDirectories, directoryMap, seenCerts)
		if err != nil {
			return fmt.Errorf("configs: failed to update nginx config: %s", err)
		}
	}

	nginxConfig.Zconfig, err = conf.Proto()
	if err != nil {
		return fmt.Errorf("configs: failed to get conf proto: %s", err)
	}

	if aux.FileLen() > 0 {
		nginxConfig.Zaux, err = aux.Proto()
		if err != nil {
			return fmt.Errorf("configs: failed to get aux proto: %s", err)
		}
	}

	setDirectoryMap(directoryMap, nginxConfig)

	return nil
}

func setDirectoryMap(directories *DirectoryMap, nginxConfig *proto.NginxConfig) {
	// empty the DirectoryMap first
	nginxConfig.DirectoryMap.Directories = nginxConfig.DirectoryMap.Directories[:0]
	keys := make([]string, 0, len(directories.paths))
	for k := range directories.paths {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		nginxConfig.DirectoryMap.Directories = append(nginxConfig.DirectoryMap.Directories, directories.paths[k])
	}
}

func updateNginxConfigFileConfig(
	conf crossplane.Config,
	nginxConfig *proto.NginxConfig,
	hostDir string,
	aux *zip.Writer,
	formatMap map[string]string,
	seen map[string]struct{},
	allowedDirectories map[string]struct{},
	directoryMap *DirectoryMap,
	seenCerts map[string]*x509.Certificate,
) error {
	err := CrossplaneConfigTraverse(&conf,
		func(parent *crossplane.Directive, directive *crossplane.Directive) (bool, error) {
			switch directive.Directive {
			case "log_format":
				if len(directive.Args) >= 2 {
					if directive.Args[0] == "ltsv" {
						formatMap[directive.Args[0]] = "ltsv"
					} else {
						formatMap[directive.Args[0]] = strings.Join(directive.Args[1:], "")
					}
				}
			case "root":
				if err := updateNginxConfigFileWithRoot(aux, directive.Args[0], seen, allowedDirectories, directoryMap); err != nil {
					return true, err
				}
			case "ssl_certificate", "proxy_ssl_certificate", "ssl_client_certificate", "ssl_trusted_certificate":
				if err := updateNginxConfigWithCert(directive.Directive, directive.Args[0], nginxConfig, aux, hostDir, directoryMap, allowedDirectories, seenCerts); err != nil {
					return true, err
				}
			case "access_log":
				updateNginxConfigWithAccessLog(
					directive.Args[0],
					getAccessLogDirectiveFormat(directive),
					nginxConfig, formatMap, seen)
			case "error_log":
				updateNginxConfigWithErrorLog(
					directive.Args[0],
					getErrorLogDirectiveLevel(directive),
					nginxConfig, seen)
			}
			return true, nil
		})
	if err != nil {
		return err
	}
	return nil
}

func updateNginxConfigWithCert(
	directive string,
	file string,
	nginxConfig *proto.NginxConfig,
	aux *zip.Writer,
	rootDir string,
	directoryMap *DirectoryMap,
	allowedDirectories map[string]struct{},
	seenCerts map[string]*x509.Certificate,
) error {
	if strings.Contains(file, "$") {
		// cannot process any filepath with variables
		return nil
	}

	if !filepath.IsAbs(file) {
		file = filepath.Join(rootDir, file)
	}
	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	isAllowed := false
	for dir := range allowedDirectories {
		if strings.HasPrefix(file, dir) {
			isAllowed = true
			break
		}
	}

	certDirectives := []string{
		"ssl_certificate",
		"proxy_ssl_certificate",
		"ssl_client_certificate",
		"ssl_trusted_certificate",
	}

	if contains(certDirectives, directive) {
		if seenCerts[file] != nil {
			log.Debugf("certs: %s duplicate. Skipping", file)
			return nil
		} else {
			cert, cErr := LoadCertificate(file)
			if cErr != nil {
				return fmt.Errorf("configs: could not load cert(%s): %s", file, cErr)
			}
			seenCerts[file] = cert
		}

		fingerprint := sha256.Sum256(seenCerts[file].Raw)
		certProto := &proto.SslCertificate{
			FileName:           file,
			PublicKeyAlgorithm: seenCerts[file].PublicKeyAlgorithm.String(),
			SignatureAlgorithm: seenCerts[file].SignatureAlgorithm.String(),
			Issuer: &proto.CertificateName{
				CommonName:         seenCerts[file].Issuer.CommonName,
				Country:            seenCerts[file].Issuer.Country,
				Locality:           seenCerts[file].Issuer.Locality,
				Organization:       seenCerts[file].Issuer.Organization,
				OrganizationalUnit: seenCerts[file].Issuer.OrganizationalUnit,
			},
			Subject: &proto.CertificateName{
				CommonName:         seenCerts[file].Subject.CommonName,
				Country:            seenCerts[file].Subject.Country,
				Locality:           seenCerts[file].Subject.Locality,
				Organization:       seenCerts[file].Subject.Organization,
				OrganizationalUnit: seenCerts[file].Subject.OrganizationalUnit,
				State:              seenCerts[file].Subject.Province,
			},
			Validity: &proto.CertificateDates{
				NotBefore: seenCerts[file].NotBefore.Unix(),
				NotAfter:  seenCerts[file].NotAfter.Unix(),
			},
			SubjAltNames:           seenCerts[file].DNSNames,
			SerialNumber:           seenCerts[file].SerialNumber.String(),
			OcspUrl:                seenCerts[file].IssuingCertificateURL,
			SubjectKeyIdentifier:   convertToHexFormat(hex.EncodeToString(seenCerts[file].SubjectKeyId)),
			Fingerprint:            convertToHexFormat(hex.EncodeToString(fingerprint[:])),
			FingerprintAlgorithm:   seenCerts[file].SignatureAlgorithm.String(),
			Version:                int64(seenCerts[file].Version),
			AuthorityKeyIdentifier: convertToHexFormat(hex.EncodeToString(seenCerts[file].AuthorityKeyId)),
		}
		certProto.Mtime = filesSDK.TimeConvert(info.ModTime())
		certProto.Size_ = info.Size()

		nginxConfig.Ssl.SslCerts = append(nginxConfig.Ssl.SslCerts, certProto)
	}

	if !isAllowed {
		log.Infof("certs: %s outside allowed directory, not including in aux payloads", file)
		// we want the meta information, but skip putting the files into the aux contents
		return nil
	}
	if err := directoryMap.appendFile(filepath.Dir(file), info); err != nil {
		return err
	}

	if err := aux.AddFile(file); err != nil {
		return fmt.Errorf("configs: could not add cert to aux file writer: %s", err)
	}

	return nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func getAccessLogDirectiveFormat(directive *crossplane.Directive) string {
	var format string
	if len(directive.Args) >= 2 {
		format = strings.ReplaceAll(directive.Args[1], "$", "")
	}
	return format
}

func getErrorLogDirectiveLevel(directive *crossplane.Directive) string {
	if len(directive.Args) >= 2 {
		return directive.Args[1]
	}
	return ""
}

func updateNginxConfigWithAccessLog(file string, format string, nginxConfig *proto.NginxConfig, formatMap map[string]string, seen map[string]struct{}) {
	if _, ok := seen[file]; ok {
		return
	}

	al := &proto.AccessLog{
		Name:     file,
		Readable: false,
	}

	info, err := os.Stat(file)
	if err == nil {
		// survivable error
		al.Readable = true
		al.Permissions = filesSDK.GetPermissions(info.Mode())
	}

	if formatMap[format] != "" {
		al.Format = formatMap[format]
	} else if format == "" || format == "combined" {
		al.Format = predefinedAccessLogFormat
	} else if format == "ltsv" {
		al.Format = format
	} else {
		al.Format = ""
	}

	nginxConfig.AccessLogs.AccessLog = append(nginxConfig.AccessLogs.AccessLog, al)
	seen[file] = struct{}{}
}

func updateNginxConfigWithAccessLogPath(file string, nginxConfig *proto.NginxConfig, seen map[string]struct{}) {
	if _, ok := seen[file]; ok {
		return
	}
	al := &proto.AccessLog{
		Name: file,
	}

	nginxConfig.AccessLogs.AccessLog = append(nginxConfig.AccessLogs.AccessLog, al)
	seen[file] = struct{}{}
}

func updateNginxConfigWithErrorLog(
	file string,
	level string,
	nginxConfig *proto.NginxConfig,
	seen map[string]struct{},
) {
	if _, ok := seen[file]; ok {
		return
	}
	el := &proto.ErrorLog{
		Name:     file,
		LogLevel: level,
		Readable: false,
	}
	info, err := os.Stat(file)
	if err == nil {
		// survivable error
		el.Permissions = filesSDK.GetPermissions(info.Mode())
		el.Readable = true
	}

	nginxConfig.ErrorLogs.ErrorLog = append(nginxConfig.ErrorLogs.ErrorLog, el)
	seen[file] = struct{}{}
}

func updateNginxConfigWithErrorLogPath(
	file string,
	nginxConfig *proto.NginxConfig,
	seen map[string]struct{},
) {
	if _, ok := seen[file]; ok {
		return
	}
	el := &proto.ErrorLog{
		Name: file,
	}
	nginxConfig.ErrorLogs.ErrorLog = append(nginxConfig.ErrorLogs.ErrorLog, el)
	seen[file] = struct{}{}
}

// root directive, so we slurp up all the files in the directory
func updateNginxConfigFileWithRoot(
	aux *zip.Writer,
	dir string,
	seen map[string]struct{},
	allowedDirectories map[string]struct{},
	directoryMap *DirectoryMap,
) error {
	if _, ok := seen[dir]; ok {
		return nil
	}
	seen[dir] = struct{}{}
	if !allowedPath(dir, allowedDirectories) {
		log.Debugf("Directory %s, is not in the allowed directory list so it will be excluded. Please add the directory to config_dirs in nginx-agent.conf", dir)
		return nil
	}

	return filepath.WalkDir(dir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if _, ok := seen[path]; ok {
				return nil
			}
			seen[path] = struct{}{}

			if d.IsDir() {
				if err := directoryMap.addDirectory(path); err != nil {
					return err
				}
				return nil
			}

			var info fs.FileInfo
			info, err = d.Info()
			if err != nil {
				return err
			}
			reader, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("could read file(%s): %s", path, err)
			}
			defer reader.Close()

			if err := directoryMap.appendFile(filepath.Dir(path), info); err != nil {
				return err
			}

			if err = aux.Add(path, info.Mode(), reader); err != nil {
				return fmt.Errorf("adding auxillary file error: %s", err)
			}

			return nil
		},
	)
}

func updateNginxConfigFileWithAuxFile(
	aux *zip.Writer,
	file string,
	config *proto.NginxConfig,
	seen map[string]struct{},
	allowedDirectories map[string]struct{},
	directoryMap *DirectoryMap,
	okIfFileNotExist bool,
) error {
	if _, ok := seen[file]; ok {
		return nil
	}
	if !allowedPath(file, allowedDirectories) {
		log.Warnf("Unable to retrieve the NAP aux file %s as it is not in the allowed directory list. Please add the directory to config_dirs in nginx-agent.conf.", file)
		return nil
	}

	info, err := os.Stat(file)
	if err != nil {
		if okIfFileNotExist {
			log.Debugf("Unable to retrieve the aux file %s.", file)
			return nil
		} else {
			return err
		}
	}

	if err := directoryMap.appendFile(filepath.Dir(file), info); err != nil {
		return err
	}

	if err := aux.AddFile(file); err != nil {
		return err
	}
	seen[file] = struct{}{}
	return nil
}

func GetNginxConfigFiles(config *proto.NginxConfig) (confFiles, auxFiles []*proto.File, err error) {
	if config.GetZconfig() == nil {
		return nil, nil, errors.New("config is empty")
	}

	confFiles, err = zip.UnPack(config.GetZconfig())
	if err != nil {
		return nil, nil, fmt.Errorf("unpack zipped config error: %s", err)
	}

	if aux := config.GetZaux(); aux != nil && len(aux.Contents) > 0 {
		auxFiles, err = zip.UnPack(aux)
		if err != nil {
			return nil, nil, fmt.Errorf("unpack zipped auxiliary error: %s", err)
		}
	}
	return confFiles, auxFiles, nil
}

// AddAuxfileToNginxConfig adds the specified newAuxFile to the Nginx Config cfg
func AddAuxfileToNginxConfig(
	confFile string,
	cfg *proto.NginxConfig,
	newAuxFile string,
	allowedDirectories map[string]struct{},
	okIfFileNotExist bool,
) (*proto.NginxConfig, error) {
	directoryMap := newDirectoryMap()
	for _, d := range cfg.DirectoryMap.Directories {
		for _, f := range d.Files {
			err := directoryMap.appendFileWithProto(d.Name, f)
			if err != nil {
				return nil, err
			}
		}
	}

	_, auxFiles, err := GetNginxConfigFiles(cfg)
	if err != nil {
		return nil, err
	}

	aux, err := zip.NewWriter(filepath.Dir(confFile))
	if err != nil {
		return nil, fmt.Errorf("configs: could not create auxillary zip writer: %s", err)
	}

	seen := make(map[string]struct{})
	for _, file := range auxFiles {
		seen[file.Name] = struct{}{}
		err = aux.AddFile(file.Name)
		if err != nil {
			return nil, err
		}
	}

	// add the aux file
	err = updateNginxConfigFileWithAuxFile(aux, newAuxFile, cfg, seen, allowedDirectories, directoryMap, okIfFileNotExist)
	if err != nil {
		return nil, fmt.Errorf("configs: failed to update nginx app protect metadata file: %s", err)
	}

	if aux.FileLen() > 0 {
		cfg.Zaux, err = aux.Proto()
		if err != nil {
			log.Errorf("configs: failed to get aux proto: %s", err)
			return nil, err
		}
	}

	setDirectoryMap(directoryMap, cfg)

	return cfg, nil
}

func parseAddressesFromServerDirective(parent *crossplane.Directive) []string {
	addresses := []string{}
	hosts := []string{}
	port := "80"

	for _, dir := range parent.Block {
		hostname := "127.0.0.1"

		switch dir.Directive {
		case "listen":
			host, listenPort, err := net.SplitHostPort(dir.Args[0])
			if err == nil {
				if host == "*" || host == "" {
					hostname = "127.0.0.1"
				} else if host == "::" || host == "::1" {
					hostname = "[::1]"
				} else {
					hostname = host
				}
				port = listenPort
			} else {
				if isPort(dir.Args[0]) {
					port = dir.Args[0]
				} else {
					hostname = dir.Args[0]
				}
			}
			hosts = append(hosts, hostname)
		case "server_name":
			if dir.Args[0] == "_" {
				// default server
				continue
			}
			hostname = dir.Args[0]
			hosts = append(hosts, hostname)
		}
	}

	for _, host := range hosts {
		addresses = append(addresses, fmt.Sprintf("%s:%s", host, port))
	}

	return addresses
}

func isPort(value string) bool {
	port, err := strconv.Atoi(value)
	return err == nil && port >= 1 && port <= 65535
}

func parsePathFromLocationDirective(location *crossplane.Directive) string {
	path := "/"
	if len(location.Args) > 0 {
		if location.Args[0] != "=" {
			path = location.Args[0]
		} else {
			path = location.Args[1]
		}
	}
	return path
}

func statusAPICallback(parent *crossplane.Directive, current *crossplane.Directive) string {
	ossUrls := getUrlsForLocationDirective(parent, current, stubStatusAPIDirective)
	plusUrls := getUrlsForLocationDirective(parent, current, plusAPIDirective)

	for _, url := range plusUrls {
		if pingNginxPlusApiEndpoint(url) {
			log.Debugf("api at %q found", url)
			return url
		}
		log.Debugf("api at %q is not reachable", url)
	}

	for _, url := range ossUrls {
		if pingStubStatusApiEndpoint(url) {
			log.Debugf("stub_status at %q found", url)
			return url
		}
		log.Debugf("stub_status at %q is not reachable", url)
	}

	return ""
}

// Deprecated: use either GetStubStatusApiUrl or GetNginxPlusApiUrl
func GetStatusApiInfoWithIgnoreDirectives(confFile string, ignoreDirectives []string) (statusApi string, err error) {
	payload, err := crossplane.Parse(confFile,
		&crossplane.ParseOptions{
			IgnoreDirectives:   ignoreDirectives,
			SingleFile:         false,
			StopParsingOnError: true,
			CombineConfigs:     true,
		},
	)
	if err != nil {
		return "", fmt.Errorf("error reading config from %s, error: %s", confFile, err)
	}

	for _, xpConf := range payload.Config {
		statusApi = CrossplaneConfigTraverseStr(&xpConf, statusAPICallback)
		if statusApi != "" {
			return statusApi, nil
		}
	}
	return "", errors.New("no status api reachable from the agent found")
}

// Deprecated: use either GetStubStatusApiUrl or GetNginxPlusApiUrl
// to ignore directives use GetStatusApiInfoWithIgnoreDirectives()
func GetStatusApiInfo(confFile string) (statusApi string, err error) {
	return GetStatusApiInfoWithIgnoreDirectives(confFile, []string{})
}

func GetStubStatusApiUrl(confFile string, ignoreDirectives []string) (stubStatusApiUrl string, err error) {
	payload, err := crossplane.Parse(confFile,
		&crossplane.ParseOptions{
			IgnoreDirectives:   ignoreDirectives,
			SingleFile:         false,
			StopParsingOnError: true,
			CombineConfigs:     true,
		},
	)
	if err != nil {
		return "", fmt.Errorf("error reading config from %s, error: %s", confFile, err)
	}

	for _, xpConf := range payload.Config {
		stubStatusApiUrl = CrossplaneConfigTraverseStr(&xpConf, stubStatusApiCallback)
		if stubStatusApiUrl != "" {
			return stubStatusApiUrl, nil
		}
	}
	return "", errors.New("no stub status api reachable from the agent found")
}

func GetNginxPlusApiUrl(confFile string, ignoreDirectives []string) (nginxPlusApiUrl string, err error) {
	payload, err := crossplane.Parse(confFile,
		&crossplane.ParseOptions{
			IgnoreDirectives:   ignoreDirectives,
			SingleFile:         false,
			StopParsingOnError: true,
			CombineConfigs:     true,
		},
	)
	if err != nil {
		return "", fmt.Errorf("error reading config from %s, error: %s", confFile, err)
	}

	for _, xpConf := range payload.Config {
		nginxPlusApiUrl = CrossplaneConfigTraverseStr(&xpConf, nginxPlusApiCallback)
		if nginxPlusApiUrl != "" {
			return nginxPlusApiUrl, nil
		}
	}
	return "", errors.New("no plus api reachable from the agent found")
}

func stubStatusApiCallback(parent *crossplane.Directive, current *crossplane.Directive) string {
	urls := getUrlsForLocationDirective(parent, current, stubStatusAPIDirective)

	for _, url := range urls {
		if pingStubStatusApiEndpoint(url) {
			log.Debugf("stub_status at %q found", url)
			return url
		}
		log.Debugf("stub_status at %q is not reachable", url)
	}

	return ""
}

func nginxPlusApiCallback(parent *crossplane.Directive, current *crossplane.Directive) string {
	urls := getUrlsForLocationDirective(parent, current, plusAPIDirective)

	for _, url := range urls {
		if pingNginxPlusApiEndpoint(url) {
			log.Debugf("plus API at %q found", url)
			return url
		}
		log.Debugf("plus API at %q is not reachable", url)
	}

	return ""
}

func pingStubStatusApiEndpoint(statusAPI string) bool {
	client := http.Client{Timeout: httpClientTimeout}
	resp, err := client.Get(statusAPI)
	if err != nil {
		log.Warningf("Unable to perform Stub Status API GET request: %v", err)
		return false
	}

	if resp.StatusCode != 200 {
		log.Debugf("Stub Status API responded with a %d status code", resp.StatusCode)
		return false
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warningf("Unable to read Stub Status API response body: %v", err)
		return false
	}

	// Expecting API to return data like this:
	//
	// Active connections: 2
	// server accepts handled requests
	//  18 18 3266
	// Reading: 0 Writing: 1 Waiting: 1
	body := string(bodyBytes)
	return strings.Contains(body, "Active connections") && strings.Contains(body, "server accepts handled requests")
}

func pingNginxPlusApiEndpoint(statusAPI string) bool {
	client := http.Client{Timeout: httpClientTimeout}
	resp, err := client.Get(statusAPI)
	if err != nil {
		log.Warningf("Unable to perform NGINX Plus API GET request: %v", err)
		return false
	}

	if resp.StatusCode != 200 {
		log.Debugf("NGINX Plus API responded with a %d status code", resp.StatusCode)
		return false
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Warningf("Unable to read NGINX Plus API response body: %v", err)
		return false
	}

	// Expecting API to return the api versions in an array of positive integers
	// subset example: [ ... 6,7,8,9 ...]
	var responseBody []int
	err = json.Unmarshal(bodyBytes, &responseBody)
	if err != nil {
		log.Debugf("Unable to unmarshal NGINX Plus API response body: %v", err)
		return false
	}

	return true
}

func getUrlsForLocationDirective(parent *crossplane.Directive, current *crossplane.Directive, locationDirectiveName string) []string {
	var urls []string
	// process from the location block
	if current.Directive != "location" {
		return urls
	}

	for _, locChild := range current.Block {
		if locChild.Directive != plusAPIDirective && locChild.Directive != stubStatusAPIDirective {
			continue
		}

		addresses := parseAddressesFromServerDirective(parent)

		for _, address := range addresses {
			path := parsePathFromLocationDirective(current)

			switch locChild.Directive {
			case locationDirectiveName:
				urls = append(urls, fmt.Sprintf(apiFormat, address, path))
			}
		}
	}
	return urls
}

func GetErrorAndAccessLogsWithIgnoreDirectives(confFile string, ignoreDirectives []string) (*proto.ErrorLogs, *proto.AccessLogs, error) {
	nginxConfig := &proto.NginxConfig{
		Action:       proto.NginxConfigAction_RETURN,
		ConfigData:   nil,
		Zconfig:      nil,
		Zaux:         nil,
		AccessLogs:   &proto.AccessLogs{AccessLog: make([]*proto.AccessLog, 0)},
		ErrorLogs:    &proto.ErrorLogs{ErrorLog: make([]*proto.ErrorLog, 0)},
		Ssl:          &proto.SslCertificates{SslCerts: make([]*proto.SslCertificate, 0)},
		DirectoryMap: &proto.DirectoryMap{Directories: make([]*proto.Directory, 0)},
	}

	payload, err := crossplane.Parse(confFile,
		&crossplane.ParseOptions{
			IgnoreDirectives:   ignoreDirectives,
			SingleFile:         false,
			StopParsingOnError: true,
		},
	)
	if err != nil {
		return nginxConfig.ErrorLogs, nginxConfig.AccessLogs, err
	}

	seen := make(map[string]struct{})
	for _, xpConf := range payload.Config {
		var err error
		err = CrossplaneConfigTraverse(&xpConf,
			func(parent *crossplane.Directive, current *crossplane.Directive) (bool, error) {
				switch current.Directive {
				case "access_log":
					updateNginxConfigWithAccessLogPath(current.Args[0], nginxConfig, seen)
				case "error_log":
					updateNginxConfigWithErrorLogPath(current.Args[0], nginxConfig, seen)
				}
				return true, nil
			})
		return nginxConfig.ErrorLogs, nginxConfig.AccessLogs, err
	}
	return nginxConfig.ErrorLogs, nginxConfig.AccessLogs, err
}

// to ignore directives use GetErrorAndAccessLogsWithIgnoreDirectives()
func GetErrorAndAccessLogs(confFile string) (*proto.ErrorLogs, *proto.AccessLogs, error) {
	return GetErrorAndAccessLogsWithIgnoreDirectives(confFile, []string{})
}

func GetErrorLogs(errorLogs *proto.ErrorLogs) []string {
	result := []string{}
	for _, log := range errorLogs.ErrorLog {
		result = append(result, log.Name)
	}
	return result
}

func GetAccessLogs(accessLogs *proto.AccessLogs) []string {
	result := []string{}
	for _, log := range accessLogs.AccessLog {
		result = append(result, log.Name)
	}
	return result
}

// allowedPath return true if the provided path has a prefix in the allowedDirectories, false otherwise. The
// path could be a filepath or directory.
func allowedPath(path string, allowedDirectories map[string]struct{}) bool {
	for d := range allowedDirectories {
		if strings.HasPrefix(path, d) {
			return true
		}
	}
	return false
}

func convertToHexFormat(hexString string) string {
	hexString = strings.ToUpper(hexString)
	formatted := ""
	for i := 0; i < len(hexString); i++ {
		if i > 0 && i%2 == 0 {
			formatted += ":"
		}
		formatted += string(hexString[i])
	}
	return formatted
}

func GetAppProtectPolicyAndSecurityLogFilesWithIgnoreDirectives(cfg *proto.NginxConfig, ignoreDirectives []string) ([]string, []string) {
	policyMap := make(map[string]bool)
	profileMap := make(map[string]bool)

	for _, directory := range cfg.GetDirectoryMap().GetDirectories() {
		for _, file := range directory.GetFiles() {
			confFile := path.Join(directory.GetName(), file.GetName())

			payload, err := crossplane.Parse(confFile,
				&crossplane.ParseOptions{
					IgnoreDirectives:   ignoreDirectives,
					SingleFile:         false,
					StopParsingOnError: true,
				},
			)
			if err != nil {
				continue
			}

			for _, conf := range payload.Config {
				err = CrossplaneConfigTraverse(&conf,
					func(parent *crossplane.Directive, directive *crossplane.Directive) (bool, error) {
						switch directive.Directive {
						case "app_protect_policy_file":
							if len(directive.Args) == 1 {
								_, policy := path.Split(directive.Args[0])
								policyMap[policy] = true
							}
						case "app_protect_security_log":
							if len(directive.Args) == 2 {
								_, profile := path.Split(directive.Args[0])
								profileMap[profile] = true
							}
						}
						return true, nil
					})
				if err != nil {
					continue
				}
			}
			if err != nil {
				continue
			}
		}
	}
	policies := []string{}
	for policy := range policyMap {
		policies = append(policies, policy)
	}

	profiles := []string{}
	for profile := range profileMap {
		profiles = append(profiles, profile)
	}

	return policies, profiles
}

// to ignore directives use GetAppProtectPolicyAndSecurityLogFilesWithIgnoreDirectives()
func GetAppProtectPolicyAndSecurityLogFiles(cfg *proto.NginxConfig) ([]string, []string) {
	return GetAppProtectPolicyAndSecurityLogFilesWithIgnoreDirectives(cfg, []string{})
}

func ConvertBackOffSettings(backOffSettings *proto.Backoff) backoff.BackoffSettings {
	multiplier := backoff.BACKOFF_MULTIPLIER
	if backOffSettings.GetMultiplier() != 0 {
		multiplier = backOffSettings.GetMultiplier()
	}

	jitter := backoff.BACKOFF_JITTER
	if backOffSettings.GetRandomizationFactor() != 0 {
		jitter = backOffSettings.GetRandomizationFactor()
	}
	cBackoff := backoff.BackoffSettings{
		InitialInterval: time.Duration(backOffSettings.InitialInterval * int64(time.Second)),
		MaxInterval:     time.Duration(backOffSettings.MaxInterval * int64(time.Second)),
		MaxElapsedTime:  time.Duration(backOffSettings.MaxElapsedTime * int64(time.Second)),
		Multiplier:      multiplier,
		Jitter:          jitter,
	}

	return cBackoff
}
