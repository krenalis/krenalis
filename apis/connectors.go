//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
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

// String returns the string representation of typ.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) String() string {
	switch typ {
	case AppType:
		return "App"
	case DatabaseType:
		return "Database"
	case EventStreamType:
		return "EventStream"
	case FileType:
		return "File"
	case MobileType:
		return "Mobile"
	case ServerType:
		return "Server"
	case StorageType:
		return "Storage"
	case WebsiteType:
		return "Website"
	}
	panic("invalid connector type")
}

// MarshalJSON implements the json.Marshaler interface.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + typ.String() + `"`), nil
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
	err := apis.myDB.QueryRow(
		"SELECT `name`, CAST(`type` AS UNSIGNED), `oauth_url`, `logo_url`, `oauth_client_id`, `oauth_client_secret`,"+
			" `oauth_token_endpoint`, `webhooks_per` - 1, `oauth_default_token_type`, `oauth_default_expires_in`,"+
			" `oauth_forced_expires_in`\n"+
			"FROM `connectors`\nWHERE `id` = ?", id).
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
	err := apis.myDB.QueryScan("SELECT `id`, `name`, CAST(`type` AS UNSIGNED), `oauth_url`, `logo_url`\nFROM `connectors`", func(rows *sql.Rows) error {
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
