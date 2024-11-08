# Mock Management OTel Collector

There are 3 images that need to be built in order to use the mock management OTel collector
* Agent with NGINX Plus image
* Agent with NGINX OSS image
* Custom OTel collector image

To build these images run the following
```
make local-deb-package build-test-plus-image build-test-oss-image build-mock-collector-image
```

To start run everything run the following
```
make run-mock-management-otel-collector
```

Once everything is started there should be 5 containers running
```
8e6df6d0bc73  localhost/nginx_plus_agent_ubuntu_22.04:latest                        4 minutes ago  Up 4 minutes              80/tcp, 443/tcp                                 mock-collector-agent-with-nginx-plus
a65a7efaf2b3  localhost/nginx_oss_agent_ubuntu_22.04:latest                         4 minutes ago  Up 4 minutes              80/tcp, 443/tcp                                 mock-collector-agent-with-nginx-oss
bf0f247991c0  localhost/mock-collector:latest                 go run main.go        4 minutes ago  Up 4 minutes              0.0.0.0:4320->4317/tcp, 0.0.0.0:9775->9090/tcp  mock-collector-otel-collector
67bb7bde6392  docker.io/prom/prometheus:latest                --config.file=/et...  4 minutes ago  Up 4 minutes              0.0.0.0:9090->9090/tcp, 9090/tcp                mock-collector-prometheus
a83a997eb652  docker.io/grafana/grafana:latest                                      4 minutes ago  Up 4 minutes              0.0.0.0:3002->3000/tcp, 3000/tcp                mock-collector-grafana
```

To view the metrics, the grafana UI can be used by accessing this URL http://localhost:3002/login (Note: username/password is admin/admin)

To stop everything run the following
```
make stop-mock-management-otel-collector
```
