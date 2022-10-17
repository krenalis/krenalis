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
	"io"
	"log"
	"net/http"
	"net/url"
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
// tokens of the resource. If the data source already exists for the given
// connector and resource, it updates the data source.
func (this *DataSources) Add(connector int, refreshToken, accessToken string) error {
	conn, err := this.api.apis.Connector(connector)
	if err != nil {
		return err
	}
	if conn == nil {
		return ErrConnectorNotFound
	}
	c := connectors.Connector(conn.Name, conn.ClientSecret)
	ctx := context.WithValue(context.Background(), connectors.AccessTokenContextKey{}, accessToken)
	resource, err := c.Resource(ctx)
	if err != nil {
		return err
	}
	err = this.myDB.Transaction(func(tx *sql.Tx) error {
		_, err = this.myDB.Exec("INSERT INTO `resources`\n"+
			"SET `connector` = ?, `resource` = ?, `refreshToken` = ?\n"+
			"ON DUPLICATE KEY UPDATE `refreshToken` = ?",
			connector, resource, refreshToken, refreshToken)
		if err != nil {
			return err
		}
		_, err = this.myDB.Exec("INSERT IGNORE INTO `data_sources`\n"+
			"SET `workspace` = ?, `connector` = ?, `resource` = ?\n"+
			"ON DUPLICATE KEY UPDATE `resource` = ?",
			this.workspace, connector, resource, resource)
		return err
	})
	return err
}

// Import starts the import of the users from the data source with the given
// connector. If reimport is false it imports the users from the current
// cursor, otherwise imports all users.
// Returns the ErrConnectorNotFound error if the connector does not exist.
func (this *DataSources) Import(connector int, reimport bool) error {

	if connector <= 0 {
		return errors.New("invalid connector identifier")
	}

	var name, clientSecret, accessToken, refreshToken, resource, cursor string
	var settings []byte
	var expiration time.Time
	err := this.myDB.QueryRow(
		"SELECT `c`.`name`, `c`.`clientSecret`, `r`.`accessToken`, `r`.`refreshToken`, `r`.`accessTokenExpirationTimestamp`, `ds`.`resource`, `ds`.`userCursor`, `ds`.`settings`\n"+
			"FROM `data_sources` AS `ds`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `ds`.`connector`\n"+
			"INNER JOIN `resources` AS `r` ON `r`.`connector` = `ds`.`connector` AND `r`.`resource` = `ds`.`resource`\n"+
			"WHERE `ds`.`workspace` = ? AND `ds`.`connector` = ?", this.workspace, connector).
		Scan(&name, &clientSecret, &accessToken, &refreshToken, &expiration, &resource, &cursor, &settings)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrConnectorNotFound
		}
		return err
	}
	if reimport {
		cursor = ""
	}

	accessTokenExpired := time.Now().UTC().Add(15 * time.Minute).After(expiration)

	if accessToken == "" || accessTokenExpired {
		accessToken, err = this.refreshOAuthToken(connector)
		if err != nil {
			return err
		}
	}

	var properties []string
	err = this.myDB.QueryScan("SELECT `name`\nFROM `data_sources_properties`\nWHERE `workspace` = ? AND `connector` = ?",
		this.workspace, connector, func(rows *sql.Rows) error {
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
		ctx := this.newConnectorContext(context.Background(), connector, resource, accessToken, settings)
		err := conn.Users(ctx, cursor, properties)
		if err != nil {
			log.Printf("[error] call to the Users method of the connector %d failed: %s", connector, err)
		}
	}()

	return nil
}

// List returns all data sources.
func (this *DataSources) List() ([]*DataSource, error) {
	ids := make([]int, 0, 0)
	err := this.myDB.QueryScan("SELECT `connector`\nFROM `data_sources`\nWHERE `workspace` = ?", this.workspace, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var id int
			if err = rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sources := make([]*DataSource, 0, 0)
	if len(ids) == 0 {
		return sources, nil
	}

	stringifiedIDs := make([]string, 0, 0)
	for _, id := range ids {
		stringifiedIDs = append(stringifiedIDs, strconv.Itoa(id))
	}

	err = this.myDB.QueryScan(fmt.Sprintf("SELECT `id`, `name`, `oauthURL`, `logoURL`\nFROM `connectors` WHERE id IN (%s)", strings.Join(stringifiedIDs, ", ")), func(rows *sql.Rows) error {
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
// connector.
// Returns the ErrConnectorNotFound error if the connector does not exist.
func (this *DataSources) Properties(connector int) ([]*DataSourceProperty, error) {

	if connector <= 0 {
		return nil, errors.New("invalid connector identifier")
	}

	var properties []*DataSourceProperty

	stmt := "SELECT `name`, `type`, `label`, `options`\n" +
		"FROM `data_sources_properties`\n" +
		"WHERE `workspace` = ? AND `connector` = ?\n" +
		"ORDER BY `position`"

	err := this.myDB.QueryScan(stmt, this.workspace, connector, func(rows *sql.Rows) error {
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
					return fmt.Errorf("malformed options for connector %d", connector)
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
		err := this.myDB.QueryRow("SELECT TRUE FROM `data_sources`\nWHERE `workspace` = ? AND `connector` = ?", this.workspace, connector).Scan(&exists)
		if err != nil {
			if err == sql.ErrNoRows {
				err = ErrConnectorNotFound
			}
			return nil, err
		}
		properties = []*DataSourceProperty{}
	}

	return properties, nil
}

// SetTransformationFunc sets the transformation function of the data source
// with the given connector.
// Returns the ErrConnectorNotFound error if the connector does not exist or is
// not installed.
func (this *DataSources) SetTransformationFunc(connector int, fn string) error {
	if connector <= 0 {
		return errors.New("invalid connector identifier")
	}
	if !utf8.ValidString(fn) {
		return errors.New("invalid transformation function")
	}
	// TODO(Gianluca): revise table name and column names after the merging of
	// the PR of @retini on OAuth.
	affected, err := this.myDB.Table("DataSources").Update(
		sql.Set{"transformation": fn},
		sql.Where{"workspace": this.workspace, "connector": connector})
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrConnectorNotFound
	}
	return nil
}

// TransformationFunc returns the transformation function of the data source
// with the given connector.
// Returns the ErrConnectorNotFound error if the connector does not exist or is
// not installed.
func (this *DataSources) TransformationFunc(connector int) (string, error) {
	if connector <= 0 {
		return "", errors.New("invalid connector identifier")
	}
	// TODO(Gianluca): revise table name and column names after the merging of
	// the PR of @retini on OAuth.
	row, err := this.myDB.Table("DataSources").Get(sql.Where{"workspace": this.workspace, "connector": connector}, []any{"transformation"})
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", ErrConnectorNotFound
	}
	return row["transformation"].(string), nil
}

// Uninstall uninstalls the data source with the given connector.
// If the connector does not exist, it does nothing.
func (this *DataSources) Uninstall(connector int) error {
	if connector <= 0 {
		return errors.New("invalid connector identifier")
	}
	where := sql.Where{"workspace": this.workspace, "connector": connector}
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		_, err := this.myDB.Table("DataSources").Delete(where)
		if err == nil {
			_, err = this.myDB.Table("DataSourcesProperties").Delete(where)
			_, err = this.myDB.Table("DataSourcesUsers").Delete(where)
		}
		return err
	})
	return err
}

// newConnectorContext returns a context with a Firehose used to call a
// connector method.
func (this *DataSources) newConnectorContext(ctx context.Context, connector int, resource, accessToken string, settings []byte) context.Context {
	fh := &firehose{sources: this, connector: connector, resource: resource}
	fh.context, fh.cancel = context.WithCancel(ctx)
	fh.context = context.WithValue(fh.context, connectors.AccessTokenContextKey{}, accessToken)
	fh.context = context.WithValue(fh.context, connectors.SettingsContextKey{}, settings)
	fh.context = context.WithValue(fh.context, connectors.FirehoseContextKey{}, fh)
	return fh.context
}

// refreshOAuthToken refreshes the OAuth token of the data source with the
// given connector and returns it.
// Returns the ErrConnectorNotFound error if the connector does not exist.
func (this *DataSources) refreshOAuthToken(connector int) (string, error) {

	var clientID, clientSecret, tokenEndpoint, resource, refreshToken string
	err := this.myDB.QueryRow(
		"SELECT `c`.`clientID`, `c`.`clientSecret`, `c`.`tokenEndpoint`, `r`.`resource`, `r`.`refreshToken`\n"+
			"FROM `data_sources` AS `ds`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `ds`.`connector`\n"+
			"INNER JOIN `resources` AS `r` ON `r`.`connector` = `ds`.`connector` AND `r`.`resource` = `ds`.`resource`\n"+
			"WHERE `ds`.`workspace` = ? AND `ds`.`connector` = ?", this.workspace, connector).
		Scan(&clientID, &clientSecret, &tokenEndpoint, &resource, &refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrConnectorNotFound
		}
		return "", err
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", "https://localhost:9090/admin/oauth/authorize")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		if res.StatusCode == http.StatusBadRequest {
			errData := struct {
				status string
			}{}
			err = json.NewDecoder(res.Body).Decode(&errData)
			if err != nil {
				return "", err
			}
			// TODO(@Andrea): check the status returned by services different
			// from Hubspot.
			if errData.status == "BAD_REFRESH_TOKEN" {
				return "", ErrCannotGetConnectorAccessToken
			}
		}
		return "", fmt.Errorf("unexpected status %d returned by connector while trying to get a new access token via refresh token", res.StatusCode)
	}

	response := struct {
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		AccessToken  string `json:"access_token"`
		ExpiresIn    int    `json:"expires_in"`
	}{}
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&response)
	if err != nil {
		return "", err
	}

	// Convert expires_in into a timestamp.
	expiration := time.Now().UTC().Add(time.Duration(response.ExpiresIn) * time.Second) // TODO(marco): ExpiresIn should be relative to response time?

	_, err = this.myDB.Exec(
		"UPDATE `resources`\n"+
			"SET `accessToken` = ?, `refreshToken` = ?, `accessTokenExpirationTimestamp` = ?\n"+
			"WHERE `connector` = ? AND `resource` = ?",
		response.AccessToken, response.RefreshToken, expiration, connector, resource)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}
