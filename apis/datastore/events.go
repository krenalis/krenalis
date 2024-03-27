//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/events/eventschema"
	"github.com/open2b/chichi/types"
)

const flushEventsQueueTimeout = 1 * time.Second // interval to flush queued Events the data warehouse

var eventsMergeTable = warehouses.MergeTable{
	Name:       "events",
	Properties: eventschema.SchemaWithoutGID.Properties(),
	PrimaryKeys: []types.Property{
		{Name: "messageId", Type: types.Text()},
	},
}

// flushEvents flushes a batch of events to the data warehouse.
func (store *Store) flushEvents(events []map[string]any) {
	slog.Info("flush events", "count", len(events))
	for {
		err := store.warehouse.Merge(context.Background(), eventsMergeTable, events, nil)
		if err != nil {
			slog.Error("cannot flush the event queue", "workspace", store.workspace, "err", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		break
	}
}
