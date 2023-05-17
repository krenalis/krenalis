//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	_connector "chichi/connector"
)

// importFromApp imports the users from an app.
func (this *Action) importFromApp() error {

	connection := this.action.Connection()
	connector := connection.Connector()

	var clientSecret, resourceCode, accessToken string
	if r, ok := connection.Resource(); ok {
		clientSecret = connector.OAuth.ClientSecret
		resourceCode = r.Code
		var err error
		accessToken, err = freshAccessToken(this.db, r)
		if err != nil {
			return actionExecutionError{fmt.Errorf("cannot retrive the OAuth access token: %s", err)}
		}
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

	return fh.err
}

// exportToApp exports the users to an app.
// Note that this method is only a draft, and its code may be wrong and/or
// partially implemented.
func (this *Action) exportToApp() error {

	connection := this.action.Connection()

	ctx := context.Background()

	var name, clientSecret, resourceCode, accessToken, refreshToken string
	var webhooksPer WebhooksPer
	var connector, resource int
	var settings []byte
	var expiration time.Time
	err := this.db.QueryRow(ctx,
		"SELECT `c`.`name`, `c`.`oAuthClientSecret`, `c`.`webhooksPer` - 1, `r`.`code`,"+
			" `r`.`oAuthAccessToken`, `r`.`oAuthRefreshToken`, `r`.`oAuthExpiresIn`, `s`.`connector`,"+
			" `s`.`resource`, `s`.`settings`\n"+
			"FROM `connections` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
			"WHERE `s`.`id` = ?", connection.ID).Scan(
		&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
		&resource, &settings)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	fh, err := this.newFirehose(context.Background())
	if err != nil {
		return err
	}
	ws := this.action.Connection().Workspace()

	c, err := _connector.RegisteredApp(name).Open(fh.ctx, &_connector.AppConfig{
		Role:          _connector.SourceRole,
		Settings:      settings,
		Firehose:      fh,
		ClientSecret:  clientSecret,
		Resource:      resourceCode,
		AccessToken:   accessToken,
		PrivacyRegion: _connector.PrivacyRegion(ws.PrivacyRegion),
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}

	// Prepare the users to export to the connection.
	users := []_connector.User{}
	{
		// TODO(Gianluca): populate this map:
		internalToExternalID := map[int]string{}
		rows, err := this.db.Query(ctx, "SELECT user, goldenRecord FROM connection_users WHERE connection = $1", connection.ID)
		if err != nil {
			return err
		}
		defer rows.Close()
		toRead := []int{}
		for rows.Next() {
			var user string
			var goldenRecord int
			err := rows.Scan(&user, &goldenRecord)
			if err != nil {
				return err
			}
			toRead = append(toRead, goldenRecord)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		// Read the users from the Golden Record and apply the
		// transformation functions on them.
		grUsers, err := this.readGRUsers(toRead)
		if err != nil {
			return err
		}
		for _, user := range grUsers {
			id := internalToExternalID[user["id"].(int)]
			user, err := exportUser(id, user)
			if err != nil {
				return err
			}
			users = append(users, user)
		}
	}

	// Export the users to the connection.
	log.Printf("[info] exporting %d user(s) to the connection %d", len(users), connection.ID)
	for _, user := range users {
		err = c.(_connector.AppUsersConnection).SetUser(user)
		if err != nil {
			return errors.New("cannot export user")
		}
	}

	return fh.err
}
