//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"

	"chichi/pkg/open2b/sql"
)

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
	var s string
	switch src := src.(type) {
	case string:
		s = src
	case []byte:
		s = string(src)
	default:
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

// Connector represents a connector.
type Connector struct {
	ID          int
	Name        string
	Type        ConnectorType
	LogoURL     string
	WebhooksPer WebhooksPer
	OAuth       struct {
		URL              string
		ClientID         string
		ClientSecret     string
		TokenEndpoint    string
		DefaultTokenType string
		DefaultExpiresIn int
		ForcedExpiresIn  int
	}
}

// Connector returns the connector with the given identifier.
func (apis *APIs) Connector(id int) (*Connector, error) {
	if id <= 0 || id > maxInt32 {
		return nil, errors.New("invalid connector identifier")
	}
	connector := Connector{ID: id}
	err := apis.db.QueryRow(
		"SELECT name, type, oauth_url, logo_url, oauth_client_id, oauth_client_secret, oauth_token_endpoint,"+
			" webhooks_per, oauth_default_token_type, oauth_default_expires_in, oauth_forced_expires_in\n"+
			"FROM connectors\nWHERE id = $1", id).
		Scan(&connector.Name, &connector.Type, &connector.OAuth.URL, &connector.LogoURL, &connector.OAuth.ClientID,
			&connector.OAuth.ClientSecret, &connector.OAuth.TokenEndpoint, &connector.WebhooksPer,
			&connector.OAuth.DefaultTokenType, &connector.OAuth.DefaultExpiresIn, &connector.OAuth.ForcedExpiresIn)
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
	err := apis.db.QueryScan("SELECT id, name, type, oauth_url, logo_url FROM connectors", func(rows *sql.Rows) error {
		var err error
		for rows.Next() {
			var connector Connector
			if err = rows.Scan(&connector.ID, &connector.Name, &connector.Type, &connector.OAuth.URL, &connector.LogoURL); err != nil {
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
