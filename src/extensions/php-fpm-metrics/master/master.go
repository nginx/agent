package master

import (
	//"bytes"
	"fmt"
	//"os/exec"
	re "regexp"
	"strings"
	"unicode"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pkg/phpfpm"
	log "github.com/sirupsen/logrus"
)

var (
	master_re, _            = re.Compile(`.*\((?P<conf_path>\/[^\)]*)\).*`)
	Shell        core.Shell = core.ExecShellCommand{}
)

type Master struct {
	host string
}

func New(host string) *Master {
	return &Master{
		host: host,
	}
}

// GetAll gets meta data of phpfpm master processes
func (m *Master) GetAll(phpProcess []*phpfpm.PhpProcess) (map[int32]*MetaData, error) {
	masterByPid := make(map[int32]*MetaData)
	for _, process := range phpProcess {
		find(process, masterByPid)
	}
	m.updateMetaData(masterByPid)
	return masterByPid, nil
}

// updateMetaData updates meta data of master processes from their config
func (m *Master) updateMetaData(masterByPid map[int32]*MetaData) {
	for _, meta := range masterByPid {
		meta.DisplayName = hostName(meta.Name, m.host)
		cmdSplit := strings.Split(meta.Cmd, "/")
		if len(cmdSplit) < 3 && cmdSplit[0] != "etc" && cmdSplit[1] != "php" {
			continue
		}
		meta.Version = cmdSplit[3]
		version, version_line, e := findVersion(meta.BinPath)

		if e == nil {
			meta.Version = version
			meta.VersionLine = version_line
		}
	}
}

func find(p *phpfpm.PhpProcess, masterByPid map[int32]*MetaData) {
	if p.IsMaster {
		_, ok := masterByPid[p.Pid]
		if !ok {
			masterByPid[p.Pid] = &MetaData{}
		}
		md := masterByPid[p.Pid]
		md.Name = "master"
		md.Type = "phpfpm"
		md.Cmd = p.Command
		md.ConfPath = configPath(md.Cmd)
		md.Pid = p.Pid
		md.BinPath = p.BinPath
	} else {
		_, ok := masterByPid[p.ParentPid]
		if !ok {
			md := &MetaData{}
			masterByPid[p.ParentPid] = md
		}
		masterByPid[p.ParentPid].NumWorkers++
	}
}

func findVersion(bin_path string) (string, string, error) {
	output, err := Shell.Exec(bin_path, "--version")
	if err != nil {
		log.Warnf("failed to get version and version line for php master process :  %v", err)
		return "", "", err
	}

	raw_lines := strings.Split(string(output), "\n")
	// Example: "PHP 7.4.33 (fpm-fcgi) (built: Feb 14 2023 18:31:23)"
	version_line := raw_lines[0]
	raw_version := strings.Fields(version_line)[1]

	var version string
	for _, character := range raw_version {
		if unicode.IsDigit(character) || character == '.' || character == '-' {
			version = fmt.Sprintf("%s%c", version, character)
		} else {
			break
		}
	}

	return version, version_line, nil
}

func configPath(line string) string {
	// Example: line = "php-fpm: master process (/etc/php/7.4/fpm/php-fpm.conf)"
	configPath := master_re.FindStringSubmatch(line)
	return configPath[1]
}

func hostName(name, host string) string {
	return fmt.Sprintf("phpfpm %s @ %s", name, host)
}
