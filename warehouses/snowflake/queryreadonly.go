// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"errors"

	"github.com/meergo/meergo/warehouses"
)

// QueryReadOnly executes a read-only query and returns the results and the
// number of columns in each row.
func (warehouse *Snowflake) QueryReadOnly(_ context.Context, _ string) (warehouses.Rows, int, error) {
	// TODO(Gianluca): implement read-only query validation for Snowflake before
	// allowing this execution path. See https://github.com/meergo/meergo/issues/1665.
	return nil, 0, errors.New("QueryReadOnly is not implemented for Snowflake")
}
