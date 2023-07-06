package nginx

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	NGINXBin           = "/usr/sbin/nginx"
	NGINXStdOutLogFile = "./nginx-out.log"
	NGINXStdErrLogFile = "./nginx-err.log"
	ConfigFile         = "/etc/nginx/nginx.conf"
)

type NginxCommand struct {
	conf *NginxConf

	nginxCommand *exec.Cmd
	err          *os.File
	out          *os.File
}

func NewNginxCommand(conf *NginxConf) (*NginxCommand, error) {
	cfg, err := conf.Build()
	if err != nil {
		return nil, fmt.Errorf("config build failed: %w", err)
	}
	logrus.Debugf("Generating config: %s", string(cfg))

	err = os.WriteFile(ConfigFile, []byte(cfg), 0o666)
	if err != nil {
		return nil, fmt.Errorf("config write failed: %w", err)
	}
	cmd := exec.Command(NGINXBin, "-g", `daemon off;`)

	fd_out, err := os.Create(NGINXStdOutLogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %s: %w", NGINXStdOutLogFile, err)
	}
	cmd.Stdout = fd_out

	fd_err, err := os.Create(NGINXStdErrLogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %s: %w", NGINXStdErrLogFile, err)
	}
	cmd.Stderr = fd_err

	return &NginxCommand{
		conf:         conf,
		nginxCommand: cmd,
		err:          fd_err,
		out:          fd_out,
	}, nil
}

func (c *NginxCommand) Start(t *testing.T) {
	err := c.nginxCommand.Start()
	require.NoError(t, err)

	time.Sleep(time.Second * 2)

	t.Cleanup(func() {
		err = exec.Command(NGINXBin, "-s", "quit").Run()
		assert.NoError(t, err)

		err := c.nginxCommand.Wait()
		assert.NoError(t, err)

		c.err.Close()
		c.out.Close()

		data, err := os.ReadFile(NGINXStdErrLogFile)
		assert.NoError(t, err)
		logrus.Debugf("Nginx error logs:%s", string(data))

		data, err = os.ReadFile(NGINXStdOutLogFile)
		assert.NoError(t, err)
		logrus.Debugf("Nginx out logs:%s", string(data))
	})
}
