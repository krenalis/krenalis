module chichi

go 1.19

replace (
	chichi/connectors/csv => ./connectors/csv
	chichi/connectors/dummy => ./connectors/dummy
	chichi/connectors/hubspot => ./connectors/hubspot
	chichi/connectors/mysql => ./connectors/mysql
)

require (
	chichi/connectors/csv v0.0.0-00010101000000-000000000000
	chichi/connectors/dummy v0.0.0-00010101000000-000000000000
	chichi/connectors/hubspot v0.0.0-00010101000000-000000000000
	chichi/connectors/mysql v0.0.0-00010101000000-000000000000
	github.com/ClickHouse/clickhouse-go/v2 v2.3.0
	github.com/evanw/esbuild v0.15.5
	github.com/go-sql-driver/mysql v1.6.0
	github.com/mssola/user_agent v0.5.3
	github.com/open2b/nuts v1.5.3
	github.com/open2b/scriggo v0.56.1
	github.com/oschwald/geoip2-golang v1.8.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/text v0.3.7
	gopkg.in/gcfg.v1 v1.2.3
)

require (
	github.com/ClickHouse/ch-go v0.47.3 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/go-faster/city v1.0.1 // indirect
	github.com/go-faster/errors v0.6.1 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/oschwald/maxminddb-golang v1.10.0 // indirect
	github.com/paulmach/orb v0.7.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	go.opentelemetry.io/otel v1.9.0 // indirect
	go.opentelemetry.io/otel/trace v1.9.0 // indirect
	golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)
