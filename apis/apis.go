//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type APIs struct {
	myDB     *sql.DB
	chDB     chDriver.Conn
	Accounts *Accounts
	Users    *Users
}

var hasBeenCalled bool

// New returns an API instance. It can only be called once.
func New(myDB *sql.DB, chDB chDriver.Conn) *APIs {
	if hasBeenCalled {
		panic("apis.New has already been called")
	}
	hasBeenCalled = true
	apis := &APIs{myDB: myDB, chDB: chDB}
	apis.Accounts = &Accounts{apis}
	apis.Users = &Users{apis}
	apis.initSchema()
	return apis
}

type AccountAPI struct {
	account int
	apis    *APIs
	myDB    *sql.DB
	chDB    chDriver.Conn
}

// AsAccount returns an API restricted to the given account.
func (apis *APIs) AsAccount(account int) *AccountAPI {
	api := &AccountAPI{account: account, apis: apis, myDB: apis.myDB, chDB: apis.chDB}
	return api
}

// AsWorkspace returns an API restricted to the given workspace.
func (api *AccountAPI) AsWorkspace(workspace int) *WorkspaceAPI {
	ws := &WorkspaceAPI{workspace: workspace, api: api, myDB: api.myDB, chDB: api.chDB}
	ws.DataSources = &DataSources{ws}
	return ws
}

var importRegexp = regexp.MustCompile(`/apis/data-sources/((\d+)/(import|reimport|properties|transformation))?`)

// ServeHTTP servers the API methods from HTTP.
func (apis *APIs) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	// Read the workspace.
	workspace, _ := strconv.Atoi(r.Header.Get("X-Workspace"))
	if workspace <= 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	// Read the account.
	var account int
	err := apis.myDB.QueryRow("SELECT `account` FROM `workspaces` WHERE `id` = ?", workspace).Scan(&account)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		log.Printf("[error] %s", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	api := apis.AsAccount(account)
	ws := api.AsWorkspace(workspace)

	m := importRegexp.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if m[1] == "" {
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "Method Not Allowed", 405)
			return
		}
		var sources []*DataSource
		sources, err = ws.DataSources.List()
		if err == nil {
			_ = json.NewEncoder(w).Encode(sources)
		}
	} else {
		id, _ := strconv.Atoi(m[2])
		if id <= 0 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		switch m[3] {
		case "properties":
			var properties []DataSourceProperty
			properties, _, err = ws.DataSources.Properties(id)
			if err == nil {
				_ = json.NewEncoder(w).Encode(properties)
			}
		case "transformation":
			if r.Method == "GET" {
				var transformation string
				transformation, err = ws.DataSources.TransformationFunc(id)
				if err == nil {
					w.Header().Set("Content-Type", "text/plain")
					_, _ = io.WriteString(w, transformation)
					return
				}
			} else {
				var transformation []byte
				transformation, err = io.ReadAll(r.Body)
				err = ws.DataSources.SetTransformationFunc(id, string(transformation))
			}
		case "import", "reimport":
			reimport := m[3] == "reimport"
			err = ws.DataSources.Import(id, reimport)
		default:
			panic("unexpected path")
		}
	}
	if err != nil {
		if err == ErrDataSourceNotFound {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		log.Printf("[error] call to %q failed: %s", r.URL.Path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	return
}

// Connector represents a connector.
type Connector struct {
	ID            int
	Name          string
	Type          string
	OauthURL      string
	LogoURL       string
	ClientID      string
	ClientSecret  string
	TokenEndpoint string
	WebhooksPer   string
}

// Connector returns the connector with the given identifier.
func (apis *APIs) Connector(id int) (*Connector, error) {
	connector := Connector{ID: id}
	err := apis.myDB.QueryRow("SELECT `name`, `type`, `oauthURL`, `logoURL`, `clientID`, `clientSecret`,"+
		" `tokenEndpoint`, `webhooksPer`\nFROM `connectors`\nWHERE `id` = ?", id).Scan(
		&connector.Name, &connector.Type, &connector.OauthURL, &connector.LogoURL, &connector.ClientID,
		&connector.ClientSecret, &connector.TokenEndpoint, &connector.WebhooksPer)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &connector, nil
}

// Connectors returns all connectors.
func (apis *APIs) Connectors() ([]*Connector, error) {
	connectors := []*Connector{}
	err := apis.myDB.QueryScan("SELECT `id`, `name`, `type`, `oauthURL`, `logoURL`\nFROM `connectors`", func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var connector Connector
			if err = rows.Scan(&connector.ID, &connector.Name, &connector.Type, &connector.OauthURL, &connector.LogoURL); err != nil {
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

// refreshOAuthToken refreshes the access token of the resource with identifier
// id.
// Returns the ErrResourceNotFound error if the resource does not exist.
func (apis *APIs) refreshOAuthToken(resource int) (string, error) {

	var clientID, clientSecret, tokenEndpoint, refreshToken string
	err := apis.myDB.QueryRow(
		"SELECT `c`.`clientID`, `c`.`clientSecret`, `c`.`tokenEndpoint`, `r`.`refreshToken`\n"+
			"FROM `resources` AS `r`\n"+
			"INNER JOIN `connectors` AS `c` ON `c`.`id` = `r`.`connector`\n"+
			"WHERE `r`.`id` = ?", resource).
		Scan(&clientID, &clientSecret, &tokenEndpoint, &refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrResourceNotFound
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

	_, err = apis.myDB.Exec(
		"UPDATE `resources`\n"+
			"SET `accessToken` = ?, `refreshToken` = ?, `accessTokenExpirationTimestamp` = ?\n"+
			"WHERE `id` = ?",
		response.AccessToken, response.RefreshToken, expiration, resource)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}

func (apis *APIs) initSchema() {

	apis.myDB.Scheme("Accounts", "accounts", struct {
		id          int
		name        string
		email       string
		password    string
		internalIPs string
	}{})

	apis.myDB.Scheme("Connectors", "connectors", struct {
		id            int
		oauthURL      string
		logoURL       string
		clientID      string
		clientSecret  string
		tokenEndpoint string
		webhooksPer   string
	}{})

	apis.myDB.Scheme("DataSources", "data_sources", struct {
		account                        int
		connector                      int
		accessToken                    string
		refreshToken                   string
		accessTokenExpirationTimestamp string
		transformation                 string
		userCursor                     string
		settings                       string
		properties                     string
		usedProperties                 string
	}{})

	apis.myDB.Scheme("DataSourcesUsers", "data_sources_users", struct {
		workspace int
		connector int
		user      string
		data      string
	}{})

	apis.myDB.Scheme("Devices", "devices", struct {
		property int
		id       string
		user     int
	}{})

	apis.myDB.Scheme("Domains", "domains", struct {
		property int
		name     string
	}{})

	apis.myDB.Scheme("Properties", "properties", struct {
		id      int
		code    string
		account int
	}{})

	apis.myDB.Scheme("SmartEvents", "smart_events", struct {
		property int
		id       int
		name     string
		event    string
		pages    string
		buttons  string
	}{})

	apis.myDB.Scheme("Users", "users", struct {
		property int
		id       int
		device   string
	}{})

	apis.myDB.Scheme("Workspaces", "workspaces", struct {
		id          int
		account     int
		name        string
		userSchema  string
		groupSchema string
		eventSchema string
	}{})

}

// DeprecatedProperty returns an instance of DeprecatedProperties which operates
// on the given property.
func (api *AccountAPI) DeprecatedProperty(property int) *DeprecatedProperties {
	properties := &DeprecatedProperties{
		AccountAPI: api,
		id:         property,
	}
	properties.SmartEvents = &SmartEvents{properties}
	properties.Visualization = &Visualization{properties}
	return properties
}
