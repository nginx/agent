package core

type Info struct {
	name    *string
	version *string
}

func NewInfo(name string, version string) *Info {
	info := new(Info)
	info.name = &name
	info.version = &version
	return info
}

func (info *Info) Name() string {
	return *info.name
}

func (info *Info) Version() string {
	return *info.version
}
