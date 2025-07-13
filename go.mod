module github.com/instill-ai/pipeline-backend

go 1.24.4

require (
	cloud.google.com/go/bigquery v1.69.0
	cloud.google.com/go/iam v1.5.2
	cloud.google.com/go/longrunning v0.6.7
	cloud.google.com/go/storage v1.54.0
	code.sajari.com/docconv v1.3.8
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/JohannesKaufmann/html-to-markdown v1.6.0
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/PuerkitoBio/goquery v1.10.3
	github.com/anthropics/anthropic-sdk-go v0.2.0-alpha.3
	github.com/belong-inc/go-hubspot v0.9.0
	github.com/chromedp/chromedp v0.13.6
	github.com/cohere-ai/cohere-go/v2 v2.14.1
	github.com/dslipak/pdf v0.0.2
	github.com/elastic/go-elasticsearch/v8 v8.18.0
	github.com/emersion/go-imap/v2 v2.0.0-beta.5
	github.com/emersion/go-message v0.18.2
	github.com/extrame/xls v0.0.1
	github.com/fogleman/gg v1.3.0
	github.com/frankban/quicktest v1.14.6
	github.com/gabriel-vasile/mimetype v1.4.9
	github.com/gage-technologies/mistral-go v1.1.0
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
	github.com/go-chi/chi/v5 v5.2.2
	github.com/go-openapi/strfmt v0.23.0
	github.com/go-redis/redismock/v9 v9.2.0
	github.com/go-resty/resty/v2 v2.16.5
	github.com/go-sql-driver/mysql v1.9.2
	github.com/gocolly/colly/v2 v2.2.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gogo/status v1.1.1
	github.com/gojuno/minimock/v3 v3.4.5
	github.com/golang-migrate/migrate/v4 v4.18.3
	github.com/google/go-cmp v0.7.0
	github.com/google/go-github/v62 v62.0.0
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.1
	github.com/h2non/filetype v1.1.3
	github.com/iancoleman/strcase v0.3.0
	github.com/influxdata/influxdb-client-go/v2 v2.14.0
	github.com/instill-ai/protogen-go v0.3.3-alpha.0.20250707160902-77023eb2f033
	github.com/instill-ai/usage-client v0.4.0
	github.com/instill-ai/x v0.8.0-alpha.0.20250713164718-3ad9822ddd60
	github.com/itchyny/gojq v0.12.17
	github.com/jackc/pgx/v5 v5.7.5
	github.com/jmoiron/sqlx v1.4.0
	github.com/json-iterator/go v1.1.12
	github.com/k3a/html2text v1.2.1
	github.com/knadh/koanf v1.5.0
	github.com/launchdarkly/go-semver v1.0.3
	github.com/lestrrat-go/jspointer v0.0.0-20181205001929-82fadba7561c
	github.com/lestrrat-go/jsref v0.0.0-20211028120858-c0bcbb5abf20
	github.com/lestrrat-go/option v1.0.1
	github.com/lestrrat-go/pdebug v0.0.0-20210111095411-35b07dbf089b
	github.com/lestrrat-go/structinfo v0.0.0-20210312050401-7f8bd69d6acb
	github.com/lib/pq v1.10.9
	github.com/mennanov/fieldmask-utils v1.1.2
	github.com/nakagami/firebirdsql v0.9.15
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/openfga/api/proto v0.0.0-20240318145204-66b9e5cb403c
	github.com/pdfcpu/pdfcpu v0.11.0
	github.com/pkoukk/tiktoken-go v0.1.7
	github.com/redis/go-redis/v9 v9.9.0
	github.com/sijms/go-ora v1.3.2
	github.com/slack-go/slack v0.17.0
	github.com/stretchr/testify v1.10.0
	github.com/tmc/langchaingo v0.1.13
	github.com/u2takey/ffmpeg-go v0.5.0
	github.com/warmans/ffmpeg-go v1.0.0
	github.com/weaviate/weaviate v1.26.0-rc.1
	github.com/weaviate/weaviate-go-client/v4 v4.15.0
	github.com/xuri/excelize/v2 v2.9.1
	github.com/xwb1989/sqlparser v0.0.0-20180606152119-120387863bf2
	github.com/zaf/resample v1.5.0
	go.einride.tech/aip v0.70.2
	go.mongodb.org/mongo-driver v1.17.3
	go.opentelemetry.io/contrib/propagators/b3 v1.36.0
	go.opentelemetry.io/otel v1.37.0
	go.temporal.io/api v1.49.1
	go.temporal.io/sdk v1.34.0
	go.temporal.io/sdk/contrib/opentelemetry v0.6.0
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6
	golang.org/x/image v0.27.0
	golang.org/x/mod v0.25.0
	golang.org/x/net v0.41.0
	golang.org/x/oauth2 v0.30.0
	golang.org/x/text v0.26.0
	google.golang.org/api v0.235.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
	gopkg.in/guregu/null.v4 v4.0.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/datatypes v1.2.5
	gorm.io/driver/postgres v1.6.0
	gorm.io/gorm v1.30.0
	gorm.io/plugin/dbresolver v1.6.0
)

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go/monitoring v1.24.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.27.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.51.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.51.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.2 // indirect
	github.com/cncf/xds/go v0.0.0-20250501225837-2ac532fd4443 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.32.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20250517221953-25912455fbc8 // indirect
	github.com/hhrutter/lzw v1.0.0 // indirect
	github.com/hhrutter/pkcs7 v0.2.0 // indirect
	github.com/hhrutter/tiff v1.0.2 // indirect
	github.com/minio/crc64nvme v1.0.2 // indirect
	github.com/minio/minio-go/v7 v7.0.92 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nakagami/chacha20 v0.1.0 // indirect
	github.com/nexus-rpc/sdk-go v0.4.0 // indirect
	github.com/nlnwa/whatwg-url v0.6.2 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.64.0 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/spiffe/go-spiffe/v2 v2.5.0 // indirect
	github.com/tiendc/go-deepcopy v1.6.0 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	github.com/twmb/murmur3 v1.1.8 // indirect
	github.com/uber-go/tally/v4 v4.1.17 // indirect
	github.com/zeebo/errs v1.4.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.36.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.13.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.13.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.37.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.37.0 // indirect
	go.opentelemetry.io/otel/log v0.13.0 // indirect
	go.opentelemetry.io/otel/sdk v1.37.0 // indirect
	go.opentelemetry.io/otel/sdk/log v0.13.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.opentelemetry.io/proto/otlp v1.7.0 // indirect
	go.temporal.io/sdk/contrib/tally v0.2.0 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

require (
	cloud.google.com/go v0.121.2 // indirect
	cloud.google.com/go/auth v0.16.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.7.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/JalfResi/justext v0.0.0-20221106200834-be571e3e3052 // indirect
	github.com/PaesslerAG/gval v1.2.4 // indirect
	github.com/advancedlogic/GoOse v0.0.0-20231203033844-ae6b36caf275 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/antchfx/htmlquery v1.3.4 // indirect
	github.com/antchfx/xmlquery v1.4.4 // indirect
	github.com/antchfx/xpath v1.3.4 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/apapsch/go-jsonmerge/v2 v2.0.0 // indirect
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.55.7 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/smithy-go v1.22.3 // indirect
	github.com/catalinc/hashcash v1.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chromedp/cdproto v0.0.0-20250527225801-8f9bc3ce9e31 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.7.0 // indirect
	github.com/emersion/go-sasl v0.0.0-20241020182733-b788ff22d5a6 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/extrame/ole2 v0.0.0-20160812065207-d69429661ad7 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/set v0.2.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gigawattio/window v0.0.0-20180317192513-0f5467e35573 // indirect
	github.com/go-audio/riff v1.0.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/analysis v0.23.0 // indirect
	github.com/go-openapi/errors v0.22.1 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/loads v0.22.0 // indirect
	github.com/go-openapi/spec v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-openapi/validate v0.24.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/influxdata/line-protocol v0.0.0-20210922203350-b1ad95c89adf // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jaytaylor/html2text v0.0.0-20230321000545-74c2419ad056 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kennygrant/sanitize v1.2.4 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/levigross/exp-html v0.0.0-20120902181939-8df60c69a8f5 // indirect
	github.com/machinebox/graphql v0.2.2
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/matryer/is v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/oapi-codegen/runtime v1.1.1 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/olekukonko/tablewriter v0.0.4 // indirect
	github.com/otiai10/gosseract/v2 v2.4.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pinecone-io/go-pinecone v1.1.1
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/streamer45/silero-vad-go v0.2.1
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/temoto/robotstxt v1.1.2 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/u2takey/go-utils v0.3.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/nfp v0.0.1 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	gitlab.com/golang-commonmark/html v0.0.0-20191124015941-a22733972181 // indirect
	gitlab.com/golang-commonmark/linkify v0.0.0-20200225224916-64bca66f6ad3 // indirect
	gitlab.com/golang-commonmark/markdown v0.0.0-20211110145824-bf3e522c626a // indirect
	gitlab.com/golang-commonmark/mdurl v0.0.0-20191124015652-932350d1cb84 // indirect
	gitlab.com/golang-commonmark/puny v0.0.0-20191124015043-9f83538fa04f // indirect
	gitlab.com/nyarla/go-crypt v0.0.0-20160106005555-d9a5dc2b789b // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/time v0.11.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20250528174236-200df99c418a // indirect
	gorm.io/driver/mysql v1.5.7 // indirect
	modernc.org/mathutil v1.7.1 // indirect
)
