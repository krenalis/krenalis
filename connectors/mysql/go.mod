module github.com/open2b/chichi/connectors/mysql

go 1.22

replace github.com/open2b/chichi => ../../

require (
	github.com/go-sql-driver/mysql v1.7.1
	github.com/open2b/chichi v0.0.0-00010101000000-000000000000
	github.com/shopspring/decimal v1.3.1
)

require (
	golang.org/x/exp v0.0.0-20240318143956-a85f2c67cd81 // indirect
	golang.org/x/text v0.14.0 // indirect
)
