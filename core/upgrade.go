// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/tools/kms"
)

// UpgradeDBToKMS upgrades an existing PostgreSQL database from the legacy
// settings storage format to the KMS-backed encrypted format.
func UpgradeDBToKMS(ctx context.Context, conf *Config) error {
	if conf == nil {
		conf = &Config{}
	}

	db, err := db.Open(&db.Options{
		Host:           conf.DB.Host,
		Port:           conf.DB.Port,
		Username:       conf.DB.Username,
		Password:       conf.DB.Password,
		Database:       conf.DB.Database,
		Schema:         conf.DB.Schema,
		MaxConnections: 2,
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

	keyManager, err := kms.New(ctx, conf.Kms)
	if err != nil {
		return err
	}

	return initdb.UpgradeToKMS(ctx, db, keyManager)
}
