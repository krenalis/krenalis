// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"time"

	dbpkg "github.com/krenalis/krenalis/core/internal/db"
)

// UpgradeOrganizationsDB applies to the Krenalis PostgreSQL database the
// schema changes required to add support for enabling and disabling
// organizations, and then returns.
//
// The queries are idempotent, so calling UpgradeOrganizationsDB multiple
// times is safe.
func UpgradeOrganizationsDB(ctx context.Context, conf DBConfig) error {

	db, err := dbpkg.Open(&dbpkg.Options{
		Host:     conf.Host,
		Port:     conf.Port,
		Username: conf.Username,
		Password: conf.Password,
		Database: conf.Database,
		Schema:   conf.Schema,
	})
	if err != nil {
		return fmt.Errorf("cannot connect to PostgreSQL: %s", err)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := db.Ping(pingCtx); err != nil {
		return fmt.Errorf("cannot connect to PostgreSQL: %s", err)
	}

	// ALTER TYPE ... ADD VALUE cannot run inside a transaction block in
	// PostgreSQL, so the queries are executed individually outside of any
	// transaction.
	queries := []string{
		`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS enabled boolean NOT NULL DEFAULT TRUE`,
		`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'SetOrganizationStatus' AFTER 'SetConnectionSettings'`,
	}
	for _, query := range queries {
		if _, err := db.Exec(ctx, query); err != nil {
			return fmt.Errorf("cannot execute upgrade query %q: %s", query, err)
		}
	}

	return nil
}
