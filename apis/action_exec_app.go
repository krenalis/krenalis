//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"fmt"

	_connector "chichi/connector"
)

// importFromApp imports the users from an app.
func (this *Action) importFromApp() error {

	c := this.action.Connection()

	var resource string
	if r, ok := c.Resource(); ok {
		resource = r.Code
	}

	// Read the properties to read.
	_, properties, err := this.schema()
	if err != nil {
		return fmt.Errorf("cannot read user schema: %s", err)
	}

	fh, err := this.newFirehose(context.Background())
	if err != nil {
		return err
	}
	ws := this.action.Connection().Workspace()
	connector, err := _connector.RegisteredApp(c.Connector().Name).Open(fh.ctx, &_connector.AppConfig{
		Role:          _connector.SourceRole,
		Settings:      c.Settings,
		Firehose:      fh,
		Resource:      resource,
		HTTPClient:    this.http.ConnectionClient(c.ID),
		PrivacyRegion: _connector.PrivacyRegion(ws.PrivacyRegion),
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	cursor := this.action.UserCursor
	execution, _ := this.action.Execution()
	if execution.Reimport {
		cursor = ""
	}
	err = connector.(_connector.AppUsersConnection).Users(cursor, properties)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
	}
	if fh.err != nil {
		return actionExecutionError{fh.err}
	}

	return nil
}
