# Examples

## This folder contains examples of how to set up a control plane server to communicate using this sdk

To run the server example from this context:
```
go run server.go

INFO[0000] http listening                    
INFO[0000] grpc listening
```

The above command should show the following output in the application:
INFO[0000] http listening                       
INFO[0000] grpc listening    

Configuring the Agent to run against the example server
### Agent Settings
```yaml
server:
  # host of the control plane
  host: 127.0.0.1
  grpcPort: 54789
# tls options - NOT RECOMMENDED FOR PRODUCTION
tls:
  enable: false
  skip_verify: true
```

### Viewing the HTTP Static content
[HTTP Server Webpage](http://localhost:54790/)

