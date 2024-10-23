//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"context"

	"github.com/meergo/meergo"
)

// AlterSchema alters the user schema.
func (warehouse *Snowflake) AlterSchema(ctx context.Context, userColumns []meergo.Column, operations []meergo.AlterSchemaOperation) error {
	panic("TODO: not implemented")
}

// AlterSchemaQueries returns the queries of a schema altering operation.
func (warehouse *Snowflake) AlterSchemaQueries(ctx context.Context, userColumns []meergo.Column, operations []meergo.AlterSchemaOperation) ([]string, error) {
	panic("TODO: not implemented")
}
