//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// There is no need to modify Chichi's source code to compile a customized
// version. Just follow these steps:
//
//  1. Copy this file (main.go) into a new empty folder
//  2. Execute 'go mod init chichi'
//  3. Execute 'go get github.com/open2b/chichi@latest'
//  4. Edit the imports of the connectors below to include the connectors you want
//  5. Execute 'go mod tidy'
//  6. Execute 'go install' or 'go build' to install/build a custom binary
package main

import (
	"github.com/open2b/chichi/cmd"

	// Connectors that are going to be included in the build of Chichi.
	_ "github.com/open2b/chichi/connectors/clickhouse"
	_ "github.com/open2b/chichi/connectors/csv"
	_ "github.com/open2b/chichi/connectors/dummy"
	_ "github.com/open2b/chichi/connectors/excel"
	_ "github.com/open2b/chichi/connectors/filesystem"
	_ "github.com/open2b/chichi/connectors/googleanalytics"
	_ "github.com/open2b/chichi/connectors/http"
	_ "github.com/open2b/chichi/connectors/hubspot"
	_ "github.com/open2b/chichi/connectors/json"
	_ "github.com/open2b/chichi/connectors/kafka"
	_ "github.com/open2b/chichi/connectors/klaviyo"
	_ "github.com/open2b/chichi/connectors/mailchimp"
	_ "github.com/open2b/chichi/connectors/mixpanel"
	_ "github.com/open2b/chichi/connectors/mobile"
	_ "github.com/open2b/chichi/connectors/mysql"
	_ "github.com/open2b/chichi/connectors/parquet"
	_ "github.com/open2b/chichi/connectors/postgresql"
	_ "github.com/open2b/chichi/connectors/rabbitmq"
	_ "github.com/open2b/chichi/connectors/s3"
	_ "github.com/open2b/chichi/connectors/server"
	_ "github.com/open2b/chichi/connectors/sftp"
	_ "github.com/open2b/chichi/connectors/snowflake"
	_ "github.com/open2b/chichi/connectors/stripe"
	_ "github.com/open2b/chichi/connectors/uisample"
	_ "github.com/open2b/chichi/connectors/website"
)

func main() {
	cmd.Main()
}
