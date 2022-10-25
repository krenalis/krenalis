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
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/connectors"
	"chichi/pkg/open2b/sql"
)

type DataSources struct {
	*WorkspaceAPI
}

var ErrConnectorNotFound = errors.New("connector does not exist")
var ErrInvalidConnectorType = errors.New("connector has an invalid type")
var ErrDataSourceNotFound = errors.New("data source does not exist")
var ErrResourceNotFound = errors.New("resource does not exist")
var ErrCannotGetConnectorAccessToken = errors.New("cannot get access token")

const (
	rawPropertiesMaxSize = 16_777_215 // maximum size in runes of the 'property' column of the 'data_sources' table.
	queryMaxSize         = 16_777_215 // maximum size in runes of a data source query.
)

// DataSource represents a data source.
type DataSource struct {
	ID       int
	Name     string
	Type     string
	OauthURL string
	LogoURL  string
}

// DataSourceInfo represents a data source.
type DataSourceInfo struct {
	ID                 int
	Type               string
	TransformationFunc string
	UsersQuery         string // only for databases.
}

// PropertyType represents the type of a property.
type PropertyType string

// DataSourcePropertyOption represents an option of a data source property.
type DataSourcePropertyOption struct {
	Label string
	Value string
}

// DataSourceProperty represents a data source property.
type DataSourceProperty struct {
	Name       string
	Type       PropertyType
	Label      string
	Options    []DataSourcePropertyOption
	Properties []DataSourceProperty
}

// AddApp adds an app data source given its connector and the OAuth refresh and
// access tokens and returns its identifier.
//
// If the connector does not exist, it returns the ErrConnectorNotFound error.
// If the connector is not an app, it returns the ErrInvalidConnectorType
// error.
func (this *DataSources) AddApp(connector int, refreshToken, accessToken string) (int, error) {
	conn, err := this.api.apis.Connector(connector)
	if err != nil {
		return 0, err
	}
	if conn == nil {
		return 0, ErrConnectorNotFound
	}
	if conn.Type != "App" {
		return 0, ErrInvalidConnectorType
	}
	c, err := connectors.NewAppConnection(context.Background(), conn.Name, &connectors.AppConfig{
		ClientSecret: conn.ClientSecret,
		AccessToken:  accessToken,
	})
	if err != nil {
		return 0, err
	}
	resourceCode, err := c.Resource()
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
		result, err := tx.Exec("INSERT INTO `data_sources` SET `workspace` = ?, `type` = 'App', `connector` = ?, `resource` = ?",
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

// AddDatabase adds a database data source given its database connector and
// returns its identifier.
//
// If the connector does not exist, it returns the ErrConnectorNotFound error.
// If the connector is not a database, it returns the ErrInvalidConnectorType
// error.
func (this *DataSources) AddDatabase(connector int) (int, error) {
	var id int64
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var connectorType string
		err := tx.QueryRow("SELECT `type` FROM `connectors` WHERE `id` = ?", connector).Scan(&connectorType)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectorNotFound
			}
			return err
		}
		if connectorType != "Database" {
			return ErrInvalidConnectorType
		}
		result, err := tx.Exec("INSERT INTO `data_sources` SET `workspace` = ?, `type` = 'Database', `connector` = ?",
			this.workspace, connector)
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// AddFileStream adds a file-stream data source given its file and stream
// connectors and returns its identifier.
//
// If a connector does not exist, it returns the ErrConnectorNotFound error. If
// the connectors are not a file and a stream respectively, it returns the
// ErrInvalidConnectorType error.
func (this *DataSources) AddFileStream(fileConnector, streamConnector int) (int, error) {
	var id int64
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		var connectorType string
		stmt, err := tx.Prepare("SELECT `type` FROM `connectors` WHERE `id` = ?")
		if err != nil {
			return err
		}
		// Check the file connector.
		err = stmt.QueryRow(fileConnector).Scan(&connectorType)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectorNotFound
			}
			return err
		}
		if connectorType != "File" {
			return ErrInvalidConnectorType
		}
		// Check the stream connector.
		err = stmt.QueryRow(streamConnector).Scan(&connectorType)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrConnectorNotFound
			}
			return err
		}
		if connectorType != "Stream" {
			return ErrInvalidConnectorType
		}
		// Add the data source.
		result, err := tx.Exec("INSERT INTO `data_sources` SET `workspace` = ?, `type` = 'FileStream', `connector` = ? AND `stream` = ?",
			this.workspace, fileConnector, streamConnector)
		if err != nil {
			return err
		}
		id, err = result.LastInsertId()
		return err
	})
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// Get returns the data source with identifier id. If the data source does not
// exist, it returns the ErrDataSourceNotFound error.
func (this *DataSources) Get(id int) (*DataSourceInfo, error) {
	if id <= 0 {
		return nil, errors.New("invalid data source identifier")
	}
	s := DataSourceInfo{ID: id}
	err := this.myDB.QueryRow("SELECT `type`, `transformation`, `usersQuery`\nFROM `data_sources`\nWHERE `id` = ? AND `workspace` = ?",
		id, this.workspace).Scan(&s.Type, &s.TransformationFunc, &s.UsersQuery)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrDataSourceNotFound
		}
	}
	return &s, nil
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

	var name, connectorType, clientSecret, webhooksPer, resourceCode, accessToken, refreshToken, cursor string
	var connector, resource int
	var settings, rawUsedProperties []byte
	var expiration time.Time
	err := this.myDB.QueryRow(
		"SELECT `c`.`name`, `c`.`type`, `c`.`clientSecret`, `c`.`webhooksPer`, `r`.`code`, `r`.`accessToken`,"+
			" `r`.`refreshToken`, `r`.`accessTokenExpirationTimestamp`, `s`.`connector`,"+
			" `s`.`resource`, `s`.`userCursor`, `s`.`settings`, `s`.`usedProperties`\n"+
			"FROM `data_sources` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?", id, this.workspace).Scan(
		&name, &connectorType, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration, &connector,
		&resource, &cursor, &settings, &rawUsedProperties)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrDataSourceNotFound
		}
		return err
	}
	if reimport {
		cursor = ""
	}
	var properties [][]string
	err = json.Unmarshal(rawUsedProperties, &properties)
	if err != nil {
		return fmt.Errorf("cannon unmarshal used properties of data source %d: %s", id, err)
	}

	accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)

	if accessToken == "" || accessTokenExpired {
		accessToken, err = this.api.apis.refreshOAuthToken(resource)
		if err != nil {
			return err
		}
	}

	go func() {
		fh := this.newFirehose(context.Background(), id, connector, resource, connectorType, webhooksPer)
		c, err := connectors.NewAppConnection(fh.ctx, name, &connectors.AppConfig{
			Settings:     settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			log.Printf("[error] cannot connect to the connector %d of the data source %d: %s", connector, id, err)
			return
		}
		err = c.Users(cursor, properties)
		if err != nil {
			log.Printf("[error] call to the Users method of the data source %d failed: %s", id, err)
		}
	}()

	return nil
}

// List returns all data sources.
func (this *DataSources) List() ([]*DataSource, error) {
	sources := []*DataSource{}
	err := this.myDB.QueryScan("SELECT `ds`.`id`, `c`.`name`, `c`.`type`, `c`.`oauthURL`, `c`.`logoURL`\n"+
		"FROM `data_sources` as `ds`\n"+
		"INNER JOIN `connectors` AS `c` ON `c`.`id` = `ds`.`connector`\n"+
		"WHERE `workspace` = ?", this.workspace, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var source DataSource
			if err = rows.Scan(&source.ID, &source.Name, &source.Type, &source.OauthURL, &source.LogoURL); err != nil {
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

// Properties returns the properties and the used properties of the data source
// with the given identifier.
// Returns the ErrDataSourceNotFound error if the data source does not exist.
func (this *DataSources) Properties(id int) ([]DataSourceProperty, [][]string, error) {
	if id <= 0 {
		return nil, nil, errors.New("invalid data source identifier")
	}
	var rawProperties, rawUsedProperties []byte
	err := this.myDB.QueryRow("SELECT `properties`, `usedProperties` FROM `data_sources` WHERE `id` = ?", id).
		Scan(&rawProperties, &rawUsedProperties)
	if err != nil {
		return nil, nil, err
	}
	var properties []DataSourceProperty
	if len(rawProperties) > 0 {
		err = json.Unmarshal(rawProperties, &properties)
		if err != nil {
			return nil, nil, errors.New("cannot unmarshal data source properties")
		}
	} else {
		properties = []DataSourceProperty{}
	}
	var usedProperties [][]string
	if len(rawUsedProperties) > 0 {
		err = json.Unmarshal(rawUsedProperties, &usedProperties)
		if err != nil {
			return nil, nil, errors.New("cannot unmarshal data source used properties")
		}
	} else {
		usedProperties = [][]string{}
	}

	return properties, usedProperties, nil
}

// Column represents a column of a database data source.
type Column struct {
	Name string
	Type string
}

// Query executes the given query on the data source with identifier id and
// returns the resulting columns and rows.
//
// query must be UTF-8 encoded, it cannot be longer than 16,777,215 runes and
// must contain the ':limit' placeholder. limit must be between 1 and 100.
//
// It returns the ErrDataSourceNotFound error if the data source does not
// exist and the ErrInvalidConnectorType error if the data source is not a
// database.
func (this *DataSources) Query(id int, query string, limit int) ([]Column, [][]string, error) {

	if id <= 0 {
		return nil, nil, errors.New("invalid data source identifier")
	}

	if !utf8.ValidString(query) {
		return nil, nil, errors.New("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return nil, nil, fmt.Errorf("query is longer than %d", queryMaxSize)
	}
	if !strings.Contains(query, ":limit") {
		return nil, nil, errors.New("query does not contain the placeholder \":limit\"")
	}
	if limit < 1 || limit > 100 {
		return nil, nil, errors.New("invalid limit")
	}

	var connector int
	var connectorName, connectorType string
	var settings []byte
	err := this.myDB.QueryRow(
		"SELECT `s`.`connector`, `s`.`settings`, `c`.`name`, `c`.`type`\n"+
			"FROM `data_sources` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?", id, this.workspace).Scan(
		&connector, &settings, &connectorName, &connectorType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrDataSourceNotFound
		}
		return nil, nil, err
	}
	if connectorType != "Database" {
		return nil, nil, ErrInvalidConnectorType
	}

	// Execute the query.
	query, err = this.compileQueryWithLimit(query, limit)
	if err != nil {
		return nil, nil, err
	}
	fh := this.newFirehose(context.Background(), id, connector, 0, connectorType, "")
	c, err := connectors.NewDatabaseConnection(fh.ctx, connectorName, settings, fh)
	if err != nil {
		return nil, nil, err
	}
	rawColumns, rawRows, err := c.Query(query)
	if err != nil {
		return nil, nil, err
	}

	// Fill the columns.
	columns := make([]Column, len(rawColumns))
	for i, c := range rawColumns {
		columns[i].Name = c.Name
		columns[i].Type = c.Type
	}

	// Fill the rows.
	var rows [][]string
	values := make([]any, len(columns))
	for i := range values {
		var value string
		values[i] = &value
	}
	for rawRows.Next() {
		if err := rawRows.Scan(values...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(rawColumns))
		for i, v := range values {
			row[i] = *(v.(*string))
		}
		rows = append(rows, row)
	}
	err = rawRows.Close()
	if err != nil {
		return nil, nil, err
	}
	if rows == nil {
		rows = [][]string{}
	}

	return columns, rows, nil
}

// ServeUserInterface serves the user interface for the data source with the
// given identifier.
// Returns the ErrDataSourceNotFound error if the data source does not exist.
func (this *DataSources) ServeUserInterface(id int, w http.ResponseWriter, r *http.Request) error {

	if id <= 0 {
		return errors.New("invalid data source identifier")
	}

	// TODO(marco) The following code is duplicated in the Import method (apart from the 'usedProperties' column).
	var connectorName, connectorType, clientSecret, webhooksPer, resourceCode, accessToken, refreshToken, cursor string
	var connector, resource int
	var settings []byte
	var expiration time.Time
	err := this.myDB.QueryRow(
		"SELECT `c`.`name`, `c`.`type`, `c`.`clientSecret`, `c`.`webhooksPer`, `r`.`code`, `r`.`accessToken`,"+
			" `r`.`refreshToken`, `r`.`accessTokenExpirationTimestamp`, `s`.`connector`,"+
			" `s`.`resource`, `s`.`userCursor`, `s`.`settings`\n"+
			"FROM `data_sources` AS `s`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
			"INNER JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
			"WHERE `s`.`id` = ? AND `s`.`workspace` = ?", id, this.workspace).Scan(
		&connectorName, &connectorType, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken,
		&expiration, &connector, &resource, &cursor, &settings)
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

	fh := this.newFirehose(r.Context(), id, connector, resource, connectorType, webhooksPer)
	c, err := connectors.NewAppConnection(fh.ctx, connectorName, &connectors.AppConfig{
		Settings:     settings,
		Firehose:     fh,
		ClientSecret: clientSecret,
		Resource:     resourceCode,
		AccessToken:  accessToken,
	})
	if err != nil {
		return err
	}
	r.Header.Del("Cookie") // remove the cookies from the request.
	c.ServeUserInterface(w, r)

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

// SetUsersQuery sets the users query of the data source with identifier id.
// query must be UTF-8 encoded, it cannot be longer than 16,777,215 runes and
// must contain the ':limit' placeholder.
//
// It returns the ErrDataSourceNotFound error if the data source does not
// exist and the ErrInvalidConnectorType error if the data source is not a
// database.
func (this *DataSources) SetUsersQuery(id int, query string) error {

	if id <= 0 {
		return errors.New("invalid data source identifier")
	}

	if !utf8.ValidString(query) {
		return errors.New("query is not UTF-8 encoded")
	}
	if utf8.RuneCountInString(query) > queryMaxSize {
		return fmt.Errorf("query is longer than %d", queryMaxSize)
	}
	if !strings.Contains(query, ":limit") {
		return errors.New("query does not contain the placeholder \":limit\"")
	}

	result, err := this.myDB.Exec("UPDATE `data_sources` SET `usersQuery` = ? WHERE `id` = ? AND `workspace` = ? AND `type` = 'Database'",
		query, id, this.workspace)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var exists bool
		err = this.myDB.QueryRow("SELECT TRUE FROM `data_sources` WHERE `id` = ? AND `workspace` = ?",
			id, this.workspace).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			return ErrInvalidConnectorType
		}
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
	if id <= 0 {
		return nil, errors.New("invalid data source identifier")
	}
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

// newFirehose returns a new Firehose used to call a connection method.
func (this *DataSources) newFirehose(ctx context.Context, source, connector, resource int, connectorType, webhooksPer string) *firehose {
	fh := &firehose{
		sources:       this,
		source:        source,
		resource:      resource,
		connector:     connector,
		connectorType: connectorType,
		webhooksPer:   webhooksPer,
	}
	fh.ctx, fh.cancel = context.WithCancel(ctx)
	return fh
}

var ErrRecordStop = errors.New("stop record")

// reloadProperties reloads the properties of the data source with identifier
// id. If the data source does not exist it returns the ErrDataSourceNotFound
// error.
func (this *DataSources) reloadProperties(id int) error {

	if id <= 0 {
		return errors.New("invalid data source identifier")
	}

	var typ string
	err := this.myDB.QueryRow("SELECT `type` FROM `data_sources` WHERE `id` = ? AND `workspace` = ?",
		id, this.workspace).Scan(&typ)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrDataSourceNotFound
		}
		return err
	}

	var properties []connectors.Property

	switch typ {
	case "App":

		// TODO(marco) The following code is duplicated in the Import method.
		var connectorName, clientSecret, webhooksPer, resourceCode, accessToken, refreshToken, cursor string
		var connector, resource int
		var settings []byte
		var expiration *time.Time
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `c`.`clientSecret`, `c`.`webhooksPer`, IFNULL(`r`.`code`, ''), IFNULL(`r`.`accessToken`, ''),"+
				" IFNULL(`r`.`refreshToken`, ''), `r`.`accessTokenExpirationTimestamp`, `s`.`connector`,"+
				" `s`.`resource`, `s`.`userCursor`, `s`.`settings`\n"+
				"FROM `data_sources` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"LEFT JOIN `resources` AS `r` ON `r`.`id` = `s`.`resource`\n"+
				"WHERE `s`.`id` = ?", id).Scan(
			&connectorName, &clientSecret, &webhooksPer, &resourceCode, &accessToken, &refreshToken, &expiration,
			&connector, &resource, &cursor, &settings)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrDataSourceNotFound
			}
			return err
		}

		accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(*expiration)

		if accessToken == "" || accessTokenExpired {
			accessToken, err = this.api.apis.refreshOAuthToken(resource)
			if err != nil {
				return err
			}
		}
		fh := this.newFirehose(context.Background(), id, connector, resource, "App", webhooksPer)
		c, err := connectors.NewAppConnection(fh.ctx, connectorName, &connectors.AppConfig{
			Settings:     settings,
			Firehose:     fh,
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})
		if err != nil {
			return err
		}
		properties, _, err = c.Properties()
		if err != nil {
			return err
		}

	case "Database":

		var connectorName, usersQuery string
		var connector int
		var settings []byte
		err = this.myDB.QueryRow(
			"SELECT `c`.`name`, `s`.`connector`, `s`.`settings`, `s`.`usersQuery`\n"+
				"FROM `data_sources` AS `s`\n"+
				"INNER JOIN `connectors` AS `c` ON `c`.`id` = `s`.`connector`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&connectorName, &connector, &settings, &usersQuery)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrDataSourceNotFound
			}
			return err
		}

		usersQuery, err := this.compileQueryWithLimit(usersQuery, 0)
		if err != nil {
			return err
		}
		fh := this.newFirehose(context.Background(), id, connector, 0, "Database", "")
		c, err := connectors.NewDatabaseConnection(fh.ctx, connectorName, settings, fh)
		if err != nil {
			return err
		}
		columns, rows, err := c.Query(usersQuery)
		if err != nil {
			return err
		}
		err = rows.Close()
		if err != nil {
			return err
		}
		properties = make([]connectors.Property, len(columns))
		for i := 0; i < len(properties); i++ {
			properties[i].Name = columns[i].Name
			properties[i].Type = columns[i].Type
		}

	case "FileStream":

		var fileConnectorName, streamConnectorName string
		var fileConnector, streamConnector int
		var fileSettings, streamSettings []byte
		err = this.myDB.QueryRow(
			"SELECT `c1`.`name`, `c2`.`name`, `s`.`connector`, `s`.`stream`, `s`.`settings`, `s`.`streamSettings`\n"+
				"FROM `data_sources` AS `s`\n"+
				"INNER JOIN `connectors` AS `c1` ON `c1`.`id` = `s`.`connector`\n"+
				"INNER JOIN `connectors` AS `c2` ON `c2`.`id` = `s`.`stream`\n"+
				"WHERE `s`.`id` = ?", id).Scan(&fileConnectorName, &streamConnectorName, &fileConnector,
			&streamConnector, &fileSettings, &streamSettings)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrDataSourceNotFound
			}
			return err
		}

		// Connect to the stream connector.
		fh := this.newFirehose(context.Background(), id, streamConnector, 0, "Stream", "")
		stream, err := connectors.NewStreamConnection(fh.ctx, streamConnectorName, streamSettings, fh)
		if err != nil {
			return err
		}
		r, err := stream.Reader("")
		if err != nil {
			return err
		}

		// Connect to the file connector and read only the first record.
		fh = this.newFirehose(context.Background(), id, streamConnector, 0, "File", "")
		file, err := connectors.NewFileConnection(fh.ctx, fileConnectorName, fileSettings, fh)
		if err != nil {
			return err
		}
		var columns []string
		err = file.Read(r, func(record []string) error {
			columns = record
			return ErrRecordStop
		})
		if err != nil && err != ErrRecordStop {
			return err
		}
		properties = make([]connectors.Property, len(columns))
		for i := 0; i < len(properties); i++ {
			properties[i].Name = columns[i]
			properties[i].Type = "string"
		}

	}

	rawProperties, err := json.Marshal(properties)
	if err != nil {
		return fmt.Errorf("cannot marshal the properties of the data source %d : %s", id, err)
	}
	if utf8.RuneCount(rawProperties) > rawPropertiesMaxSize {
		return fmt.Errorf("cannot marshal the properties of the data source %d: data is too large", id)
	}

	_, err = this.myDB.Exec("UPDATE `data_sources` SET `properties` = ? WHERE `id` = ?", rawProperties, id)

	return err
}

// compileQuery compiles the given query and returns it. If the query does not
// contain the limit placeholder it returns the ErrNoLimitPlaceholderInQuery error.
func (this *DataSources) compileQueryWithLimit(query string, limit int) (string, error) {
	return strings.ReplaceAll(query, ":limit", strconv.Itoa(limit)), nil
}
