//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type Connectors struct {
	*APIs
	connectors map[int]*Connector
}

// newConnectors returns a new *Connectors value.
func newConnectors(apis *APIs, connectors map[int]*Connector) *Connectors {
	return &Connectors{APIs: apis, connectors: connectors}
}

// Connector represents a connector.
type Connector struct {
	id          int
	name        string
	typ         ConnectorType
	logoURL     string
	webhooksPer WebhooksPer
	oAuth       *ConnectorOAuth
	resources   *resourceSet // keys in resources can change over time.
}

// A ConnectorOAuth represents OAuth data required to authenticate with a
// connector.
type ConnectorOAuth struct {
	URL              string
	ClientID         string
	ClientSecret     string
	TokenEndpoint    string
	DefaultTokenType string
	DefaultExpiresIn int
	ForcedExpiresIn  int
}

// newResourceSet returns a new empty resourceSet.
func newResourceSet() *resourceSet {
	return &resourceSet{m: map[int]*Resource{}}
}

// resourceSet is the set of resources of a connector.
type resourceSet struct {
	sync.Mutex
	m map[int]*Resource
}

// delete deletes the resource with identifier id.
// If the resource does not exist, it does nothing.
func (rs *resourceSet) delete(id int) {
	rs.Lock()
	delete(rs.m, id)
	rs.Unlock()
}

// get returns the resource with identifier id.
// The boolean return value reports whether the resource exists.
func (rs *resourceSet) get(id int) (*Resource, bool) {
	rs.Lock()
	r, ok := rs.m[id]
	rs.Unlock()
	return r, ok
}

// getByCode returns the resource with the given code.
// The boolean return value reports whether the resource exists.
func (rs *resourceSet) getByCode(code string) (*Resource, bool) {
	rs.Lock()
	for _, r := range rs.m {
		if r.code == code {
			rs.Unlock()
			return r, true
		}
	}
	rs.Unlock()
	return nil, false
}

// add adds a resource with the given id, code and OAuth data and returns it.
// If a resource with the same id already exists, add replaces it.
func (rs *resourceSet) add(id int, code, accessToken, refreshToken string, expiresIn time.Time) *Resource {
	r := &Resource{
		id:                id,
		code:              code,
		oAuthAccessToken:  accessToken,
		oAuthRefreshToken: refreshToken,
		oAuthExpiresIn:    expiresIn,
	}
	rs.Lock()
	rs.m[id] = r
	rs.Unlock()
	return r
}

// Resource represents a resource.
type Resource struct {
	id                int
	code              string
	oAuthAccessToken  string
	oAuthRefreshToken string
	oAuthExpiresIn    time.Time
}

// A ConnectorInfo describes a connector as returned by Get and List.
type ConnectorInfo struct {
	ID          int
	Name        string
	Type        ConnectorType
	LogoURL     string
	WebhooksPer WebhooksPer
	OAuth       *ConnectorOAuth
}

// ConnectorType represents a connector type.
type ConnectorType int

const (
	AppType ConnectorType = iota + 1
	DatabaseType
	EventStreamType
	FileType
	MobileType
	ServerType
	StorageType
	WebsiteType
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + typ.String() + `"`), nil
}

// Scan implements the sql.Scanner interface.
func (typ *ConnectorType) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.ConnectorType value", src)
	}
	var t ConnectorType
	switch s {
	case "App":
		t = AppType
	case "Database":
		t = DatabaseType
	case "EventStream":
		t = EventStreamType
	case "File":
		t = FileType
	case "Mobile":
		t = MobileType
	case "Server":
		t = ServerType
	case "Storage":
		t = StorageType
	case "Website":
		t = WebsiteType
	default:
		return fmt.Errorf("invalid api.ConnectionType: %s", s)
	}
	*typ = t
	return nil
}

// String returns the string representation of typ.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) String() string {
	s, err := typ.Value()
	if err != nil {
		panic("invalid connector type")
	}
	return s.(string)
}

// Value implements driver.Valuer interface.
// It returns an error if typ is not a valid ConnectorType.
func (typ ConnectorType) Value() (driver.Value, error) {
	switch typ {
	case AppType:
		return "App", nil
	case DatabaseType:
		return "Database", nil
	case EventStreamType:
		return "EventStream", nil
	case FileType:
		return "File", nil
	case MobileType:
		return "Mobile", nil
	case ServerType:
		return "Server", nil
	case StorageType:
		return "Storage", nil
	case WebsiteType:
		return "Website", nil
	}
	return nil, fmt.Errorf("not a valid ConnectorType: %d", typ)
}

// A ConnectorNotFoundError error indicates that a connector does not exist.
type ConnectorNotFoundError struct {
	Type ConnectorType
}

func (err ConnectorNotFoundError) Error() string {
	if err.Type == 0 {
		return "connector does not exist"
	}
	return fmt.Sprintf("%s connector does not exist", strings.ToLower(err.Type.String()))
}

// refreshOAuth refreshes the OAuth token of the given resource of the
// connector with identifier id. The connector must support OAuth.
//
// If the connector does not exist, it returns a ConnectorNotExistError. If the
// resource does not exist it does nothing.
func (this *Connectors) refreshOAuthToken(id, resource int) (*Resource, error) {

	connector, ok := this.connectors[id]
	if !ok {
		return nil, ConnectorNotFoundError{}
	}
	if connector.oAuth == nil {
		return nil, errors.New("connector does not support OAuth")
	}
	r, ok := connector.resources.get(resource)
	if !ok {
		return nil, nil
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", connector.oAuth.ClientID)
	data.Set("client_secret", connector.oAuth.ClientSecret)
	data.Set("redirect_uri", "https://localhost:9090/admin/oauth/authorize")
	data.Set("refresh_token", r.oAuthRefreshToken)

	req, err := http.NewRequest("POST", connector.oAuth.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
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
				return nil, err
			}
			// TODO(@Andrea): check the status returned by services different
			// from Hubspot.
			if errData.status == "BAD_REFRESH_TOKEN" {
				return nil, ErrCannotGetConnectorAccessToken
			}
		}
		return nil, fmt.Errorf("unexpected status %d returned by connector while trying to get a new access token via refresh token", res.StatusCode)
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
		return nil, err
	}

	// Convert expires_in into a timestamp.
	expiresIn := time.Now().UTC().Add(time.Duration(response.ExpiresIn) * time.Second) // TODO(marco): ExpiresIn should be relative to response time?

	_, err = this.db.Exec(
		"UPDATE resources\n"+
			"SET oauth_access_token = $1, oauth_refresh_token = $2, oauth_expires_in = $3\n"+
			"WHERE id = $4",
		response.AccessToken, response.RefreshToken, expiresIn, r.id)
	if err != nil {
		return nil, err
	}

	r = connector.resources.add(r.id, r.code, response.AccessToken, response.RefreshToken, expiresIn)

	return r, nil
}

// Get returns a ConnectorInfo describing the connector with identifier id.
// Returns a ConnectorNotFoundError error if the connector does not exist.
func (this *Connectors) Get(id int) (*ConnectorInfo, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.New("invalid connector identifier")
	}
	c, err := this.get(id)
	if err != nil {
		return nil, ConnectorNotFoundError{}
	}
	info := ConnectorInfo{
		ID:          c.id,
		Name:        c.name,
		Type:        c.typ,
		LogoURL:     c.logoURL,
		WebhooksPer: c.webhooksPer,
	}
	if c.oAuth != nil {
		info.OAuth = &ConnectorOAuth{}
		*info.OAuth = *c.oAuth
	}
	return &info, nil
}

// List returns a list of ConnectorInfo describing all connectors.
func (this *Connectors) List() []*ConnectorInfo {
	var infos = make([]*ConnectorInfo, 0, len(this.connectors))
	for _, c := range this.connectors {
		info := ConnectorInfo{
			ID:          c.id,
			Name:        c.name,
			Type:        c.typ,
			LogoURL:     c.logoURL,
			WebhooksPer: c.webhooksPer,
		}
		if c.oAuth != nil {
			info.OAuth = &ConnectorOAuth{}
			*info.OAuth = *c.oAuth
		}
		infos = append(infos, &info)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name || infos[i].Name == infos[j].Name && infos[i].ID < infos[j].ID
	})
	return infos
}

var errConnectorNotFound = errors.New("connector does not exist")

// get returns the connector with identifier id.
// Returns the errConnectorNotFound error if the connector does not exist.
func (this *Connectors) get(id int) (*Connector, error) {
	c, ok := this.connectors[id]
	if !ok {
		return nil, errConnectorNotFound
	}
	return c, nil
}
