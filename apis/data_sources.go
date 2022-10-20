//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
	"unicode/utf8"

	"chichi/connectors"
	"chichi/pkg/open2b/sql"
)

type DataSources struct {
	*WorkspaceAPI
}

var ErrConnectorNotFound = errors.New("connector does not exist")
var ErrDataSourceNotFound = errors.New("data source does not exist")
var ErrResourceNotFound = errors.New("resource does not exist")
var ErrCannotGetConnectorAccessToken = fmt.Errorf("cannot get access token")

// DataSource represents a data source.
type DataSource struct {
	ID       int
	Name     string
	OauthURL string
	LogoURL  string
}

// PropertyType represents the type of a property.
type PropertyType string

// DataSourcePropertyOption represents an option of a data source property.
type DataSourcePropertyOption struct {
	Label string
	Value string
}

// DataSourceProperty represents a connector property.
type DataSourceProperty struct {
	Name    string
	Type    PropertyType
	Label   string
	Options []DataSourcePropertyOption
}

// Add adds a data source given its connector and the OAuth refresh and access
// tokens of the resource and returns its identifier.
// If the data source already exists for the given connector and resource, it
// updates the data source.
func (this *DataSources) Add(connector int, refreshToken, accessToken string) (int, error) {
	conn, err := this.api.apis.Connector(connector)
	if err != nil {
		return 0, err
	}
	if conn == nil {
		return 0, ErrConnectorNotFound
	}
	c := connectors.Connector(conn.Name, conn.ClientSecret)
	ctx := context.WithValue(context.Background(), connectors.AccessTokenContextKey{}, accessToken)
	resourceCode, err := c.Resource(ctx)
	if err != nil {
		return 0, err
	}
	var id int64
	err = this.myDB.Transaction(func(tx *sql.Tx) error {
		var resource int
		var currentRefreshToken string
		err := tx.QueryRow("SELECT `id`, `refreshToken` FROM `resources` WHERE `connector` = ? AND `code` = ?",
			connector, resourceCode).Scan(&resource, &currentRefreshToken)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}
			err = nil
		}
		if resource == 0 {
			result, err := tx.Exec("INSERT INTO `resources` SET `connector` = ?, `code` = ?, `refreshToken` = ?",
				connector, resourceCode, refreshToken)
			if err != nil {
				return err
			}
			resourceID, err := result.LastInsertId()
			resource = int(resourceID)
		} else if refreshToken != currentRefreshToken {
			_, err = tx.Exec("UPDATE `resources` SET `refreshToken` = ? WHERE `id` = ?", refreshToken, resource)
		}
		if err != nil {
			return err
		}
		result, err := tx.Exec("INSERT INTO `data_sources` SET `workspace` = ?, `connector` = ?, `resource` = ?",
			this.workspace, connector, resource)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return 0, err
	}

	go func() {
		err := this.reloadProperties(int(id))
		if err != nil {
			log.Printf("[error] cannot reload properties for data source %d: %s", id, err)
		}
	}()

	return int(id), err
}

// Delete deletes the data source with the given identifier.
// If the data source does not exist, it does nothing.
func (this *DataSources) Delete(id int) error {
	if id <= 0 {
		return errors.New("invalid data source identifier")
	}
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		source, err := tx.Table("DataSources").Get(
			sql.Where{"id": id, "workspace": this.workspace},
			sql.Columns{"resource"})
		if err != nil {
			return err
		}
		if source == nil {
			return nil
		}
		_, err = tx.Table("DataSources").Delete(sql.Where{"id": id})
		if err != nil {
			return err
		}
		_, err = tx.Table("DataSourcesProperties").Delete(sql.Where{"source": id})
		_, err = tx.Table("DataSourcesUsers").Delete(sql.Where{"source": id})
		// Delete the resource of the deleted data source if it has no other data sources.
		_, err = tx.Exec("DELETE `r`\n"+
			"FROM `resources` AS `r`\n"+
			"LEFT JOIN `data_sources` AS `s` ON `s`.`resource` = `r`.`id`\n"+
			"WHERE `r`.`id` = ? AND `s`.`resource` IS NULL", source["resource"])
		return err
	})
	return err
}

// Import starts the import of the users from the data source with the given
// identifier. If reimport is false it imports the users from the current
// cursor, otherwise imports all users.
// Returns the ErrDataSourceNotFound error if the data source does not exist.
func (this *DataSources) Import(id int, reimport bool) error {

	if id <= 0 {
		return errors.New("invalid data source identifier")
	}

	var name, clientSecret, webhooksPer, resourceCode, accessToken, refreshToken, cursor string
	var connector, resource int
	var settings []byte
	var expiration time.Time
	err := this.myDB.QueryRow(
		"SELECT `c`.`name`, `c`.`clientSecret`, `c`.`webhooksPer`, `r`.`code`, `r`.`accessToken`,"+
			" `r`.`refreshToken`, `r`.`accessTokenExpirationTimestamp`, `s`.`connector`,"+
			" `s`.`resource`, `s`.`userCursor`, `s`.`settings`\n"+
			"FROM `data_sources` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?", id, this.workspace).Scan(
		&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
		&resource, &cursor, &settings)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrDataSourceNotFound
		}
		return err
	}
	if reimport {
		cursor = ""
	}

	accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)

	if accessToken == "" || accessTokenExpired {
		accessToken, err = this.api.apis.refreshOAuthToken(resource)
		if err != nil {
			return err
		}
	}

	var properties []string
	err = this.myDB.QueryScan("SELECT `name`\nFROM `data_sources_properties`\nWHERE `source` = ?", id, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var name string
			if err = rows.Scan(&name); err != nil {
				return err
			}
			properties = append(properties, name)
		}
		return nil
	})
	if err != nil {
		return err
	}

	go func() {
		conn := connectors.Connector(name, clientSecret)
		ctx := this.newConnectorContext(context.Background(), id, resource, connector, resourceCode, accessToken,
			webhooksPer, settings)
		err := conn.Users(ctx, cursor, properties)
		if err != nil {
			log.Printf("[error] call to the Users method of the data source %d failed: %s", id, err)
		}
	}()

	return nil
}

// List returns all data sources.
func (this *DataSources) List() ([]*DataSource, error) {
	sources := []*DataSource{}
	err := this.myDB.QueryScan("SELECT `ds`.`id`, `c`.`name`, `c`.`oauthURL`, `c`.`logoURL`\n"+
		"FROM `data_sources` as `ds`\n"+
		"INNER JOIN `connectors` AS `c` ON `c`.`id` = `ds`.`connector`\n"+
		"WHERE `workspace` = ?", this.workspace, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var source DataSource
			if err = rows.Scan(&source.ID, &source.Name, &source.OauthURL, &source.LogoURL); err != nil {
				return err
			}
			sources = append(sources, &source)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sources, nil
}

// Properties returns the properties of the data source with the given
// identifier.
// Returns the ErrDataSourceNotFound error if the data source does not exist.
func (this *DataSources) Properties(id int) ([]*DataSourceProperty, error) {

	if id <= 0 {
		return nil, errors.New("invalid data source identifier")
	}

	var properties []*DataSourceProperty

	stmt := "SELECT `name`, `type`, `label`, `options`\n" +
		"FROM `data_sources_properties`\n" +
		"INNER JOIN `data_sources` ON `id` = `source`\n" +
		"WHERE `source` = ? AND `workspace` = ?\n" +
		"ORDER BY `position`"

	err := this.myDB.QueryScan(stmt, id, this.workspace, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var property DataSourceProperty
			var options []byte
			if err = rows.Scan(&property.Name, &property.Type, &property.Label, &options); err != nil {
				return err
			}
			if len(options) > 0 {
				property.Options = []DataSourcePropertyOption{}
				err := json.Unmarshal(options, &property.Options)
				if err != nil {
					return fmt.Errorf("malformed options for data source %d", id)
				}
			}
			properties = append(properties, &property)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if properties == nil {
		var exists bool
		err := this.myDB.QueryRow("SELECT TRUE FROM `data_sources`\nWHERE `id` = ? AND `workspace` = ?", id, this.workspace).Scan(&exists)
		if err != nil {
			if err == sql.ErrNoRows {
				err = ErrDataSourceNotFound
			}
			return nil, err
		}
		properties = []*DataSourceProperty{}
	}

	return properties, nil
}

// ServeUserInterface serves the user interface for the data source with the
// given identifier.
// Returns the ErrDataSourceNotFound error if the data source does not exist.
func (this *DataSources) ServeUserInterface(id int, w http.ResponseWriter, r *http.Request) error {

	// TODO(marco) The following code is duplicated in the Import method.
	var name, clientSecret, webhooksPer, resourceCode, accessToken, refreshToken, cursor string
	var connector, resource int
	var settings []byte
	var expiration time.Time
	err := this.myDB.QueryRow(
		"SELECT `c`.`name`, `c`.`clientSecret`, `c`.`webhooksPer`, `r`.`code`, `r`.`accessToken`,"+
			" `r`.`refreshToken`, `r`.`accessTokenExpirationTimestamp`, `s`.`connector`,"+
			" `s`.`resource`, `s`.`userCursor`, `s`.`settings`\n"+
			"FROM `data_sources` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?", id, this.workspace).Scan(
		&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
		&resource, &cursor, &settings)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrDataSourceNotFound
		}
		return err
	}

	accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)

	if accessToken == "" || accessTokenExpired {
		accessToken, err = this.api.apis.refreshOAuthToken(resource)
		if err != nil {
			return err
		}
	}

	conn := connectors.Connector(name, clientSecret)
	ctx := this.newConnectorContext(r.Context(), id, resource, connector, resourceCode, accessToken, webhooksPer, settings)
	r.Clone(ctx)
	r.Header.Del("Cookie") // remove the cookies from the request.
	conn.ServeUserInterface(w, r)

	return nil
}

// SetTransformationFunc sets the transformation function of the data source
// with the given identifier.
// Returns the ErrDataSourceNotFound error if the data source does not exist.
func (this *DataSources) SetTransformationFunc(id int, fn string) error {
	if id <= 0 {
		return errors.New("invalid data source identifier")
	}
	if !utf8.ValidString(fn) {
		return errors.New("invalid transformation function")
	}
	affected, err := this.myDB.Table("DataSources").Update(
		sql.Set{"transformation": fn},
		sql.Where{"id": id, "workspace": this.workspace})
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrDataSourceNotFound
	}
	return nil
}

// DataSourcesStats represents the statistics on a data source for the last 24
// hours.
type DataSourcesStats struct {
	UsersIn [24]int // ingested users per hour
}

// Stats returns statistics on the data source with identifier id for the last
// 24 hours.
func (this *DataSources) Stats(id int) (*DataSourcesStats, error) {
	now := time.Now().UTC()
	toSlot := statsTimeSlot(now)
	fromSlot := toSlot - 23
	stats := &DataSourcesStats{
		UsersIn: [24]int{},
	}
	query := "SELECT `timeSlot`, `usersIn`\nFROM `data_sources_stats`\nWHERE `source` = ? AND `timeSlot` BETWEEN ? AND ?"
	err := this.myDB.QueryScan(query, id, fromSlot, toSlot, func(rows *sql.Rows) error {
		var err error
		var slot, usersIn int
		for rows.Next() {
			if err = rows.Scan(&slot, &usersIn); err != nil {
				return err
			}
			stats.UsersIn[slot-fromSlot] = usersIn
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// TransformationFunc returns the transformation function of the data source
// with the given identifier.
// Returns the ErrDataSourceNotFound error if the data source does not exist.
func (this *DataSources) TransformationFunc(id int) (string, error) {
	if id <= 0 {
		return "", errors.New("invalid data source identifier")
	}
	row, err := this.myDB.Table("DataSources").Get(sql.Where{"id": id, "workspace": this.workspace}, []any{"transformation"})
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", ErrDataSourceNotFound
	}
	return row["transformation"].(string), nil
}

// newConnectorContext returns a context with a Firehose used to call a
// connector method.
func (this *DataSources) newConnectorContext(ctx context.Context, source, resource, connector int, resourceCode,
	accessToken, webhooksPer string, settings []byte) context.Context {

	fh := &firehose{sources: this, source: source, resource: resource, connector: connector, webhooksPer: webhooksPer}
	fh.context, fh.cancel = context.WithCancel(ctx)
	fh.context = context.WithValue(fh.context, connectors.ResourceContextKey{}, resourceCode)
	fh.context = context.WithValue(fh.context, connectors.AccessTokenContextKey{}, accessToken)
	fh.context = context.WithValue(fh.context, connectors.SettingsContextKey{}, settings)
	fh.context = context.WithValue(fh.context, connectors.FirehoseContextKey{}, fh)

	return fh.context
}

// reloadProperties reloads the properties of the data source with identifier
// id.
func (this *DataSources) reloadProperties(id int) error {

	// TODO(marco) The following code is duplicated in the Import method.
	var name, clientSecret, webhooksPer, resourceCode, accessToken, refreshToken, cursor string
	var connector, resource int
	var settings []byte
	var expiration time.Time
	err := this.myDB.QueryRow(
		"SELECT `c`.`name`, `c`.`clientSecret`, `c`.`webhooksPer`, `r`.`code`, `r`.`accessToken`,"+
			" `r`.`refreshToken`, `r`.`accessTokenExpirationTimestamp`, `s`.`connector`,"+
			" `s`.`resource`, `s`.`userCursor`, `s`.`settings`\n"+
			"FROM `data_sources` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?", id, this.workspace).Scan(
		&name, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
		&resource, &cursor, &settings)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrDataSourceNotFound
		}
		return err
	}

	accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)

	if accessToken == "" || accessTokenExpired {
		accessToken, err = this.api.apis.refreshOAuthToken(resource)
		if err != nil {
			return err
		}
	}

	conn := connectors.Connector(name, clientSecret)
	ctx := this.newConnectorContext(context.Background(), id, resource, connector, resourceCode, accessToken,
		webhooksPer, settings)
	properties, _, err := conn.Properties(ctx)
	if err != nil {
		return err
	}

	rows := make([][]any, len(properties))
	for i, p := range properties {
		rows[i] = []any{id, p.Name, p.Type, p.Label, i}
	}
	err = this.myDB.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec("DELETE FROM `data_sources_properties` WHERE `source` = ?", id)
		if err != nil {
			return err
		}
		_, err = tx.Table("DataSourcesProperties").Insert([]string{"source", "name", "type", "label", "position"}, rows, nil)
		return err
	})

	return err
}
