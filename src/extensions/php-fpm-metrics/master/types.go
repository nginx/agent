package master

import "github.com/nginx/agent/v2/src/extensions/php-fpm-metrics/pool/worker"

type MetaData struct {
	Type        string
	LocalId     string
	Uuid        string
	Cmd         string
	Name        string
	DisplayName string
	ConfPath    string
	NumWorkers  int32
	Version     string
	VersionLine string
	Pools       []*worker.MetaData
	Status      string
	Pid         int32
	BinPath     string
	Agent       string
}
