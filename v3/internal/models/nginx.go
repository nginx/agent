package models

type Nginx struct {
    version string
}

func NewNginx(version string) *Nginx {
    return &Nginx{version: version}
}