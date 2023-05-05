//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"fmt"
	"time"

	_connector "chichi/connector"
	"chichi/connector/types"
)

// importFromDatabase imports the users from a database.
func (this *Action) importFromDatabase() error {

	connection := this.action.Connection()
	connector := connection.Connector()

	query, err := compileActionQuery(this.action.Query, noQueryLimit)
	if err != nil {
		return actionExecutionError{err}
	}
	fh := this.newFirehose(context.Background())
	c, err := _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
		Role:     _connector.SourceRole,
		Settings: connection.Settings,
		Firehose: fh,
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	rawRows, properties, err := c.Query(query)
	if err != nil {
		return actionExecutionError{err}
	}
	defer rawRows.Close()
	var hasIdentity bool
	var hasTimestamp bool
	for _, p := range properties {
		if p.Name == identityLabel {
			hasIdentity = true
		}
		if p.Name == timestampLabel {
			hasTimestamp = true
		}
	}
	if !hasIdentity {
		return actionExecutionError{fmt.Errorf("missing identity column %q", identityLabel)}
	}
	now := time.Now().UTC()
	dest := make([]any, len(properties))
	for rawRows.Next() {
		row := make(map[string]any, len(properties))
		for i, p := range properties {
			dest[i] = databaseScanValue{property: p, row: row}
		}
		if err := rawRows.Scan(dest...); err != nil {
			return actionExecutionError{fmt.Errorf("query execution failed: %s", err)}
		}
		ts := now
		if hasTimestamp {
			ts = row[timestampLabel].(time.Time)
		}
		timestamps := map[string]time.Time{}
		for _, p := range properties {
			timestamps[p.Name] = ts
		}
		err = this.setUser(row[identityLabel].(string), row, timestamps)
		if err != nil {
			return err
		}
	}
	if err = rawRows.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}
	// Handle errors occurred in the firehose.
	if fh.err != nil {
		return fh.err
	}

	return nil
}

// databaseScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type databaseScanValue struct {
	property types.Property
	row      map[string]any
}

func (sv databaseScanValue) Scan(src any) error {
	value, err := normalizeDatabaseFilePropertyValue(sv.property, src)
	if err != nil {
		return err
	}
	sv.row[sv.property.Name] = value
	return nil
}
