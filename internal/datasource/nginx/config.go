package nginx

import (
	"fmt"
	"log/slog"
	"os/exec"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.7.0 -generate
//counterfeiter:generate -o mock_nginx_config.go . DataplaneConfigInterface
//go:generate sh -c "grep -v github.com/nginx/agent/v3/internal/datasource/nginx mock_nginx_config.go | sed -e s\\/nginx\\\\.\\/\\/g > mock_nginx_config_fixed.go"
//go:generate mv mock_nginx_config_fixed.go mock_nginx_config.go
type DataplaneConfigInterface interface {
	Validate() error
	Reload() error
}

type NginxConfig struct{}

func NewNginxConfig() *NginxConfig {
	return &NginxConfig{}
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
