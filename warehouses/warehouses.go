// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Package warehouses imports the standard data warehouses, included within
// Meergo.
package warehouses

import (
	_ "github.com/meergo/meergo/warehouses/postgresql"
	_ "github.com/meergo/meergo/warehouses/snowflake"
)
