// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package connectors imports the standard connectors, included within Meergo.
package connectors

import (
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
)
