package utils

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
)

// StartFakeProcesses creates a fake process for each of the string names and
// each fake process lasts for fakeProcsDuration of time (seconds), the
// function that is returned can be ran to kill all the fake processes that
// were created.
func StartFakeProcesses(names []string, fakeProcsDuration string) func() {
	pList := make([]*os.Process, 0)
	for _, name := range names {
		pCmd := exec.Command("bash", "-c", fmt.Sprintf("exec -a %s sleep %s", name, fakeProcsDuration))
		_ = pCmd.Start()

		// Arbitrary sleep to ensure process has time to come up
		time.Sleep(time.Millisecond * 150)

		pList = append(pList, pCmd.Process)
	}

	return func() {
		for _, p := range pList {
			_ = p.Kill()
		}
	}
}

func GetProcessMap() map[string][]*proto.NginxDetails {
	return map[string][]*proto.NginxDetails{
		"12345": {
			{
				ProcessId: "1",
			},
		},
	}
}