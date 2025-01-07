# Mock Management OTel Collector

There are 3 images that need to be built in order to use the mock management OTel collector
* Agent with NGINX Plus & NGINX App Protect image
* Agent with NGINX OSS image
* Custom OTel collector image

To build these images run the following
```
OSARCH=amd64 make local-deb-package build-test-plus-image
make local-deb-package build-test-oss-image build-mock-management-otel-collector-image
```

[**Note:** We need to build the test NGINX Plus image with the environment variable `OSARCH=amd64` since NGINX App Protect doesn't support ARM yet.]

To start run everything run the following
```
make run-mock-management-otel-collector
```

Once everything is started there should be 6 containers running
```
98d601aeb602  localhost/nginx_plus_agent_ubuntu_22.04:latest                        14 seconds ago  Up 14 seconds  80/tcp, 443/tcp                                 mock-collector-agent-with-nginx-plus
abaa9f70bfb6  localhost/nginx_oss_agent_ubuntu_22.04:latest                         12 seconds ago  Up 13 seconds  80/tcp, 443/tcp                                 mock-collector-agent-with-nginx-oss
7158c3d0035b  localhost/mock-collector:latest                 /mock-management-...  11 seconds ago  Up 11 seconds  0.0.0.0:4320->4317/tcp, 0.0.0.0:9775->9090/tcp  mock-collector-otel-collector
3f3a50c5fff4  docker.io/prom/prometheus:latest                --config.file=/et...  9 seconds ago   Up 9 seconds   0.0.0.0:9090->9090/tcp, 9090/tcp                mock-collector-prometheus
b4541072ef8b  docker.io/grafana/grafana:latest                                      7 seconds ago   Up 8 seconds   0.0.0.0:3002->3000/tcp, 3000/tcp                mock-collector-grafana
52b16ef06a10  docker.io/grafana/loki:latest                   -config.file=/etc...  6 seconds ago   Up 6 seconds   0.0.0.0:3100->3100/tcp, 3100/tcp                mock-collector-loki

```

To view the metrics & logs, the grafana UI can be used by accessing this URL http://localhost:3002/login (Note: username/password is admin/admin)

To stop everything run the following
```
make stop-mock-management-otel-collector
```
