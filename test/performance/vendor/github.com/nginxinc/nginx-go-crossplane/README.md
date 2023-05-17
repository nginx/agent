# nginx-go-crossplane
A Go port of the NGINX config/JSON converter [crossplane](https://github.com/nginxinc/crossplane).

## Parse
This is an example that takes a path to an NGINX config file, converts it to JSON, and prints the result to stdout.
```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nginxinc/nginx-go-crossplane"
)

func main() {
	path := os.Args[1]

	payload, err := crossplane.Parse(path, &crossplane.ParseOptions{})
	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(b))
}
```

## Build
This is an example that takes a path to a JSON file, converts it to an NGINX config, and prints the result to stdout.
```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nginxinc/nginx-go-crossplane"
)

func main() {
	path := os.Args[1]

	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	content, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var payload crossplane.Payload
	if err = json.Unmarshal(content, &payload); err != nil {
		panic(err)
	}

	var buf bytes.Buffer
	if err = crossplane.Build(&buf, payload.Config[0], &crossplane.BuildOptions{}); err != nil {
		panic(err)
	}

	fmt.Println(buf.String())
}
```

## Contributing

If you'd like to contribute to the project, please read our [Contributing guide](CONTRIBUTING.md).

## License

[Apache License, Version 2.0](https://github.com/nginxinc/nginx-go-crossplane/blob/main/LICENSE)

&copy; [F5 Networks, Inc.](https://www.f5.com/) 2022
