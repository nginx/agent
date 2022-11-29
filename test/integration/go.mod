module github.com/nginx/agent/test/integration

go 1.19

require (
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
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4 // indirect
	golang.org/x/sys v0.0.0-20220804214406-8e32c043e418 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/nginx/agent/v2/test/integration => ./

replace github.com/nginx/agent/sdk/v2 => ./../../sdk

replace github.com/nginx/agent/v2 => ./../../
