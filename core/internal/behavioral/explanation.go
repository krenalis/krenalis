// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"time"

	"github.com/krenalis/krenalis/warehouses"
)

// Explanation retains the prepared shape and its closed outcomes in preorder.
// Filter contains the resolved periods; neither it nor Nodes contains SQL.
type Explanation struct {
	Filter warehouses.Filter
	Nodes  []warehouses.FilterNodeOutcome
	At     time.Time
}
