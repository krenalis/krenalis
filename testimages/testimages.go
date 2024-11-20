//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

// Package testimages encapsulates in a single point of code the versions of
// Docker images to be used in Meergo tests, so that they are not scattered
// throughout the repository and to have a single point to keep them under
// control.
//
// A test that uses an image should refer to the constants defined in this file.
package testimages

const (
	ClickHouse = "clickhouse/clickhouse-server:23.3.8.21-alpine"
	MySQL      = "mysql:8.0.36"
	PostgreSQL = "postgres:16-alpine"
)
