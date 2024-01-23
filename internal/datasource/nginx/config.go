package nginx

import (
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/nginx/agent/v3/internal/datasource/config"
	"github.com/nginx/agent/v3/internal/datasource/os"

	"github.com/google/uuid"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_nginx_config.go . NginxConfigInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource/nginx mock_nginx_config.go | sed -e s\\/nginx\\\\.\\/\\/g > mock_nginx_config_fixed.go"
//go:generate mv mock_nginx_config_fixed.go mock_nginx_config.go
type NginxConfigInterface interface {
	Write(previousFileCache os.FileCache, filesUrl string, tenantID uuid.UUID) (currentFileCache os.FileCache, skippedFiles map[string]struct{}, err error)
	Validate() error
	Reload() error
}

type NginxConfig struct {
	instanceId   string
	configWriter config.ConfigWriter
}

func NewNginxConfig(instanceId string, configWriter config.ConfigWriter) NginxConfig {
	return NginxConfig{
		instanceId:   instanceId,
		configWriter: configWriter,
	}
}

func (nc NginxConfig) Write(previousFileCache os.FileCache, filesUrl string, tenantID uuid.UUID) (currentFileCache os.FileCache, skippedFiles map[string]struct{}, err error) {
	return nc.configWriter.Write(previousFileCache, filesUrl, tenantID)
}

func (nc NginxConfig) Validate() error {
	out, err := exec.Command("nginx", "-t").CombinedOutput()
	if err != nil {
		return fmt.Errorf("NGINX config test failed %w: %s", err, out)
	}
	slog.Info("NGINX config tested", "output", out)
	return nil
}

func (nc NginxConfig) Reload() error {
	out, err := exec.Command("nginx", "-s", "reload").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload NGINX %w: %s", err, out)
	}
	slog.Info("NGINX reloaded")

	return nil
}
