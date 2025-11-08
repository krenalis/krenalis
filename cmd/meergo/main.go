//go:generate go run github.com/meergo/meergo/admin

// To compile:
//
//  1. go generate
//  2. go build
//
// To add your custom connectors, data warehouses, or choose what connector
// or data warehouse to build into Meergo:
//
//  1. Create a new directory: mkdir meergo
//  2. Change into it: cd meergo
//  3. Copy this file into the new directory
//  4. (optional) Edit the copied file to add your connectors, or data warehouses:
//     import _ "github.com/example/connector"
//     import _ "github.com/example/warehouse"
//  5. Initialize a Go module: go mod init meergo
//  6. Tidy the module: go mod tidy
//  7. Generate the Admin console assets: go generate
//  8. Build: go build
//
// Note: You can provide the '-trimpath' option to the 'go build' command to
// remove absolute paths from any error stack traces in Meergo. This way, if
// telemetry is enabled, the absolute paths will not be sent.
//
// Note: Re-execute 'go generate' if you change Meergo module version.
//
// TODO: insert URL which points to the page in the doc for compiling Meergo from sources.
package main

import (
	"embed"

	"github.com/meergo/meergo/cmd"

	// Add your custom connectors and data warehouses here:
	// _ "github.com/example/connector"
	// _ "github.com/example/warehouse"

	// Imports the standard connectors:
	_ "github.com/meergo/meergo/connectors/clickhouse"
	_ "github.com/meergo/meergo/connectors/csv"
	_ "github.com/meergo/meergo/connectors/excel"
	_ "github.com/meergo/meergo/connectors/filesystem"
	_ "github.com/meergo/meergo/connectors/googleanalytics"
	_ "github.com/meergo/meergo/connectors/http"
	_ "github.com/meergo/meergo/connectors/hubspot"
	_ "github.com/meergo/meergo/connectors/json"
	_ "github.com/meergo/meergo/connectors/klaviyo"
	_ "github.com/meergo/meergo/connectors/mailchimp"
	_ "github.com/meergo/meergo/connectors/mixpanel"
	_ "github.com/meergo/meergo/connectors/mysql"
	_ "github.com/meergo/meergo/connectors/parquet"
	_ "github.com/meergo/meergo/connectors/postgresql"
	_ "github.com/meergo/meergo/connectors/rudderstack"
	_ "github.com/meergo/meergo/connectors/s3"
	_ "github.com/meergo/meergo/connectors/sdk"
	_ "github.com/meergo/meergo/connectors/segment"
	_ "github.com/meergo/meergo/connectors/sftp"
	_ "github.com/meergo/meergo/connectors/snowflake"
	_ "github.com/meergo/meergo/connectors/stripe"
	_ "github.com/meergo/meergo/connectors/webhook"

	// Imports the standard data warehouses:
	_ "github.com/meergo/meergo/warehouses/postgresql"
	_ "github.com/meergo/meergo/warehouses/snowflake"
)

//go:embed meergo-assets/*
var assets embed.FS

func main() {
	cmd.Main(assets)
}
