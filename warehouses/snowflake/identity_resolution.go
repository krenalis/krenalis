//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"context"
	_ "embed"
	"time"

	"github.com/meergo/meergo"
)

// ResolveIdentities resolves the identities.
func (warehouse *Snowflake) ResolveIdentities(ctx context.Context, identifiers, userColumns []meergo.Column, userPrimarySources map[string]int) error {
	panic("TODO: not implemented")
}

// LastIdentityResolution returns information about the last Identity
// Resolution.
func (warehouse *Snowflake) LastIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	panic("TODO: not implemented")
}
