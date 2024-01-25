module chichi

go 1.21

replace github.com/ClickHouse/clickhouse-go/v2 => github.com/open2b/clickhouse-go/v2 v2.12.0-fix

replace (
	chichi/connectors/clickhouse => ./connectors/clickhouse
	chichi/connectors/csv => ./connectors/csv
	chichi/connectors/dummy => ./connectors/dummy
	chichi/connectors/excel => ./connectors/excel
	chichi/connectors/filesystem => ./connectors/filesystem
	chichi/connectors/googleanalytics4 => ./connectors/googleanalytics4
	chichi/connectors/http => ./connectors/http
	chichi/connectors/hubspot => ./connectors/hubspot
	chichi/connectors/json => ./connectors/json
	chichi/connectors/kafka => ./connectors/kafka
	chichi/connectors/klaviyo => ./connectors/klaviyo
	chichi/connectors/mailchimp => ./connectors/mailchimp
	chichi/connectors/mixpanel => ./connectors/mixpanel
	chichi/connectors/mysql => ./connectors/mysql
	chichi/connectors/parquet => ./connectors/parquet
	chichi/connectors/postgresql => ./connectors/postgresql
	chichi/connectors/rabbitmq => ./connectors/rabbitmq
	chichi/connectors/s3 => ./connectors/s3
	chichi/connectors/server => ./connectors/server
	chichi/connectors/sftp => ./connectors/sftp
	chichi/connectors/snowflake => ./connectors/snowflake
	chichi/connectors/stripe => ./connectors/stripe
	chichi/connectors/uisample => ./connectors/uisample
	chichi/connectors/website => ./connectors/website
)

require (
	chichi/connectors/clickhouse v0.0.0-00010101000000-000000000000
	chichi/connectors/csv v0.0.0-00010101000000-000000000000
	chichi/connectors/dummy v0.0.0-00010101000000-000000000000
	chichi/connectors/excel v0.0.0-00010101000000-000000000000
	chichi/connectors/filesystem v0.0.0-00010101000000-000000000000
	chichi/connectors/googleanalytics4 v0.0.0-00010101000000-000000000000
	chichi/connectors/http v0.0.0-00010101000000-000000000000
	chichi/connectors/hubspot v0.0.0-00010101000000-000000000000
	chichi/connectors/json v0.0.0-00010101000000-000000000000
	chichi/connectors/kafka v0.0.0-00010101000000-000000000000
	chichi/connectors/klaviyo v0.0.0-00010101000000-000000000000
	chichi/connectors/mailchimp v0.0.0-00010101000000-000000000000
	chichi/connectors/mixpanel v0.0.0-00010101000000-000000000000
	chichi/connectors/mysql v0.0.0-00010101000000-000000000000
	chichi/connectors/parquet v0.0.0-00010101000000-000000000000
	chichi/connectors/postgresql v0.0.0-00010101000000-000000000000
	chichi/connectors/rabbitmq v0.0.0-00010101000000-000000000000
	chichi/connectors/s3 v0.0.0-00010101000000-000000000000
	chichi/connectors/server v0.0.0-00010101000000-000000000000
	chichi/connectors/sftp v0.0.0-00010101000000-000000000000
	chichi/connectors/snowflake v0.0.0-00010101000000-000000000000
	chichi/connectors/stripe v0.0.0-00010101000000-000000000000
	chichi/connectors/uisample v0.0.0-00010101000000-000000000000
	chichi/connectors/website v0.0.0-00010101000000-000000000000
	github.com/ClickHouse/clickhouse-go/v2 v2.14.3
	github.com/aws/aws-sdk-go-v2 v1.21.2
	github.com/aws/aws-sdk-go-v2/config v1.19.0
	github.com/aws/aws-sdk-go-v2/credentials v1.13.43
	github.com/aws/aws-sdk-go-v2/service/lambda v1.40.0
	github.com/aws/smithy-go v1.15.0
	github.com/evanw/esbuild v0.19.12
	github.com/go-chi/chi/v5 v5.0.11
	github.com/go-json-experiment/json v0.0.0-20231102232822-2e55bd4e08b0
	github.com/golang/snappy v0.0.4
	github.com/google/go-cmp v0.6.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/securecookie v1.1.2
	github.com/itchyny/timefmt-go v0.1.5
	github.com/jackc/pgx/v5 v5.5.2
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
	github.com/jxskiss/base62 v1.1.0
	github.com/mssola/useragent v1.0.0
	github.com/oschwald/geoip2-golang v1.9.0
	github.com/relvacode/iso8601 v1.3.0
	github.com/segmentio/analytics-go/v3 v3.3.0
	github.com/segmentio/ksuid v1.0.4
	github.com/shopspring/decimal v1.3.1
	github.com/snowflakedb/gosnowflake v1.7.1
	go.opentelemetry.io/otel v1.16.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.39.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.16.0
	go.opentelemetry.io/otel/sdk v1.16.0
	go.opentelemetry.io/otel/sdk/metric v0.39.0
	go.opentelemetry.io/otel/trace v1.16.0
	golang.org/x/crypto v0.18.0
	golang.org/x/exp v0.0.0-20231226003508-02704c960a9b
	golang.org/x/text v0.14.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.4.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.1.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.0.0 // indirect
	github.com/ClickHouse/ch-go v0.52.1 // indirect
	github.com/JohnCGriffin/overflow v0.0.0-20211019200055-46fa312c352c // indirect
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/apache/arrow/go/v12 v12.0.1 // indirect
	github.com/apache/thrift v0.16.0 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.14 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.11.59 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.43 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.37 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.45 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.27 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.30 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.37 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.14.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.37.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.15.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.23.2 // indirect
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/dvsekhvalnov/jose2go v1.5.0 // indirect
	github.com/form3tech-oss/jwt-go v3.2.5+incompatible // indirect
	github.com/fraugster/parquet-go v0.12.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.6.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/goccy/go-json v0.10.0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/flatbuffers v23.1.21+incompatible // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/asmfmt v1.3.2 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.3 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/minio/asm2plan9s v0.0.0-20200509001527-cdd76441f9d8 // indirect
	github.com/minio/c2goasm v0.0.0-20190812172519-36a3d3bbc4f3 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/oschwald/maxminddb-golang v1.11.0 // indirect
	github.com/paulmach/orb v0.9.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.19 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.6 // indirect
	github.com/rabbitmq/amqp091-go v1.9.0 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.3 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/segmentio/backo-go v1.0.1 // indirect
	github.com/sirupsen/logrus v1.9.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/twmb/franz-go v1.15.4 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.7.0 // indirect
	github.com/xuri/efp v0.0.0-20230802181842-ad255f2331ca // indirect
	github.com/xuri/excelize/v2 v2.8.0 // indirect
	github.com/xuri/nfp v0.0.0-20230819163627-dc951e3ffe1a // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.16.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.39.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/term v0.16.0 // indirect
	golang.org/x/tools v0.16.0 // indirect
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/genproto v0.0.0-20230913181813-007df8e322eb // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230913181813-007df8e322eb // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230920204549-e6e6cdab5c13 // indirect
	google.golang.org/grpc v1.58.2 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
