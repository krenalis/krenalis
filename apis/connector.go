//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"chichi/apis/errors"
	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/ui"

	"github.com/jxskiss/base62"
)

// Connector represents a connector.
type Connector struct {
	connector   *state.Connector
	ID          int
	Name        string
	Type        ConnectorType
	HasSettings bool
	LogoURL     string
	WebhooksPer WebhooksPer
	OAuth       *ConnectorOAuth
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

// ServeUI serves the user interface for the connector with the given role.
// event is the event and values contains the form values in JSON format.
// oAuth is the OAuth token returned by the (*Workspace).OAuth method, it is
// required if the connector requires OAuth.
//
// If the event does not exist, it returns an errors.UnprocessableError error
// with code EventNotExist.
func (this *Connector) ServeUI(event string, values []byte, role ConnectionRole, oAuth string) ([]byte, error) {

	c := this.connector

	if (oAuth == "") != (c.OAuth == nil) {
		if oAuth == "" {
			return nil, errors.BadRequest("OAuth is required by connector %d", c.ID)
		}
		return nil, errors.BadRequest("connector %d does not support OAuth", c.ID)
	}

	// Decode oAuth.
	var r authorizedResource
	if oAuth != "" {
		data, err := base62.DecodeString(oAuth)
		if err != nil {
			return nil, errors.BadRequest("oAuth is not valid")
		}
		err = json.Unmarshal(data, &r)
		if err != nil {
			return nil, errors.BadRequest("oAuth is not valid")
		}
	}

	var err error
	var connection any

	switch c.Type {
	case state.AppType:

		var clientSecret, resourceCode, accessToken string
		if oAuth != "" {
			clientSecret = c.OAuth.ClientSecret
			resourceCode = r.Code
			accessToken = r.AccessToken
		}

		connection, err = _connector.RegisteredApp(c.Name).Open(context.Background(), &_connector.AppConfig{
			Role:         _connector.Role(role),
			ClientSecret: clientSecret,
			Resource:     resourceCode,
			AccessToken:  accessToken,
		})

	default:

		ctx := context.Background()

		switch c.Type {
		case state.DatabaseType:
			connection, err = _connector.RegisteredDatabase(c.Name).Open(ctx, &_connector.DatabaseConfig{
				Role: _connector.Role(role),
			})
		case state.FileType:
			connection, err = _connector.RegisteredFile(c.Name).Open(ctx, &_connector.FileConfig{
				Role: _connector.Role(role),
			})
		case state.MobileType:
			connection, err = _connector.RegisteredMobile(c.Name).Open(ctx, &_connector.MobileConfig{
				Role: _connector.Role(role),
			})
		case state.ServerType:
			connection, err = _connector.RegisteredServer(c.Name).Open(ctx, &_connector.ServerConfig{
				Role: _connector.Role(role),
			})
		case state.StorageType:
			connection, err = _connector.RegisteredStorage(c.Name).Open(ctx, &_connector.StorageConfig{
				Role: _connector.Role(role),
			})
		case state.StreamType:
			connection, err = _connector.RegisteredStream(c.Name).Open(ctx, &_connector.StreamConfig{
				Role: _connector.Role(role),
			})
		case state.WebsiteType:
			connection, err = _connector.RegisteredWebsite(c.Name).Open(ctx, &_connector.WebsiteConfig{
				Role: _connector.Role(role),
			})
		}

	}
	if err != nil {
		return nil, err
	}
	connectionUI, ok := connection.(_connector.UI)
	if !ok {
		return nil, errors.BadRequest("connector %d does not have a UI", c.ID)
	}

	// TODO: check and delete alternative fieldsets keys that have 'null' value
	// before saving to database
	form, alert, err := connectionUI.ServeUI(event, values)
	if err != nil {
		if err == ui.ErrEventNotExist {
			err = errors.Unprocessable(EventNotExist, "UI event %q does not exist for %s connector", event, c.Name)
		}
		return nil, err
	}

	return marshalUIFormAlert(form, alert, ui.Role(role))
}
