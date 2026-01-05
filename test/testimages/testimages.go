// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package testimages encapsulates in a single point of code the versions of
// Docker images to be used in Meergo tests, so that they are not scattered
// throughout the repository and to have a single point to keep them under
// control.
//
// A test that uses an image should refer to the constants defined in this file.
package testimages

const (
	ClickHouse = "clickhouse/clickhouse-server:25.8-alpine"
	MySQL      = "mysql:9.5"
	PostgreSQL = "postgres:18.1-alpine3.23"
)
