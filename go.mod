module github.com/nginx/agent/v3

go 1.22

toolchain go1.22.2

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.33.0-20240401165935-b983156c5e99.1
	github.com/bufbuild/protovalidate-go v0.2.1
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/docker/docker v26.1.4+incompatible
	github.com/go-resty/resty/v2 v2.12.0
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c
	github.com/nginxinc/nginx-plus-go-client v1.2.2
	github.com/nginxinc/nginx-prometheus-exporter v1.3.0
	github.com/nxadm/tail v1.4.11
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/exceptionsconnector v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/grafanacloudconnector v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/servicegraphconnector v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/ackextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/asapauthextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/awsproxy v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/httpforwarderextension v0.100.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/oauth2clientauthextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/dockerobserver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/hostobserver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/k8sobserver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/oidcauthextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/sigv4authextension v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden v0.105.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest v0.105.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza v0.104.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/cumulativetodeltaprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbyattrsprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricsgenerationprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor v0.100.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/remotetapprocessor v0.100.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/routingprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/bigipreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dockerstatsreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/httpcheckreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sclusterreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8seventsreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sobjectsreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kubeletstatsreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/podmanreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/simpleprometheusreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/syslogreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/tcplogreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/udplogreceiver v0.102.0
	github.com/open-telemetry/opentelemetry-collector-contrib/testbed v0.102.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	github.com/testcontainers/testcontainers-go v0.31.0
	github.com/trivago/grok v1.0.0
	go.opentelemetry.io/collector/component v0.105.0
	go.opentelemetry.io/collector/config/confighttp v0.104.0
	go.opentelemetry.io/collector/config/configtelemetry v0.105.0
	go.opentelemetry.io/collector/config/configtls v1.12.0
	go.opentelemetry.io/collector/confmap v0.104.0
	go.opentelemetry.io/collector/confmap/converter/expandconverter v0.104.0
	go.opentelemetry.io/collector/confmap/provider/envprovider v0.104.0
	go.opentelemetry.io/collector/confmap/provider/fileprovider v0.104.0
	go.opentelemetry.io/collector/confmap/provider/httpprovider v0.104.0
	go.opentelemetry.io/collector/confmap/provider/httpsprovider v0.104.0
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v0.104.0
	go.opentelemetry.io/collector/connector v0.102.0
	go.opentelemetry.io/collector/connector/forwardconnector v0.102.0
	go.opentelemetry.io/collector/consumer v0.105.0
	go.opentelemetry.io/collector/exporter v0.102.0
	go.opentelemetry.io/collector/exporter/debugexporter v0.102.0
	go.opentelemetry.io/collector/exporter/otlpexporter v0.102.0
	go.opentelemetry.io/collector/exporter/otlphttpexporter v0.102.0
	go.opentelemetry.io/collector/extension v0.104.0
	go.opentelemetry.io/collector/otelcol v0.102.0
	go.opentelemetry.io/collector/pdata v1.12.0
	go.opentelemetry.io/collector/processor v0.103.0
	go.opentelemetry.io/collector/processor/batchprocessor v0.102.0
	go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.102.0
	go.opentelemetry.io/collector/receiver v0.104.0
	go.opentelemetry.io/collector/receiver/nopreceiver v0.104.0
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.104.0
	go.opentelemetry.io/otel v1.28.0
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.0
	golang.org/x/mod v0.19.0
	golang.org/x/sync v0.7.0
	google.golang.org/protobuf v1.34.2
)

require (
	bitbucket.org/atlassian/go-asap/v2 v2.8.0 // indirect
	cloud.google.com/go/auth v0.4.2 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.11.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.5.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.6.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5 v5.5.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v4 v4.3.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources v1.2.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/Code-Hex/go-generics-cache v1.3.1 // indirect
	github.com/GehirnInc/crypt v0.0.0-20200316065508-bb7000b8a962 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.23.0 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Microsoft/hcsshim v0.12.2 // indirect
	github.com/SermoDigital/jose v0.9.2-0.20180104203859-803625baeddc // indirect
	github.com/Showmax/go-fqdn v1.0.0 // indirect
	github.com/alecthomas/participle/v2 v2.1.1 // indirect
	github.com/alecthomas/units v0.0.0-20231202071711-9a357b53e9c9 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230512164433-5d1fd1a340c9 // indirect
	github.com/apache/thrift v0.20.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/aws/aws-sdk-go v1.53.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.27.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.27.16 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.7 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.20.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.24.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.28.10 // indirect
	github.com/aws/smithy-go v1.20.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.6.1 // indirect
	github.com/bytedance/sonic v1.11.3 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.1 // indirect
	github.com/cncf/xds/go v0.0.0-20240423153145-555b57ec207b // indirect
	github.com/containerd/containerd v1.7.15 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/coreos/go-oidc/v3 v3.10.0 // indirect
	github.com/cpuguy83/dockercfg v0.3.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/digitalocean/godo v1.109.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/envoyproxy/go-control-plane v0.12.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/expr-lang/expr v1.16.9 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-jose/go-jose/v4 v4.0.1 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.20.2 // indirect
	github.com/go-openapi/jsonreference v0.20.4 // indirect
	github.com/go-openapi/swag v0.22.9 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.19.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.0.0 // indirect
	github.com/go-zookeeper/zk v1.0.3 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/cel-go v0.17.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.4 // indirect
	github.com/gophercloud/gophercloud v1.8.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/grafana/regexp v0.0.0-20221122212121-6b5c0a4cb7fd // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/hashicorp/consul/api v1.28.3 // indirect
	github.com/hashicorp/cronexpr v1.1.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.4 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/nomad/api v0.0.0-20240306004928-3e7191ccb702 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/hetznercloud/hcloud-go/v2 v2.6.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/ionos-cloud/sdk-go/v6 v6.1.11 // indirect
	github.com/jaegertracing/jaeger v1.57.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jstemmer/go-junit-report v1.0.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.1 // indirect
	github.com/kolo/xmlrpc v0.0.0-20220921171641-a4b6fa1dd06b // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-syslog/v4 v4.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/leodido/ragel-machinery v0.0.0-20190525184631-5f46317e436b // indirect
	github.com/leoluk/perflib_exporter v0.2.1 // indirect
	github.com/lightstep/go-expohisto v1.0.0 // indirect
	github.com/linode/linodego v1.33.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20240513124658-fba389f38bae // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.8.1 // indirect
	github.com/miekg/dns v1.1.58 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mostynb/go-grpc-compression v1.2.3 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opencensusexporter v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/syslogexporter v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/proxy v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.104.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.104.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/docker v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/kubelet v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/metadataproviders v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpersignal v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/experimentalmetricmetadata v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.105.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheus v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver v0.102.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver v0.102.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/openshift/api v3.9.0+incompatible // indirect
	github.com/openshift/client-go v0.0.0-20210521082421-73d9475a9142 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/ovh/go-ovh v1.4.3 // indirect
	github.com/pelletier/go-toml/v2 v2.2.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/pquerna/cachecontrol v0.1.0 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/common/sigv4 v0.1.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/prometheus/prometheus v0.51.2-0.20240405174432-b4a973753c6e // indirect
	github.com/rs/cors v1.11.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.25 // indirect
	github.com/shirou/gopsutil/v4 v4.24.6 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tg123/go-htpasswd v1.2.2 // indirect
	github.com/tidwall/gjson v1.14.3 // indirect
	github.com/tilinna/clock v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/vincent-petithory/dataurl v1.0.0 // indirect
	github.com/vultr/govultr/v2 v2.17.2 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/collector v0.104.0 // indirect
	go.opentelemetry.io/collector/config/configauth v0.104.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.11.0 // indirect
	go.opentelemetry.io/collector/config/configgrpc v0.104.0 // indirect
	go.opentelemetry.io/collector/config/confignet v0.104.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.12.0 // indirect
	go.opentelemetry.io/collector/config/configretry v0.102.0 // indirect
	go.opentelemetry.io/collector/config/internal v0.104.0 // indirect
	go.opentelemetry.io/collector/extension/auth v0.104.0 // indirect
	go.opentelemetry.io/collector/extension/ballastextension v0.102.0 // indirect
	go.opentelemetry.io/collector/extension/zpagesextension v0.102.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.11.0 // indirect
	go.opentelemetry.io/collector/filter v0.104.0 // indirect
	go.opentelemetry.io/collector/semconv v0.105.0 // indirect
	go.opentelemetry.io/collector/service v0.102.0 // indirect
	go.opentelemetry.io/contrib/config v0.8.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.52.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.52.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.27.0 // indirect
	go.opentelemetry.io/contrib/zpages v0.52.0 // indirect
	go.opentelemetry.io/otel/bridge/opencensus v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.4.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.50.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.28.0 // indirect
	go.opentelemetry.io/otel/log v0.4.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/sdk v1.28.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.4.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.7.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/term v0.22.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.23.0 // indirect
	gonum.org/v1/gonum v0.15.0 // indirect
	google.golang.org/api v0.182.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240711142825-46eb208f015d // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gotest.tools/v3 v3.5.1 // indirect
	k8s.io/api v0.29.3 // indirect
	k8s.io/apimachinery v0.29.3 // indirect
	k8s.io/client-go v0.29.3 // indirect
	k8s.io/klog/v2 v2.120.1 // indirect
	k8s.io/kube-openapi v0.0.0-20231010175941-2dd684a91f00 // indirect
	k8s.io/kubelet v0.29.3 // indirect
	k8s.io/utils v0.0.0-20240502163921-fe8a2dddb1d0 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/nginxinc/nginx-go-crossplane v0.4.46
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/samber/slog-gin v1.11.0
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/spf13/cobra v1.8.0
	github.com/spf13/viper v1.18.2
	github.com/vardius/message-bus v1.1.5
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.28.0
	golang.org/x/crypto v0.25.0 // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/grpc v1.65.0
)
