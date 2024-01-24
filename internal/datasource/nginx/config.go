package nginx

import (
	"fmt"
	"log/slog"
	"os/exec"

	config_writer "github.com/nginx/agent/v3/internal/datasource/config"

	"github.com/google/uuid"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_nginx_config.go . NginxConfigInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource/nginx mock_nginx_config.go | sed -e s\\/nginx\\\\.\\/\\/g > mock_nginx_config_fixed.go"
//go:generate mv mock_nginx_config_fixed.go mock_nginx_config.go
type NginxConfigInterface interface {
	Write(filesUrl string, tenantID uuid.UUID) (skippedFiles map[string]struct{}, err error)
	Validate() error
	Reload() error
}

type NginxConfigParameters struct {
	configWriter config_writer.ConfigWriterInterface
}

type NginxConfig struct {
	configWriter config_writer.ConfigWriterInterface
}

func NewNginxConfig(nginxConfigParameters NginxConfigParameters) *NginxConfig {
	if nginxConfigParameters.configWriter == nil {
		// TODO: Figure out where we are going to get the Cache Path From
		nginxConfigParameters.configWriter = config_writer.NewConfigWriter(&config_writer.ConfigWriterParameters{}, "")
	}

	return &NginxConfig{
		configWriter: nginxConfigParameters.configWriter,
	}
}

func (nc NginxConfig) Write(filesUrl string, tenantID uuid.UUID) (skippedFiles map[string]struct{}, err error) {
	skippedFiles, err = nc.configWriter.Write(filesUrl, tenantID)

	return skippedFiles, err
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
