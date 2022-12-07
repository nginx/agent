module github.com/nginx/agent/test/integration

go 1.19

require (
	github.com/go-resty/resty/v2 v2.7.0
	github.com/nginx/agent/v2 v2.0.0-00010101000000-000000000000
	github.com/nginx/agent/v2/test/integration v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.0
	github.com/stretchr/testify v1.8.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/orcaman/concurrent-map v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	golang.org/x/net v0.0.0-20220906165146-f3363e06e74c // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/docker/cli => github.com/docker/cli v20.10.3-0.20221013132413-1d6c6e2367e2+incompatible // 22.06 master branch
	github.com/docker/docker => github.com/docker/docker v20.10.3-0.20221013203545-33ab36d6b304+incompatible // 22.06 branch
	github.com/jinzhu/gorm => github.com/jinzhu/gorm v1.9.16
	github.com/jinzhu/inflection => github.com/jinzhu/inflection v1.0.0
	github.com/moby/buildkit => github.com/moby/buildkit v0.10.1-0.20220816171719-55ba9d14360a // same as buildx
	github.com/nginx/agent/sdk/v2 => ./../../sdk
	github.com/nginx/agent/v2 => ./../../
	github.com/nginx/agent/v2/test/integration => ./

	github.com/opencontainers/runc => github.com/opencontainers/runc v1.1.2 // Can be removed on next bump of containerd to > 1.6.4
	// For k8s dependencies, we use a replace directive, to prevent them being
	// upgraded to the version specified in containerd, which is not relevant to the
	// version needed.
	// See https://github.com/docker/buildx/pull/948 for details.
	// https://github.com/docker/buildx/blob/v0.8.1/go.mod#L62-L64
	k8s.io/api => k8s.io/api v0.22.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.4
	k8s.io/client-go => k8s.io/client-go v0.22.4
)
