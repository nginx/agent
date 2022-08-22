# NGINX Agent Install tests

## Set up
To run the install test you'll need to copy over the agent package to your /tmp directory and assign environment variable AGENT_PACKAGE_FILE to it's location.

```
cp <agent_package> /tmp/
export AGENT_PACKAGE_FILE=/tmp/<agent_package>
```

## Run the tests
To run the test. You'll need to be in the agent project root directory.

```
cd /path/to/agent_project
go test -v ./test/install
```

Alternatively you can use the Makefile command in the project directory.

```
make test-install
```
 