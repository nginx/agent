package pool

import (
	"os"

	log "github.com/sirupsen/logrus"
)

type Pool struct {
	dir string
}

func New(dir string) *Pool {
	return &Pool{
		dir: dir,
	}
}

// GetConfigs returns workers configuration in dir
func (p *Pool) GetConfigs(dir string) ([]string, error) {
	var files []string
	fileInfo, err := os.ReadDir(dir)
	if err != nil {
		log.Warnf("Unable to reading directory %s: %v ", dir, err)
		return nil, err
	}

	for _, file := range fileInfo {
		files = append(files, file.Name())
	}

	return files, nil
}
