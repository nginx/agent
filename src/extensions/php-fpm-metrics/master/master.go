package master

import (
	"fmt"
	re "regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/nginx/agent/v2/src/core"
	log "github.com/sirupsen/logrus"
)

// Todo: Leverage gopsutil
var Shell core.Shell = core.ExecShellCommand{}

var (
	master_re, _ = re.Compile(`.*\((?P<conf_path>\/[^\)]*)\).*`)
	process_re   = re.MustCompile(`\s*(?P<pid>\d+)\s+(?P<ppid>\d+)\s+(?P<cmd>.+)\s*`)
)

type Master struct {
	host string
}

func NewMaster(host string) *Master {
	return &Master{
		host: host,
	}
}

// GetAll gets meta data of phpfpm master processes
func (m *Master) GetAll() (map[string]*MetaData, error) {
	masterByPid := make(map[string]*MetaData)
	ps, err := Shell.Exec("ps xao pid,ppid,command | grep 'php-fpm[:]'")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ps info about php-fpm: %v", err)
	}

	psSplit := strings.Split(string(ps), "\n")
	for _, l := range psSplit {
		if len(l) == 0 {
			continue
		}
		// find master process pid and creates corresponding metadata object, or else increment worker count for pid
		find(l, masterByPid)
	}
	m.updateMetaData(masterByPid)
	return masterByPid, nil
}

// updateMetaData updates meta data of master processes from their config
func (m *Master) updateMetaData(masterByPid map[string]*MetaData) {
	for ppid, meta := range masterByPid {
		meta.DisplayName = hostName(meta.Name, m.host)
		binPath, err := findBinPath(ppid)
		if err != nil {
			continue
		}
		meta.BinPath = binPath
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

func find(l string, masterByPid map[string]*MetaData) {
	parsed := parseProcess(l)
	pid, ppid, cmd := parsed[1], parsed[2], parsed[3]

	// get master info, otherwise a pool worker
	if strings.Contains(cmd, "master process") {
		md := &MetaData{}
		masterByPid[pid] = md
		md.Cmd = cmd
		md.Name = "master"
		md.Type = "phpfpm"
		md.ConfPath = configPath(cmd)
		pidAsInt, err := strconv.Atoi(pid)
		if err != nil {
			log.Warnf("failed to convert pid %s to integer", pid)
			return
		}
		md.Pid = int32(pidAsInt)
	} else {
		masterByPid[ppid].NumWorkers++
	}
}

func findBinPath(ppid string) (string, error) {
	// Example: lrwxrwxrwx 1 root root 0 Aug 24 21:09 /proc/654040/exe -> /usr/sbin/php-fpm7.4
	binCmd := fmt.Sprintf("/proc/%s/exe", ppid)
	output, err := Shell.Exec("sudo", "ls", "-la", binCmd)
	if err != nil {
		log.Warnf("failed to run bin command : %s, %v", binCmd, err)
		return "", err
	} else {
		binFields := strings.Fields(string(output))
		l := len(binFields) - 1
		return binFields[l], nil
	}
}

func findVersion(bin_path string) (string, string, error) {
	output, err := Shell.Exec(fmt.Sprintf("sudo %s --version", bin_path))
	if err != nil {
		log.Warnf("failed to get version and version line for php master process%v", err)
		return "", "", err
	}

	raw_lines := strings.Split(string(output), "\n")
	// Example: "PHP 7.4.33 (fpm-fcgi) (built: Feb 14 2023 18:31:23)"
	version_line := raw_lines[1]
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

func parseProcess(line string) []string {
	// parse ps response line.
	// Example: line = 36  1 php-fpm: master process (/etc/php/7.0/fpm/php-fpm.conf)
	return process_re.FindStringSubmatch(line)
}
