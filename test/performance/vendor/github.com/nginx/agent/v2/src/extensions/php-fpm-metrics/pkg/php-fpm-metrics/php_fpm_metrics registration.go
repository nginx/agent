package php_fpm

import (
	"fmt"
	"os/exec"
	re "regexp"
	"strconv"
	"strings"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	logger "github.com/sirupsen/logrus"
)

type PhpFpmRegistration struct {
	Env core.Environment
}

const (
	phpFpmDegradedMessage = "Nginx App Protect is installed but is not running"
)

func (p *PhpFpmRegistration) GeneratePhpFpmDetailsProtoCommand(version string) *proto.DataplaneSoftwareDetails_PhpFpmDetails {
	phpfpmMetaReport, err := phpFpmMetaDataReport(p.Env, version)
	var phpfpmStatus proto.PhpFpmHealth_PhpFpmHealthStatus
	degradedReason := ""
	switch phpfpmMetaReport.Status {
	case UNKNOWN.String():
		phpfpmStatus = proto.PhpFpmHealth_UNKNOWN
	case INSTALLED.String():
		phpfpmStatus = proto.PhpFpmHealth_DEGRADED
		degradedReason = phpFpmDegradedMessage
	case RUNNING.String():
		phpfpmStatus = proto.PhpFpmHealth_ACTIVE
	}

	if err != nil {
		logger.Warnf("failed to generate php report: %v", err)
		return &proto.DataplaneSoftwareDetails_PhpFpmDetails{
			PhpFpmDetails: &proto.PhpFpmDetails{
				Health: &proto.PhpFpmHealth{
					PhpfpmHealthStatus: phpfpmStatus,
					SystemId:           p.Env.GetSystemUUID(),
				},
			},
		}
	}
	details := &proto.DataplaneSoftwareDetails_PhpFpmDetails{
		PhpFpmDetails: &proto.PhpFpmDetails{
			Type:     phpfpmMetaReport.Type,
			RootUuid: phpfpmMetaReport.RootUUID,
			LocalId:  phpfpmMetaReport.LocalId,
			Name:     phpfpmMetaReport.Name,
			Cmd:      phpfpmMetaReport.Cmd,
			ConfPath: phpfpmMetaReport.ConfPath,
			Workers:  phpfpmMetaReport.NumWorkers,
			Version:  phpfpmMetaReport.Version,
			Pid:      phpfpmMetaReport.Pid,
			BinPath:  phpfpmMetaReport.BinPath,
			Children: toProtoPhpFpmPool(phpfpmMetaReport.Pools, phpfpmMetaReport.LocalId),
			Health: &proto.PhpFpmHealth{
				PhpfpmHealthStatus: phpfpmStatus,
				SystemId:           p.Env.GetSystemUUID(),
				DegradedReason:     degradedReason,
			},
		},
	}

	return details
}

func toProtoPhpFpmPool(children []*PhpFpmPool, parent_local_id string) []*proto.PhpFpmPool {
	pools := []*proto.PhpFpmPool{}
	for _, child := range children {
		pool := &proto.PhpFpmPool{
			Type:            child.Type,
			RootUuid:        child.RootUUID,
			LocalId:         child.LocalId,
			Name:            child.Name,
			DisplayName:     child.DisplayName,
			Listen:          child.Listen,
			Flisten:         child.Flisten,
			StatusPath:      child.StatusPath,
			CanHaveChildren: child.CanHaveChildren,
			Agent:           child.Agent,
			ParentLocalId:   parent_local_id,
		}
		pools = append(pools, pool)
	}
	return pools
}

func masterProcess(line string) string {
	// TODO: Should be initialized only once
	regex, _ := re.Compile(`.*\((?P<conf_path>\/[^\)]*)\).*`)
	configPath := regex.FindStringSubmatch(line)
	return configPath[0]
}

func processParse(line string) []string {
	// TODO: Should be initialized only once
	//# parse ps response line...examples::
	//#    36     1 php-fpm: master process (/etc/php/7.0/fpm/php-fpm.conf)
	//#    37    36 php-fpm: pool www
	//#    38    36 php-fpm: pool www
	regex := re.MustCompile(`\s*(?P<pid>\d+)\s+(?P<ppid>\d+)\s+(?P<cmd>.+)\s*`)
	return regex.FindStringSubmatch(line)
}

func phpFpmMetaDataReport(env core.Environment, agent string) (*PhpFpmMetaData, error) {
	// search for the php fpm master process. Is it running? installed? degraded?
	status, err := GetPhpFpmStatus()
	if err != nil {
		return nil, err
	}

	// if the status is not running, we have nothing more to do
	if status != RUNNING {
		return &PhpFpmMetaData{
			Status: status.String(),
		}, nil
	}

	report, err := populateMetaData()
	if err != nil {
		return report, err
	}

	report.Status = status.String()
	report.Name = "master"
	report.RootUUID = env.GetSystemUUID()
	report.DisplayName = hostName(report.Name, env.GetHostname())
	report.Type = "phpfpm"
	report.Agent = agent
	report.Pools, err = ParsePools(report.RootUUID, agent, report.LocalId, report.Version, env.GetHostname())
	if err != nil {
		return report, err
	}

	return report, nil
}

func populateMetaData() (*PhpFpmMetaData, error) {
	report := &PhpFpmMetaData{}
	ps, err := exec.Command("bash", "-c", "ps xao pid,ppid,command | grep 'php-fpm[:]'").Output()
	if err != nil {
		return report, fmt.Errorf("failed to retrieve ps info about php-fpm: %v", err)
	}

	psSplit := strings.Split(string(ps), "\n")
	var workers int
	for _, l := range psSplit {
		if len(l) == 0 {
			continue
		}

		parsed := processParse(l)
		pid, cmd := parsed[1], parsed[3]
		// master info, otherwise a pool worker
		// Assumption : There will be only 1 master running.
		if strings.Contains(cmd, "master process") {
			report.ConfPath = masterProcess(cmd)
			pidAsInt, err := strconv.Atoi(pid)
			if err != nil {
				return report, fmt.Errorf("failed to convert pid string %s to int: %v", pid, err)
			}
			report.Pid = int32(pidAsInt)
		} else {
			workers++
		}
	}
	report.NumWorkers = int32(workers)

	// get bin from master proc.
	// Todo : Architecture based..
	binCmd := fmt.Sprintf("/proc/%d/exe", report.Pid)
	// TODO : get rid of sudo here.
	output, err := exec.Command("sudo", "ls", "-la", binCmd).Output()
	if err != nil {
		return report, fmt.Errorf("failed to retrieve bin from proc: %v", err)
	}

	binFields := strings.Fields(string(output))
	l := len(binFields) - 1
	report.BinPath = binFields[l]

	// ensure the conf path is located in /etc/php
	confPathSplit := strings.Split(report.ConfPath, "/")
	if len(confPathSplit) < 3 && confPathSplit[0] != "etc" && confPathSplit[1] != "php" {
		return report, fmt.Errorf("conf path was not located within /etc/php/. Do you have a non-standard install?")
	}
	report.Version = confPathSplit[3]

	report.Cmd = report.ConfPath
	report.LocalId = core.GenerateNginxID("%s_%s_%s", report.BinPath, report.ConfPath, report.Cmd)

	return report, nil
}
