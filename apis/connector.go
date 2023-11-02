//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"chichi/apis/errors"
	"chichi/apis/state"
	_connector "chichi/connector"
)

// Connector represents a connector.
type Connector struct {
	apis                   *APIs
	connector              *state.Connector
	ID                     int
	Name                   string
	SourceDescription      string
	DestinationDescription string
	TermForUsers           string
	TermForGroups          string
	Type                   ConnectorType
	HasSheets              bool
	HasSettings            bool
	Icon                   string
	FileExtension          string
	SampleQuery            string
	WebhooksPer            WebhooksPer
	OAuth                  bool
}

// ConnectorTargets represents connector targets.
type ConnectorTargets int

const (
	EventsFlag = 1 << iota
	UsersFlag
	GroupsFlag
)

// MarshalJSON implements the json.Marshaler interface.
func (t ConnectorTargets) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	b.WriteString(`{`)
	_, _ = fmt.Fprintf(b, "\"Events\":%t", t&EventsFlag != 0)
	_, _ = fmt.Fprintf(b, ",\"Users\":%t", t&UsersFlag != 0)
	_, _ = fmt.Fprintf(b, ",\"Groups\":%t", t&GroupsFlag != 0)
	b.WriteString(`}`)
	return b.Bytes(), nil
}

// ConnectorType represents a connector type.
type ConnectorType int

const (
	AppType ConnectorType = iota + 1
	DatabaseType
	FileType
	MobileType
	ServerType
	StorageType
	StreamType
	WebsiteType
)

// MarshalJSON implements the json.Marshaler interface.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + typ.String() + `"`), nil
}

// String returns the string representation of typ.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) String() string {
	switch typ {
	case AppType:
		return "App"
	case DatabaseType:
		return "Database"
	case FileType:
		return "File"
	case MobileType:
		return "Mobile"
	case ServerType:
		return "Server"
	case StorageType:
		return "Storage"
	case StreamType:
		return "Stream"
	case WebsiteType:
		return "Website"
	}
	panic("invalid connector type")
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (typ *ConnectorType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.ConnectorType value: %s", err)
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an api.ConnectorType value", v)
	}
	var t ConnectorType
	switch s {
	case "App":
		t = AppType
	case "Database":
		t = DatabaseType
	case "File":
		t = FileType
	case "Mobile":
		t = MobileType
	case "Server":
		t = ServerType
	case "Storage":
		t = StorageType
	case "Stream":
		t = StreamType
	case "Website":
		t = WebsiteType
	default:
		return fmt.Errorf("invalid apis.ConnectionType: %s", s)
	}
	*typ = t
	return nil
}

// AuthCodeURL returns a URL that directs to the consent page of an OAuth 2.0
// provider. This page requests explicit permissions for the required scopes.
// After that, the provider redirects to the URL specified by redirectURI.
func (this *Connector) AuthCodeURL(redirectURI string) (string, error) {
	this.apis.mustBeOpen()
	if this.connector.OAuth == nil {
		return "", errors.BadRequest("connector %d does not support OAuth", this.connector.ID)
	}
	return this.apis.http.AuthCodeURL(this.connector.OAuth, redirectURI)
}

// openUI opens the UI of the connector, and returns the UI or nil if the
// connector does not have the UI.
//
// If the returned value implements the io.Close interface, it is the caller's
// responsibility to call the Close method
func (this *Connector) openUI(role Role, resource, clientSecret, accessToken string, region state.PrivacyRegion) (_connector.UI, error) {
	var err error
	var connection any
	switch c := this.connector; c.Type {
	case state.AppType:
		connection, err = _connector.RegisteredApp(c.Name).New(&_connector.AppConfig{
			Role:       _connector.Role(role),
			Resource:   resource,
			HTTPClient: this.apis.http.Client(clientSecret, accessToken),
			Region:     _connector.PrivacyRegion(region),
		})
	case state.DatabaseType:
		var database _connector.DatabaseConnection
		database, err = _connector.RegisteredDatabase(c.Name).New(&_connector.DatabaseConfig{
			Role: _connector.Role(role),
		})
		defer database.Close()
		connection = database
	case state.FileType:
		connection, err = _connector.RegisteredFile(c.Name).New(&_connector.FileConfig{
			Role: _connector.Role(role),
		})
	case state.MobileType:
		connection, err = _connector.RegisteredMobile(c.Name).New(&_connector.MobileConfig{
			Role: _connector.Role(role),
		})
	case state.ServerType:
		connection, err = _connector.RegisteredServer(c.Name).New(&_connector.ServerConfig{
			Role: _connector.Role(role),
		})
	case state.StorageType:
		connection, err = _connector.RegisteredStorage(c.Name).New(&_connector.StorageConfig{
			Role: _connector.Role(role),
		})
	case state.StreamType:
		connection, err = _connector.RegisteredStream(c.Name).New(&_connector.StreamConfig{
			Role: _connector.Role(role),
		})
	case state.WebsiteType:
		connection, err = _connector.RegisteredWebsite(c.Name).New(&_connector.WebsiteConfig{
			Role: _connector.Role(role),
		})
	}
	if err != nil {
		return nil, err
	}
	connectorUI, ok := connection.(_connector.UI)
	if !ok {
		if c, ok := connection.(io.Closer); ok {
			_ = c.Close()
		}
		return nil, nil
	}
	return connectorUI, nil
}
