module github.com/nginx/agent/v3

go 1.22.0

toolchain go1.22.5

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.33.0-20240401165935-b983156c5e99.1
	github.com/bufbuild/protovalidate-go v0.2.1
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/docker/docker v27.1.1+incompatible
	github.com/fsnotify/fsnotify v1.7.0
	github.com/go-resty/resty/v2 v2.13.1
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.1.0
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c
	github.com/nginxinc/nginx-plus-go-client v1.2.2
	github.com/nxadm/tail v1.4.11
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/cumulativetodeltaprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbyattrsprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricsgenerationprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/remotetapprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/routingprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/nginxreceiver v0.108.0
	github.com/open-telemetry/opentelemetry-collector-contrib/testbed v0.108.0
	github.com/shirou/gopsutil/v4 v4.24.7
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	github.com/testcontainers/testcontainers-go v0.33.0
	github.com/trivago/grok v1.0.0
	go.opentelemetry.io/collector/component v0.108.1
	go.opentelemetry.io/collector/config/confighttp v0.108.1
	go.opentelemetry.io/collector/confmap v1.14.1
	go.opentelemetry.io/collector/confmap/provider/envprovider v0.108.1
	go.opentelemetry.io/collector/confmap/provider/fileprovider v0.108.1
	go.opentelemetry.io/collector/confmap/provider/httpprovider v0.108.1
	go.opentelemetry.io/collector/confmap/provider/httpsprovider v0.106.1
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v0.108.1
	go.opentelemetry.io/collector/connector v0.108.1
	go.opentelemetry.io/collector/consumer v0.108.1
	go.opentelemetry.io/collector/consumer/consumertest v0.108.1
	go.opentelemetry.io/collector/exporter v0.108.1
	go.opentelemetry.io/collector/exporter/debugexporter v0.108.1
	go.opentelemetry.io/collector/exporter/otlpexporter v0.108.1
	go.opentelemetry.io/collector/exporter/otlphttpexporter v0.108.1
	go.opentelemetry.io/collector/extension v0.108.1
	go.opentelemetry.io/collector/otelcol v0.108.1
	go.opentelemetry.io/collector/pdata v1.14.1
	go.opentelemetry.io/collector/processor v0.108.1
	go.opentelemetry.io/collector/processor/batchprocessor v0.108.1
	go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.108.1
	go.opentelemetry.io/collector/receiver v0.108.1
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.108.1
	go.uber.org/goleak v1.3.0
	go.uber.org/zap v1.27.0
	golang.org/x/mod v0.19.0
	golang.org/x/sync v0.8.0
	google.golang.org/protobuf v1.34.2
)

require (
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.24.1 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Showmax/go-fqdn v1.0.0 // indirect
	github.com/alecthomas/participle/v2 v2.1.1 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr/v4 v4.0.0-20230512164433-5d1fd1a340c9 // indirect
	github.com/apache/thrift v0.20.0 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/aws/aws-sdk-go v1.55.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bytedance/sonic v1.11.3 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20230717121745-296ad89f973d // indirect
	github.com/chenzhuoyu/iasm v0.9.1 // indirect
	github.com/containerd/containerd v1.7.18 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/elastic/go-grok v0.3.1 // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/expr-lang/expr v1.16.9 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
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
	github.com/go-viper/mapstructure/v2 v2.1.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/cel-go v0.17.1 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20240711041743-f6c9dda6c6da // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/hashicorp/consul/api v1.29.4 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/serf v0.10.1 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jaegertracing/jaeger v1.60.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jonboulle/clockwork v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jstemmer/go-junit-report v1.0.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.1 // indirect
	github.com/leodido/go-syslog/v4 v4.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/leodido/ragel-machinery v0.0.0-20190525184631-5f46317e436b // indirect
	github.com/lightstep/go-expohisto v1.0.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20240513124658-fba389f38bae // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/maxbrunsfeld/counterfeiter/v6 v6.8.1 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
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
	github.com/nginxinc/nginx-prometheus-exporter v0.11.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opencensusexporter v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/syslogexporter v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/ecsutil v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/k8sconfig v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/metadataproviders v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/pdatautil v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/batchpersignal v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/experimentalmetricmetadata v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/sampling v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheus v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/syslogreceiver v0.108.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver v0.108.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/openshift/api v3.9.0+incompatible // indirect
	github.com/openshift/client-go v0.0.0-20210521082421-73d9475a9142 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus-community/windows_exporter v0.27.2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.55.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rs/cors v1.11.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/shoenig/test v1.7.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/soheilhy/cmux v0.1.5 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ua-parser/uap-go v0.0.0-20240611065828-3a4781585db6 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/collector v0.108.1 // indirect
	go.opentelemetry.io/collector/client v1.14.1 // indirect
	go.opentelemetry.io/collector/component/componentprofiles v0.108.1 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.108.1 // indirect
	go.opentelemetry.io/collector/config/configauth v0.108.1 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.14.1 // indirect
	go.opentelemetry.io/collector/config/configgrpc v0.108.1 // indirect
	go.opentelemetry.io/collector/config/confignet v0.108.1 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.14.1 // indirect
	go.opentelemetry.io/collector/config/configretry v1.14.1 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.108.1 // indirect
	go.opentelemetry.io/collector/config/configtls v1.14.1 // indirect
	go.opentelemetry.io/collector/config/internal v0.108.1 // indirect
	go.opentelemetry.io/collector/consumer/consumerprofiles v0.108.1 // indirect
	go.opentelemetry.io/collector/extension/auth v0.108.1 // indirect
	go.opentelemetry.io/collector/extension/zpagesextension v0.108.1 // indirect
	go.opentelemetry.io/collector/featuregate v1.14.1 // indirect
	go.opentelemetry.io/collector/filter v0.108.1 // indirect
	go.opentelemetry.io/collector/internal/globalgates v0.108.1 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.108.1 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.108.1 // indirect
	go.opentelemetry.io/collector/semconv v0.108.1 // indirect
	go.opentelemetry.io/collector/service v0.108.1 // indirect
	go.opentelemetry.io/contrib/config v0.8.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.28.0 // indirect
	go.opentelemetry.io/contrib/zpages v0.53.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
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
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.7.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/oauth2 v0.22.0 // indirect
	golang.org/x/term v0.23.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	golang.org/x/tools v0.23.0 // indirect
	gonum.org/v1/gonum v0.15.1 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240814211410-ddb44dafa142 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240814211410-ddb44dafa142 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.31.0 // indirect
	k8s.io/apimachinery v0.31.0 // indirect
	k8s.io/client-go v0.31.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20240228011516-70dd3763d340 // indirect
	k8s.io/utils v0.0.0-20240711033017-18e509b52bc8 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/nginxinc/nginx-go-crossplane v0.4.46
	github.com/prometheus/client_golang v1.20.2 // indirect
	github.com/samber/slog-gin v1.11.0
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/spf13/cobra v1.8.1
	github.com/spf13/viper v1.19.0
	github.com/vardius/message-bus v1.1.5
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.28.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	google.golang.org/grpc v1.65.0
)
