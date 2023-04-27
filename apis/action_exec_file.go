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

	_connector "chichi/connector"
)

// importFromFile imports the users from a file.
func (ac *Action) importFromFile() error {

	connection := ac.action.Connection()
	connector := connection.Connector()

	var ctx = context.Background()

	// Retrieve the storage associated to the file connection.
	var storage _connector.StorageConnection
	{
		s, _ := connection.Storage()
		fh := ac.newFirehoseForConnection(ctx, s)
		ctx = fh.ctx
		var err error
		storage, err = _connector.RegisteredStorage(s.Connector().Name).Open(ctx, &_connector.StorageConfig{
			Role:     _connector.SourceRole,
			Settings: s.Settings,
			Firehose: fh,
		})
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot connect to the storage connector: %s", err)}
		}
	}

	// Connect to the file connector.
	fh := ac.newFirehose(context.Background())
	file, err := _connector.RegisteredFile(connector.Name).Open(fh.ctx, &_connector.FileConfig{
		Role:     _connector.SourceRole,
		Settings: connection.Settings,
		Firehose: fh,
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the file connector: %s", err)}
	}

	// Read the records.
	rc, timestamp, err := storage.Reader(file.Path())
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot get ReadCloser from storage: %s", err)}
	}
	defer rc.Close()
	records := fh.newRecordWriter(identityColumn, timestampColumn, timestamp, false)
	err = file.Read(rc, records)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot read the file: %s", err)}
	}
	err = rc.Close()
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot close the storage: %s", err)}
	}

	// Handle errors occurred in the firehose.
	if fh.err != nil {
		return fh.err
	}

	return nil
}
