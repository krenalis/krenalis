//go:generate go run github.com/krenalis/krenalis/admin

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
//  4. (optional) Add a new file for your connectors and warehouse platforms:
//     package meergo
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

	// Import connectors.
	_ "github.com/krenalis/krenalis/connectors/brevo"
	_ "github.com/krenalis/krenalis/connectors/clickhouse"
	_ "github.com/krenalis/krenalis/connectors/csv"
	_ "github.com/krenalis/krenalis/connectors/excel"
	_ "github.com/krenalis/krenalis/connectors/filesystem"
	_ "github.com/krenalis/krenalis/connectors/googleanalytics"
	_ "github.com/krenalis/krenalis/connectors/http"
	_ "github.com/krenalis/krenalis/connectors/hubspot"
	_ "github.com/krenalis/krenalis/connectors/json"
	_ "github.com/krenalis/krenalis/connectors/klaviyo"
	_ "github.com/krenalis/krenalis/connectors/mailchimp"
	_ "github.com/krenalis/krenalis/connectors/mixpanel"
	_ "github.com/krenalis/krenalis/connectors/mysql"
	_ "github.com/krenalis/krenalis/connectors/parquet"
	_ "github.com/krenalis/krenalis/connectors/postgresql"
	_ "github.com/krenalis/krenalis/connectors/posthog"
	_ "github.com/krenalis/krenalis/connectors/rudderstack"
	_ "github.com/krenalis/krenalis/connectors/s3"
	_ "github.com/krenalis/krenalis/connectors/sdk"
	_ "github.com/krenalis/krenalis/connectors/segment"
	_ "github.com/krenalis/krenalis/connectors/sftp"
	_ "github.com/krenalis/krenalis/connectors/snowflake"
	_ "github.com/krenalis/krenalis/connectors/stripe"
	_ "github.com/krenalis/krenalis/connectors/webhook"

	// Import data warehouses.
	_ "github.com/krenalis/krenalis/warehouses/postgresql"
	_ "github.com/krenalis/krenalis/warehouses/snowflake"

	"github.com/krenalis/krenalis/cmd"
)

//go:embed admin/assets/*
var assets embed.FS

func main() {
	cmd.Main(assets)
}
