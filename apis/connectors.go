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

	"chichi/connectors"
	"chichi/pkg/open2b/sql"
)

type Connectors struct {
	*APIs
}

type Connector struct {
	ID            int
	Name          string
	OauthURL      string
	LogoURL       string
	ClientID      string
	ClientSecret  string
	TokenEndpoint string
}

func (this *Connectors) Find() ([]*Connector, error) {
	connectors := make([]*Connector, 0, 0)
	err := this.myDB.QueryScan("SELECT `id`, `name`, `oauth_url`, `logo_url`\nFROM `connectors`", func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var connector Connector
			if err = rows.Scan(&connector.ID, &connector.Name, &connector.OauthURL, &connector.LogoURL); err != nil {
				return err
			}
			connectors = append(connectors, &connector)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connectors, nil
}

// Install installs a connector given its identifier and the OAuth refresh
// token. If the connector is already installed it does not install it but
// updates its refresh token and removes the access token.
func (this *Connectors) Install(id int, refreshToken string) error {
	var account = 1 // TODO(marco)
	_, err := this.myDB.Exec("INSERT INTO `account_connectors`\n"+
		"SET `account` = ?, `connector` = ?, `refresh_token` = ?\n"+
		"ON DUPLICATE KEY UPDATE `access_token` = '', `refresh_token` = ?, `access_token_expiration_timestamp` = ''",
		account, id, refreshToken, refreshToken)
	return err
}

var ErrConnectorNotFound = errors.New("connector does not exist")

var ErrCannotGetConnectorAccessToken = fmt.Errorf("cannot get access token")

// Import starts the import of the users from the connector with identifier id.
// If reimport is false it imports the users from the current cursor, otherwise
// imports all users.
// Returns the ErrConnectorNotFound error if the connector does not exist.
func (this *Connectors) Import(id int, reimport bool) error {

	if id <= 0 {
		return errors.New("invalid connector identifier")
	}

	var account = 1 // TODO(marco)

	var name, clientSecret, accessToken, refreshToken, cursor string
	var expiration time.Time
	err := this.myDB.QueryRow(
		"SELECT `name`, `client_secret`, `access_token`, `refresh_token`, `access_token_expiration_timestamp`, `user_cursor`\n"+
			"FROM `connectors`\n"+
			"INNER JOIN `account_connectors` ON `connector` = `id`\n"+
			"WHERE `id` = ? AND `account` = ?", id, account).Scan(&name, &clientSecret, &accessToken, &refreshToken, &expiration, &cursor)
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
		accessToken, err = this.refreshOAuthToken(id)
		if err != nil {
			return err
		}
	}

	fh := this.NewFirehose(id, account)
	connector := connectors.Connector(context.Background(), name, clientSecret, fh)

	go func() {
		err := connector.Users(accessToken, cursor)
		if err != nil {
			log.Printf("[error] call to the Users method of the connector %d failed: %s", id, err)
		}
	}()

	return nil
}

// PropertyType represents the type of a property.
type PropertyType string

// ConnectorPropertyOption represents an option of a connector property.
type ConnectorPropertyOption struct {
	Label string
	Value string
}

// ConnectorProperty represents a connector property.
type ConnectorProperty struct {
	Name    string
	Type    PropertyType
	Label   string
	Options []ConnectorPropertyOption
}

// Properties returns the properties of the connector with identifier id.
// Returns the ErrConnectorNotFound error if the connector does not exist.
func (this *Connectors) Properties(id int) ([]*ConnectorProperty, error) {

	if id <= 0 {
		return nil, errors.New("invalid connector identifier")
	}

	var account = 1 // TODO(marco)

	var properties []*ConnectorProperty

	stmt := "SELECT `name`, `type`, `label`, `options`\n" +
		"FROM `connectors_properties`\n" +
		"WHERE `account` = ? AND `connector` = ?\n" +
		"ORDER BY `position`"

	err := this.myDB.QueryScan(stmt, account, id, func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var property ConnectorProperty
			var options []byte
			if err = rows.Scan(&property.Name, &property.Type, &property.Label, &options); err != nil {
				return err
			}
			if len(options) > 0 {
				property.Options = []ConnectorPropertyOption{}
				err := json.Unmarshal(options, &property.Options)
				if err != nil {
					return fmt.Errorf("malformed options for connector %d", id)
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
		err := this.myDB.QueryRow("SELECT TRUE FROM `account_connectors`\nWHERE `account` = ? AND `connector` = ?", account, id).Scan(&exists)
		if err != nil {
			if err == sql.ErrNoRows {
				err = ErrConnectorNotFound
			}
			return nil, err
		}
		properties = []*ConnectorProperty{}
	}

	return properties, nil
}

func (this *Connectors) Get(id int) (*Connector, error) {
	connector := Connector{ID: id}
	err := this.myDB.QueryRow("SELECT `name`, `oauth_url`, `logo_url`, `client_id`, `client_secret`, `token_endpoint`\nFROM `connectors`\nWHERE `id` = ?", id).
		Scan(&connector.Name, &connector.OauthURL, &connector.LogoURL, &connector.ClientID, &connector.ClientSecret, &connector.TokenEndpoint)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &connector, nil
}

func (this *Connectors) FindAccountConnectors(accountID int) ([]*Connector, error) {
	ids := make([]int, 0, 0)
	err := this.myDB.QueryScan("SELECT `connector`\nFROM `account_connectors`\nWHERE account = ?", accountID, func(rows *sql.Rows) error {
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

	connectors := make([]*Connector, 0, 0)
	if len(ids) == 0 {
		return connectors, nil
	}

	stringifiedIDs := make([]string, 0, 0)
	for _, id := range ids {
		stringifiedIDs = append(stringifiedIDs, strconv.Itoa(id))
	}

	err = this.myDB.QueryScan(fmt.Sprintf("SELECT `id`, `name`, `oauth_url`, `logo_url`\nFROM `connectors` WHERE id IN (%s)", strings.Join(stringifiedIDs, ", ")), func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var connector Connector
			if err = rows.Scan(&connector.ID, &connector.Name, &connector.OauthURL, &connector.LogoURL); err != nil {
				return err
			}
			connectors = append(connectors, &connector)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connectors, nil
}

func (this *Connectors) GetAccountConnector(accountID int, connectorID int) (string, string, *time.Time, error) {
	var accessToken, refreshToken string
	var expiration time.Time
	err := this.myDB.QueryRow("SELECT `access_token`, `refresh_token`, `access_token_expiration_timestamp`\nFROM `account_connectors`\nWHERE `account` = ? AND `connector` = ?", accountID, connectorID).
		Scan(&accessToken, &refreshToken, &expiration)
	if err != nil {
		return "", "", nil, err
	}
	return accessToken, refreshToken, &expiration, nil
}

// Uninstall uninstalls the connector with the given identifier.
// If the connector does not exist, it does nothing.
func (this *Connectors) Uninstall(id int) error {
	if id <= 0 {
		return errors.New("invalid connector identifier")
	}
	var account = 1 // TODO(marco)
	where := sql.Where{"account": account, "connector": id}
	err := this.myDB.Transaction(func(tx *sql.Tx) error {
		_, err := this.myDB.Table("AccountConnectors").Delete(where)
		if err == nil {
			_, err = this.myDB.Table("ConnectorsProperties").Delete(where)
		}
		return err
	})
	return err
}

// TransformationFunc returns the transformation function of the connector with
// identifier id.
// Returns the ErrConnectorNotFound error if the connector does not exist or is
// not installed.
func (this *Connectors) TransformationFunc(id int) (string, error) {
	var account = 1 // TODO(marco)
	// TODO(Gianluca): revise table name and column names after the merging of
	// the PR of @retini on OAuth.
	row, err := this.myDB.Table("AccountConnectors").Get(sql.Where{"account": account, "connector": id}, []any{"transformation"})
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", ErrConnectorNotFound
	}
	return row["transformation"].(string), nil
}

// SetTransformationFunc sets the transformation function of the connector with
// identifier id.
// Returns the ErrConnectorNotFound error if the connector does not exist or is
// not installed.
func (this *Connectors) SetTransformationFunc(id int, fn string) error {
	var account = 1 // TODO(marco)
	// TODO(Gianluca): revise table name and column names after the merging of
	// the PR of @retini on OAuth.
	affected, err := this.myDB.Table("AccountConnectors").Update(
		sql.Set{"transformation": fn},
		sql.Where{"account": account, "connector": id})
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrConnectorNotFound
	}
	return nil
}

// refreshOAuthToken refreshes the OAuth token and returns it.
// Returns the ErrConnectorNotFound error if the connector does not exist.
func (this *Connectors) refreshOAuthToken(id int) (string, error) {

	var account = 1 // TODO(marco)

	var clientID, clientSecret, refreshToken, tokenEndpoint string
	err := this.myDB.QueryRow(
		"SELECT `client_id`, `client_secret`, `refresh_token`, `token_endpoint`\n"+
			"FROM `connectors`\n"+
			"INNER JOIN `account_connectors` ON `connector` = `id`\n"+
			"WHERE `id` = ? AND `account` = 1", id).Scan(&clientID, &clientSecret, &refreshToken, &tokenEndpoint)
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
		"UPDATE `account_connectors`\n"+
			"SET `access_token` = ?, `refresh_token` = ?, `access_token_expiration_timestamp` = ?\n"+
			"WHERE `account` = ? AND `connector` = ?",
		response.AccessToken, response.RefreshToken, expiration, account, id)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}
