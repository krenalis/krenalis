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
)

// importFromDatabase imports the users from a database.
func (ac *Action) importFromDatabase() error {

	const noColumn = -1

	connection := ac.action.Connection()
	connector := connection.Connector()

	query, err := compileActionQuery(ac.action.Query, noQueryLimit)
	if err != nil {
		return actionExecutionError{err}
	}
	fh := ac.newFirehose(context.Background())
	c, err := _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
		Role:     _connector.SourceRole,
		Settings: connection.Settings,
		Firehose: fh,
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	schema, rows, err := c.Query(query)
	if err != nil {
		if err, ok := err.(*_connector.DatabaseQueryError); ok {
			return actionExecutionError{err}
		}
		return err
	}
	defer rows.Close()
	propertiesNames := schema.PropertiesNames()
	identityIndex := noColumn
	timestampIndex := noColumn
	for i, name := range propertiesNames {
		switch name {
		case identityColumn:
			identityIndex = i
		case timestampColumn:
			timestampIndex = i
		}
	}
	if identityIndex == noColumn {
		return actionExecutionError{fmt.Errorf("missing identity column %q", identityColumn)}
	}
	var now time.Time
	if timestampIndex == noColumn {
		now = time.Now().UTC()
	}
	row := make([]any, len(propertiesNames))
	for rows.Next() {
		for i := range row {
			var v string
			row[i] = &v
		}
		if err = rows.Scan(row...); err != nil {
			return actionExecutionError{fmt.Errorf("cannot read users from database: %s", err)}
		}
		identity := row[identityIndex].(*string)
		var ts time.Time
		if timestampIndex == noColumn {
			ts = now
		} else {
			ts = row[timestampIndex].(time.Time)
		}
		user := map[string]any{}
		for i, name := range propertiesNames {
			v := row[i].(*string)
			user[name] = *v
		}
		fh.SetUser(*identity, user, ts, nil)
	}
	if err = rows.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}
	// Handle errors occurred in the firehose.
	if fh.err != nil {
		return fh.err
	}

	return nil
}
