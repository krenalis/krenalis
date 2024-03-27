module github.com/open2b/chichi/connectors/kafka

go 1.22

replace github.com/open2b/chichi => ../../

require (
	github.com/open2b/chichi v0.0.0-00010101000000-000000000000
	github.com/twmb/franz-go v1.16.1
)

require (
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.19 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.7.0 // indirect
	golang.org/x/exp v0.0.0-20240318143956-a85f2c67cd81 // indirect
	golang.org/x/text v0.14.0 // indirect
)
