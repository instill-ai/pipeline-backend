module github.com/instill-ai/pipeline-backend

go 1.22.5

require (
	cloud.google.com/go/bigquery v1.60.0
	cloud.google.com/go/iam v1.1.8
	cloud.google.com/go/longrunning v0.5.7
	cloud.google.com/go/storage v1.41.0
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/JohannesKaufmann/html-to-markdown v1.5.0
	github.com/PaesslerAG/jsonpath v0.1.1
	github.com/belong-inc/go-hubspot v0.9.0
	github.com/chromedp/chromedp v0.10.0
	github.com/cohere-ai/cohere-go/v2 v2.8.5
	github.com/elastic/go-elasticsearch/v8 v8.14.0
	github.com/emersion/go-imap/v2 v2.0.0-beta.3
	github.com/emersion/go-message v0.18.1
	github.com/extrame/xls v0.0.1
	github.com/fogleman/gg v1.3.0
	github.com/frankban/quicktest v1.14.6
	github.com/gabriel-vasile/mimetype v1.4.3
	github.com/gage-technologies/mistral-go v1.1.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-openapi/strfmt v0.23.0
	github.com/go-redis/redismock/v9 v9.2.0
	github.com/gocolly/colly/v2 v2.1.0
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gogo/status v1.1.1
	github.com/gojuno/minimock/v3 v3.4.0
	github.com/golang-migrate/migrate/v4 v4.15.2
	github.com/google/go-cmp v0.6.0
	github.com/google/go-github/v62 v62.0.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0
	github.com/h2non/filetype v1.1.3
	github.com/iFaceless/godub v0.0.0-20200728093528-a30bb4d1a0f1
	github.com/iancoleman/strcase v0.3.0
	github.com/influxdata/influxdb-client-go/v2 v2.12.3
	github.com/instill-ai/protogen-go v0.3.3-alpha.0.20241012090311-e872dc0b511d
	github.com/instill-ai/usage-client v0.2.4-alpha.0.20240123081026-6c78d9a5197a
	github.com/instill-ai/x v0.5.0-alpha
	github.com/itchyny/gojq v0.12.14
	github.com/jackc/pgx/v5 v5.5.5
	github.com/jmoiron/sqlx v1.4.0
	github.com/json-iterator/go v1.1.12
	github.com/k3a/html2text v1.2.1
	github.com/knadh/koanf v1.5.0
	github.com/launchdarkly/go-semver v1.0.2
	github.com/lestrrat-go/jspointer v0.0.0-20181205001929-82fadba7561c
	github.com/lestrrat-go/jsref v0.0.0-20211028120858-c0bcbb5abf20
	github.com/lestrrat-go/option v1.0.0
	github.com/lestrrat-go/pdebug v0.0.0-20210111095411-35b07dbf089b
	github.com/lestrrat-go/structinfo v0.0.0-20210312050401-7f8bd69d6acb
	github.com/mennanov/fieldmask-utils v1.0.0
	github.com/minio/minio-go/v7 v7.0.76
	github.com/nakagami/firebirdsql v0.9.10
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/openfga/api/proto v0.0.0-20240318145204-66b9e5cb403c
	github.com/pkoukk/tiktoken-go v0.1.7
	github.com/redis/go-redis/v9 v9.5.1
	github.com/samber/lo v1.47.0
	github.com/sijms/go-ora v1.3.2
	github.com/slack-go/slack v0.12.5
	github.com/tmc/langchaingo v0.1.10
	github.com/u2takey/ffmpeg-go v0.5.0
	github.com/weaviate/weaviate v1.26.0-rc.1
	github.com/weaviate/weaviate-go-client/v4 v4.15.0
	github.com/xuri/excelize/v2 v2.8.1
	github.com/xwb1989/sqlparser v0.0.0-20180606152119-120387863bf2
	go.einride.tech/aip v0.60.0
	go.mongodb.org/mongo-driver v1.16.0
	go.opentelemetry.io/contrib/propagators/b3 v1.17.0
	go.opentelemetry.io/otel v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.24.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.24.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.24.0
	go.opentelemetry.io/otel/sdk v1.24.0
	go.opentelemetry.io/otel/sdk/metric v1.24.0
	go.opentelemetry.io/otel/trace v1.24.0
	go.temporal.io/api v1.16.0
	go.temporal.io/sdk v1.21.0
	go.uber.org/zap v1.26.0
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225
	golang.org/x/image v0.18.0
	golang.org/x/mod v0.17.0
	golang.org/x/net v0.28.0
	golang.org/x/oauth2 v0.21.0
	google.golang.org/api v0.178.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240528184218-531527333157
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/guregu/null.v4 v4.0.0
	gorm.io/datatypes v1.2.0
	gorm.io/driver/postgres v1.5.7
	gorm.io/gorm v1.25.7-0.20240204074919-46816ad31dde
	gorm.io/plugin/dbresolver v1.5.1
)

require (
	cloud.google.com/go v0.112.2 // indirect
	cloud.google.com/go/auth v0.3.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.2 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/PaesslerAG/gval v1.0.0 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/antchfx/htmlquery v1.3.0 // indirect
	github.com/antchfx/xmlquery v1.3.17 // indirect
	github.com/antchfx/xpath v1.2.4 // indirect
	github.com/apache/arrow/go/v14 v14.0.2 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go v1.55.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.30.1 // indirect
	github.com/aws/smithy-go v1.20.3 // indirect
	github.com/chromedp/cdproto v0.0.0-20240801214329-3f85d328b335 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.6.0 // indirect
	github.com/emersion/go-sasl v0.0.0-20231106173351-e73c9f7bad43 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.0.4 // indirect
	github.com/extrame/ole2 v0.0.0-20160812065207-d69429661ad7 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-openapi/analysis v0.21.2 // indirect
	github.com/go-openapi/errors v0.22.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.6 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/loads v0.21.1 // indirect
	github.com/go-openapi/spec v0.20.4 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/go-openapi/validate v0.21.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.4 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/itchyny/timefmt-go v0.1.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kennygrant/sanitize v1.2.4 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.18 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/temoto/robotstxt v1.1.2 // indirect
	github.com/tink-ab/tempfile v0.0.0-20180226111222-33beb0518f1a // indirect
	github.com/u2takey/go-utils v0.3.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/xuri/efp v0.0.0-20231025114914-d1ff6096ae53 // indirect
	github.com/xuri/nfp v0.0.0-20230919160717-d98342af3f05 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	gitlab.com/golang-commonmark/html v0.0.0-20191124015941-a22733972181 // indirect
	gitlab.com/golang-commonmark/linkify v0.0.0-20191026162114-a0c2df6c8f82 // indirect
	gitlab.com/golang-commonmark/markdown v0.0.0-20211110145824-bf3e522c626a // indirect
	gitlab.com/golang-commonmark/mdurl v0.0.0-20191124015652-932350d1cb84 // indirect
	gitlab.com/golang-commonmark/puny v0.0.0-20191124015043-9f83538fa04f // indirect
	gitlab.com/nyarla/go-crypt v0.0.0-20160106005555-d9a5dc2b789b // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	modernc.org/mathutil v1.5.0 // indirect
)

require (
	code.sajari.com/docconv v1.3.8
	github.com/JalfResi/justext v0.0.0-20221106200834-be571e3e3052 // indirect
	github.com/PuerkitoBio/goquery v1.9.1
	github.com/advancedlogic/GoOse v0.0.0-20191112112754-e742535969c1 // indirect
	github.com/andybalholm/cascadia v1.3.2 // indirect
	github.com/araddon/dateparse v0.0.0-20200409225146-d820a6159ab1 // indirect
	github.com/catalinc/hashcash v0.0.0-20220723060415-5e3ec3e24f67 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deepmap/oapi-codegen v1.8.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/fatih/set v0.2.1 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gigawattio/window v0.0.0-20180317192513-0f5467e35573 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-resty/resty/v2 v2.12.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/influxdata/line-protocol v0.0.0-20200327222509-2487e7298839 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jaytaylor/html2text v0.0.0-20230321000545-74c2419ad056 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/levigross/exp-html v0.0.0-20120902181939-8df60c69a8f5 // indirect
	github.com/lib/pq v1.10.9
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/olekukonko/tablewriter v0.0.4 // indirect
	github.com/otiai10/gosseract/v2 v2.4.1 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.3 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	github.com/ssor/bom v0.0.0-20170718123548-6386211fdfcf // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/proto/otlp v1.1.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.26.0
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/genproto v0.0.0-20240401170217-c3f982113cda // indirect
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.4.7 // indirect
)
