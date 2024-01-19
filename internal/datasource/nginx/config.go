package nginx

import (
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/nginx/agent/v3/internal/datasource"
	"github.com/nginx/agent/v3/internal/datasource/os"

	"github.com/google/uuid"
)

type NginxConfig struct {
	instanceId   uuid.UUID
	configWriter datasource.ConfigWriter
}

func (nc NginxConfig) Write(previousFileCache os.FileCache, filesUrl string, tenantID uuid.UUID) (currentFileCache os.FileCache, skippedFiles map[string]struct{}, err error) {
	return nc.configWriter.Write(previousFileCache, filesUrl, tenantID)
}

func NewNginxConfig(insanceId uuid.UUID, configWriter datasource.ConfigWriter) NginxConfig {
	return NginxConfig{
		instanceId:   insanceId,
		configWriter: configWriter,
	}
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
