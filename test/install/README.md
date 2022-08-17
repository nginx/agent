# NGINX Agent Install tests

## Set up
To run the install test you'll need to copy over the agent tarball to your VM and assign environment variable AGENT_PACKAGE_FILE to it's location.
```
cp /path/to/tarball/ /tmp/
export AGENT_PACKAGE="/[path/to/tarball"] 
```

## Run the tests
To run the test. You'll need to be in the root agent project directory while on your VM
```
cd /path/to/agent
go test -v ./test/install
```
Alternatively you can use the Makefile command in the project directory
```
make test-install
```
 