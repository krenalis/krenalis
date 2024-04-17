//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

// Package stdconnectors is a convenience package that imports all the
// connectors defined within Chichi.
package stdconnectors

import (
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
