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

	connection := this.action.Connection()
	connector := connection.Connector()
	ctx := context.Background()

	var clientSecret, resourceCode, accessToken string
	if r, ok := connection.Resource(); ok {
		clientSecret = connector.OAuth.ClientSecret
		resourceCode = r.Code
		var err error
		accessToken, err = this.oauth.AccessToken(ctx, r)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
		}
	}

	// Read the properties to read.
	_, properties, err := this.schema()
	if err != nil {
		return fmt.Errorf("cannot read user schema: %s", err)
	}

	fh, err := this.newFirehose(ctx)
	if err != nil {
		return err
	}
	ws := this.action.Connection().Workspace()
	c, err := _connector.RegisteredApp(connector.Name).Open(fh.ctx, &_connector.AppConfig{
		Role:          _connector.SourceRole,
		Settings:      connection.Settings,
		Firehose:      fh,
		ClientSecret:  clientSecret,
		Resource:      resourceCode,
		AccessToken:   accessToken,
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
	err = c.(_connector.AppUsersConnection).Users(cursor, properties)
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot get users from the connector: %s", err)}
	}
	if fh.err != nil {
		return actionExecutionError{fh.err}
	}

	return nil
}
