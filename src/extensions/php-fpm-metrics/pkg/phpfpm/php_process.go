package phpfpm

type PhpProcess struct {
	Pid       int32
	Name      string
	IsMaster  bool
	ParentPid int32
	Command   string
	BinPath   string
}
