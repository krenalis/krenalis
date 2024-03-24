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

	"chichi/apis/errors"
	"chichi/apis/state"
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
	Targets                Targets
	HasSheets              bool
	HasSettings            bool
	Icon                   string
	ExternalIDLabel        string
	FileExtension          string
	SampleQuery            string
	WebhooksPer            WebhooksPer
	OAuth                  bool
}

// ConnectorType represents a connector type.
type ConnectorType int

const (
	AppType ConnectorType = iota + 1
	DatabaseType
	FileType
	FileStorageType
	MobileType
	ServerType
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
	case FileStorageType:
		return "FileStorage"
	case MobileType:
		return "Mobile"
	case ServerType:
		return "Server"
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
	case "FileStorage":
		t = FileStorageType
	case "Mobile":
		t = MobileType
	case "Server":
		t = ServerType
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

// Targets represents the supported targets by a connector.
type Targets struct {
	Users  bool
	Groups bool
	Events bool
}

// AuthCodeURL returns a URL that directs to the consent page of an OAuth 2.0
// provider. This page requests explicit permissions for the required scopes.
// After that, the provider redirects to the URL specified by redirectURI.
func (this *Connector) AuthCodeURL(redirectURI string) (string, error) {
	this.apis.mustBeOpen()
	if this.connector.OAuth == nil {
		return "", errors.BadRequest("connector %d does not support OAuth", this.connector.ID)
	}
	return this.apis.connectors.AuthorizationEndpoint(this.connector, redirectURI)
}
