module github.com/nginx/agent/v3

go 1.24.2

toolchain go1.24.11

require (
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.9-20250912141014-52f32327d4b0.1
	buf.build/go/protovalidate v1.0.0
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/docker/docker v28.5.2+incompatible
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-resty/resty/v2 v2.16.5
	github.com/goccy/go-yaml v1.18.0
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3
	github.com/leodido/go-syslog/v4 v4.3.0
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c
	github.com/nginx/nginx-plus-go-client/v3 v3.0.1
	github.com/nginxinc/nginx-prometheus-exporter v1.3.0
	github.com/nxadm/tail v1.4.11
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/tcplogreceiver v0.141.0
	github.com/open-telemetry/opentelemetry-collector-contrib/testbed v0.141.0
	github.com/prometheus/client_model v0.6.2
	github.com/prometheus/common v0.67.4
	github.com/shirou/gopsutil/v4 v4.25.11
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.11.1
	github.com/testcontainers/testcontainers-go v0.40.0
	github.com/trivago/grok v1.0.0
	go.opentelemetry.io/collector/component v1.49.0
	go.opentelemetry.io/collector/component/componenttest v0.143.0
	go.opentelemetry.io/collector/config/confighttp v0.141.0
	go.opentelemetry.io/collector/confmap v1.49.0
	go.opentelemetry.io/collector/confmap/provider/envprovider v1.47.0
	go.opentelemetry.io/collector/confmap/provider/fileprovider v1.47.0
	go.opentelemetry.io/collector/confmap/provider/httpprovider v1.47.0
	go.opentelemetry.io/collector/confmap/provider/httpsprovider v1.47.0
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.47.0
	go.opentelemetry.io/collector/connector v0.141.0
	go.opentelemetry.io/collector/consumer v1.49.0
	go.opentelemetry.io/collector/consumer/consumertest v0.143.0
	go.opentelemetry.io/collector/exporter v1.49.0
	go.opentelemetry.io/collector/exporter/debugexporter v0.141.0
	go.opentelemetry.io/collector/exporter/otlpexporter v0.141.0
	go.opentelemetry.io/collector/exporter/otlphttpexporter v0.141.0
	go.opentelemetry.io/collector/extension v1.49.0
	go.opentelemetry.io/collector/extension/extensionauth v1.47.0
	go.opentelemetry.io/collector/extension/xextension v0.143.0
	go.opentelemetry.io/collector/filter v0.141.0
	go.opentelemetry.io/collector/otelcol v0.141.0
	go.opentelemetry.io/collector/pdata v1.49.0
	go.opentelemetry.io/collector/processor v1.47.0
	go.opentelemetry.io/collector/processor/batchprocessor v0.141.0
	go.opentelemetry.io/collector/processor/memorylimiterprocessor v0.141.0
	go.opentelemetry.io/collector/processor/processortest v0.141.0
	go.opentelemetry.io/collector/receiver v1.49.0
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.141.0
	go.opentelemetry.io/collector/receiver/receivertest v0.143.0
	go.opentelemetry.io/collector/scraper v0.141.0
	go.opentelemetry.io/collector/scraper/scraperhelper v0.141.0
	go.opentelemetry.io/collector/scraper/scrapertest v0.141.0
	go.opentelemetry.io/collector/service v0.141.0
	go.opentelemetry.io/otel v1.39.0
	go.uber.org/goleak v1.3.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.1
	golang.org/x/mod v0.30.0
	golang.org/x/sync v0.19.0
	google.golang.org/protobuf v1.36.11
)

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go/auth v0.16.5 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	dario.cat/mergo v1.0.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.19.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.12.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.5.0 // indirect
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.73.0-rc.9 // indirect
	github.com/DataDog/datadog-go/v5 v5.8.1 // indirect
	github.com/DataDog/go-sqllexer v0.1.9 // indirect
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/alecthomas/participle/v2 v2.1.4 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/antchfx/xmlquery v1.5.0 // indirect
	github.com/antchfx/xpath v1.3.5 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apache/arrow-go/v18 v18.2.0 // indirect
	github.com/apache/thrift v0.22.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.40.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.1 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.1 // indirect
	github.com/aws/smithy-go v1.23.2 // indirect
	github.com/axiomhq/hyperloglog v0.2.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.9.1 // indirect
	github.com/bytedance/sonic v1.14.0 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/dockercfg v0.3.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dennwc/varint v1.0.0 // indirect
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/ebitengine/purego v0.9.1 // indirect
	github.com/elastic/go-grok v0.3.1 // indirect
	github.com/elastic/lunes v0.2.0 // indirect
	github.com/expr-lang/expr v1.17.7 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/foxboron/go-tpm-keyfiles v0.0.0-20250903184740-5d135037bd4d // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/cel-go v0.26.1 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/go-tpm v0.9.7 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grafana/clusterurl v0.2.1 // indirect
	github.com/grafana/regexp v0.0.0-20250905093917-f7b3be9d1853 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jaegertracing/jaeger-idl v0.6.0 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/jstemmer/go-junit-report v1.0.0 // indirect
	github.com/kamstrup/intmap v0.5.1 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/knadh/koanf/maps v0.1.2 // indirect
	github.com/knadh/koanf/providers/confmap v1.0.0 // indirect
	github.com/knadh/koanf/v2 v2.3.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/leodido/ragel-machinery v0.0.0-20190525184631-5f46317e436b // indirect
	github.com/lightstep/go-expohisto v1.0.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20250317134145-8bc96cf8fc35 // indirect
	github.com/magefile/mage v1.15.0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.1.0 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mostynb/go-grpc-compression v1.2.3 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/stefexporter v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/syslogexporter v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/common v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/filter v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/gopsutilenv v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/grpcutil v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/otelarrow v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/pdatautil v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/sharedcomponent v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/core/xidutils v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/experimentalmetricmetadata v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/resourcetotelemetry v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/prometheus v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/pkg/winperfcounters v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/otelarrowreceiver v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/stefreceiver v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/syslogreceiver v0.141.0 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver v0.141.0 // indirect
	github.com/open-telemetry/otel-arrow/go v0.45.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/outcaste-io/ristretto v0.2.3 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/otlptranslator v1.0.0 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/prometheus/prometheus v0.307.3 // indirect
	github.com/prometheus/sigv4 v0.2.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/splunk/stef/go/grpc v0.0.8 // indirect
	github.com/splunk/stef/go/otel v0.0.8 // indirect
	github.com/splunk/stef/go/pdata v0.0.8 // indirect
	github.com/splunk/stef/go/pkg v0.0.8 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tilinna/clock v1.1.0 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/trivago/tgo v1.0.7 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/ua-parser/uap-go v0.0.0-20240611065828-3a4781585db6 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	github.com/valyala/fastjson v1.6.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/collector v0.141.0 // indirect
	go.opentelemetry.io/collector/client v1.49.0 // indirect
	go.opentelemetry.io/collector/component/componentstatus v0.141.0 // indirect
	go.opentelemetry.io/collector/config/configauth v1.47.0 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.47.0 // indirect
	go.opentelemetry.io/collector/config/configgrpc v0.141.0 // indirect
	go.opentelemetry.io/collector/config/configmiddleware v1.47.0 // indirect
	go.opentelemetry.io/collector/config/confignet v1.47.0 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.47.0 // indirect
	go.opentelemetry.io/collector/config/configoptional v1.49.0 // indirect
	go.opentelemetry.io/collector/config/configretry v1.49.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.141.0 // indirect
	go.opentelemetry.io/collector/config/configtls v1.47.0 // indirect
	go.opentelemetry.io/collector/confmap/xconfmap v0.143.0 // indirect
	go.opentelemetry.io/collector/connector/connectortest v0.141.0 // indirect
	go.opentelemetry.io/collector/connector/xconnector v0.141.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror v0.143.0 // indirect
	go.opentelemetry.io/collector/consumer/consumererror/xconsumererror v0.141.0 // indirect
	go.opentelemetry.io/collector/consumer/xconsumer v0.143.0 // indirect
	go.opentelemetry.io/collector/exporter/exporterhelper v0.143.0 // indirect
	go.opentelemetry.io/collector/exporter/exporterhelper/xexporterhelper v0.141.0 // indirect
	go.opentelemetry.io/collector/exporter/exportertest v0.143.0 // indirect
	go.opentelemetry.io/collector/exporter/xexporter v0.143.0 // indirect
	go.opentelemetry.io/collector/extension/extensioncapabilities v0.141.0 // indirect
	go.opentelemetry.io/collector/extension/extensionmiddleware v0.141.0 // indirect
	go.opentelemetry.io/collector/extension/extensiontest v0.143.0 // indirect
	go.opentelemetry.io/collector/extension/zpagesextension v0.141.0 // indirect
	go.opentelemetry.io/collector/featuregate v1.49.0 // indirect
	go.opentelemetry.io/collector/internal/fanoutconsumer v0.141.0 // indirect
	go.opentelemetry.io/collector/internal/memorylimiter v0.141.0 // indirect
	go.opentelemetry.io/collector/internal/sharedcomponent v0.141.0 // indirect
	go.opentelemetry.io/collector/internal/telemetry v0.141.0 // indirect
	go.opentelemetry.io/collector/pdata/pprofile v0.143.0 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.143.0 // indirect
	go.opentelemetry.io/collector/pdata/xpdata v0.143.0 // indirect
	go.opentelemetry.io/collector/pipeline v1.49.0 // indirect
	go.opentelemetry.io/collector/pipeline/xpipeline v0.141.0 // indirect
	go.opentelemetry.io/collector/processor/processorhelper v0.141.0 // indirect
	go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper v0.141.0 // indirect
	go.opentelemetry.io/collector/processor/xprocessor v0.141.0 // indirect
	go.opentelemetry.io/collector/receiver/receiverhelper v0.141.0 // indirect
	go.opentelemetry.io/collector/receiver/xreceiver v0.143.0 // indirect
	go.opentelemetry.io/collector/semconv v0.128.1-0.20250610090210-188191247685 // indirect
	go.opentelemetry.io/collector/service/hostcapabilities v0.141.0 // indirect
	go.opentelemetry.io/contrib/bridges/otelzap v0.13.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/contrib/otelconf v0.18.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.38.0 // indirect
	go.opentelemetry.io/contrib/zpages v0.63.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.60.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.14.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.38.0 // indirect
	go.opentelemetry.io/otel/log v0.14.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.14.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/arch v0.20.0 // indirect
	golang.org/x/exp v0.0.0-20251009144603-d2f985daa21b // indirect
	golang.org/x/oauth2 v0.32.0 // indirect
	golang.org/x/telemetry v0.0.0-20251008203120-078029d740a8 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	google.golang.org/api v0.250.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apimachinery v0.34.2 // indirect
	k8s.io/client-go v0.34.2 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/utils v0.0.0-20250604170112-4c0f3b243397 // indirect
	modernc.org/b/v2 v2.1.0 // indirect
)

require (
	github.com/gin-gonic/gin v1.10.1
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/nginxinc/nginx-go-crossplane v0.4.84
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/samber/slog-gin v1.17.2
	github.com/spf13/cobra v1.10.2
	github.com/spf13/viper v1.21.0
	github.com/vardius/message-bus v1.1.5
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.39.0
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/grpc v1.78.0
)
