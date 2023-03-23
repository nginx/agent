package php_fpm

type PhpFpmPool struct {
	Type            string
	LocalId         string
	RootUUID        string
	Name            string
	DisplayName     string
	Listen          string
	Flisten         string
	StatusPath      string
	CanHaveChildren bool
	Agent           string
	ParentLocalId   string
	Includes        []string
}

type PhpFpmMetaData struct {
	Type        string
	LocalId     string
	RootUUID    string
	Cmd         string
	Name        string
	DisplayName string
	ConfPath    string
	NumWorkers  int32
	Version     string
	Pools       []*PhpFpmPool
	Status      string
	Pid         int32
	BinPath     string
	Agent       string
}
type PhpFpmReport struct {
	Type        string
	LocalId     string
	RootUUID    string
	Name        string
	DisplayName string
	Cmd         string
	ConfPath    string
	Workers     int
	BinPath     string
	Version     string
	VersionLine string
	Pid         int
	Agent       string
	Children    []*PhpFpmPool
	Status      Status
}

type Config struct {
	// Unix socket address on which PhpFpmMetrics should listen for incoming metrics
	Address string
}
