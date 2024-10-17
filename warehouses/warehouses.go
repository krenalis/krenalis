//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

// Package warehouses imports the standard data warehouses, included within
// Meergo.
package warehouses

import (
	_ "github.com/meergo/meergo/warehouses/postgresql"
	_ "github.com/meergo/meergo/warehouses/snowflake"
)
