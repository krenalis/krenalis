// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"

	"github.com/krenalis/krenalis/core/internal/db"
)

// schemaVersion is the schema version of the database of this version of
// Krenalis, an integer unrelated to the Krenalis version. It is incremented
// only by the releases that change the database, whose migrate migrates a
// database of the previous schema version to it. See
// .agents/skills/release-upgrade/SKILL.md.
const schemaVersion = 1

// finalize removes, on conn, what only the Krenalis instances of the previous
// release need from a migrated database, such as the old columns and the
// triggers that keep them in sync, and adds the constraints that they could
// violate. It runs only when none of them is running anymore. This version of
// Krenalis must run on the database before, during, and after it.
//
// It may run its steps on conn outside a transaction, or group some of them
// in transactions with conn.Transaction. If it is interrupted, Finalize runs
// it again from the start, so it must skip the steps already done.
//
// It is set together with migrate, when a release that changes the database
// is prepared, with its helpers in this file, to a function that does nothing
// if there is nothing to finalize. It is set back to nil after the release is
// published, when the helpers are removed, so that Finalize fails, instead of
// recording the database as finalized, in the later releases at the same
// schema version. See .agents/skills/release-upgrade/SKILL.md.
var finalize func(ctx context.Context, conn *db.Conn) error

// migrate migrates, on conn, a database of the previous schema version to the
// current schema. It must not break the Krenalis instances of the previous
// release, which keep running on the database until they are replaced, at any
// point of the migration.
//
// It may run its steps on conn outside a transaction, or group some of them
// in transactions with conn.Transaction. If it is interrupted, Migrate runs it
// again from the start, so it must skip the steps already done.
//
// It is set when a release that changes the database is prepared, from the
// notes in RELEASE-UPGRADE.md, with its helpers in this file, to a function
// that does nothing if there is nothing to migrate. It is set back to nil after
// the release is published, when the helpers are removed, so that Migrate
// fails, instead of recording the database as migrated, in the later releases
// at the same schema version. See .agents/skills/release-upgrade/SKILL.md.
var migrate func(ctx context.Context, conn *db.Conn) error
