module chichi

go 1.19

replace (
	chichi/connectors/csv => ./connectors/csv
	chichi/connectors/dummy => ./connectors/dummy
	chichi/connectors/excel => ./connectors/excel
	chichi/connectors/http => ./connectors/http
	chichi/connectors/hubspot => ./connectors/hubspot
	chichi/connectors/kafka => ./connectors/kafka
	chichi/connectors/mailchimp => ./connectors/mailchimp
	chichi/connectors/mysql => ./connectors/mysql
	chichi/connectors/parquet => ./connectors/parquet
	chichi/connectors/postgresql => ./connectors/postgresql
	chichi/connectors/rabbitmq => ./connectors/rabbitmq
	chichi/connectors/s3 => ./connectors/s3
	chichi/connectors/server => ./connectors/server
	chichi/connectors/sftp => ./connectors/sftp
	chichi/connectors/uisample => ./connectors/uisample
	chichi/connectors/website => ./connectors/website
)

require (
	chichi/connectors/csv v0.0.0-00010101000000-000000000000
	chichi/connectors/dummy v0.0.0-00010101000000-000000000000
	chichi/connectors/excel v0.0.0-00010101000000-000000000000
	chichi/connectors/http v0.0.0-00010101000000-000000000000
	chichi/connectors/hubspot v0.0.0-00010101000000-000000000000
	chichi/connectors/kafka v0.0.0-00010101000000-000000000000
	chichi/connectors/mailchimp v0.0.0-00010101000000-000000000000
	chichi/connectors/mysql v0.0.0-00010101000000-000000000000
	chichi/connectors/parquet v0.0.0-00010101000000-000000000000
	chichi/connectors/postgresql v0.0.0-00010101000000-000000000000
	chichi/connectors/rabbitmq v0.0.0-00010101000000-000000000000
	chichi/connectors/s3 v0.0.0-00010101000000-000000000000
	chichi/connectors/server v0.0.0-00010101000000-000000000000
	chichi/connectors/sftp v0.0.0-00010101000000-000000000000
	chichi/connectors/uisample v0.0.0-00010101000000-000000000000
	chichi/connectors/website v0.0.0-00010101000000-000000000000
	github.com/ClickHouse/clickhouse-go/v2 v2.6.0
	github.com/evanw/esbuild v0.17.5
	github.com/go-chi/chi/v5 v5.0.8
	github.com/google/uuid v1.3.0
	github.com/jackc/pgpassfile v1.0.0
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b
	github.com/jackc/puddle/v2 v2.1.2
	github.com/jxskiss/base62 v1.1.0
	github.com/mssola/user_agent v0.5.3
	github.com/open2b/nuts v1.5.3
	github.com/oschwald/geoip2-golang v1.8.0
	github.com/relvacode/iso8601 v1.3.0
	github.com/shopspring/decimal v1.3.1
	github.com/tetratelabs/wazero v1.0.0-pre.4
	golang.org/x/crypto v0.5.0
	golang.org/x/exp v0.0.0-20230105202349-8879d0199aa3
	golang.org/x/text v0.6.0
	gopkg.in/gcfg.v1 v1.2.3
)

require (
	github.com/ClickHouse/ch-go v0.51.2 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/apache/thrift v0.16.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.17.3 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.4.10 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.12.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.27 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.0.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.1.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.29.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.11.28 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.17.7 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/fraugster/parquet-go v0.12.0 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.6.1 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.15.14 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/oschwald/maxminddb-golang v1.10.0 // indirect
	github.com/paulmach/orb v0.8.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.17 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.5 // indirect
	github.com/rabbitmq/amqp091-go v1.5.0 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.3 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/twmb/franz-go v1.11.0 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.2.0 // indirect
	github.com/xuri/efp v0.0.0-20220603152613-6918739fd470 // indirect
	github.com/xuri/excelize/v2 v2.7.0 // indirect
	github.com/xuri/nfp v0.0.0-20220409054826-5e722a1d9e22 // indirect
	go.opentelemetry.io/otel v1.11.2 // indirect
	go.opentelemetry.io/otel/trace v1.11.2 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/net v0.5.0 // indirect
	golang.org/x/sync v0.0.0-20220923202941-7f9b1623fab7 // indirect
	golang.org/x/sys v0.4.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
