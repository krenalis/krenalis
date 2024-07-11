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

	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/state"
)

// Connector represents a connector.
type Connector struct {
	apis                   *APIs
	connector              *state.Connector
	Name                   string
	SourceDescription      string
	DestinationDescription string
	TermForUsers           string
	TermForGroups          string
	Type                   ConnectorType
	Targets                Targets
	SendingMode            *SendingMode
	HasSheets              bool
	HasUI                  bool
	IdentityIDLabel        string
	Icon                   string
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
		return err
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into an api.ConnectorType value", v)
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
		return fmt.Errorf("json: invalid apis.ConnectionType: %s", s)
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

// SendingMode represents a sending mode.
type SendingMode string

const (
	Cloud    SendingMode = "Cloud"
	Device   SendingMode = "Device"
	Combined SendingMode = "Combined"
)

func isValidSendingMode(sm SendingMode) bool {
	switch sm {
	case "Cloud", "Device", "Combined":
		return true
	}
	return false
}

// AuthCodeURL returns a URL that directs to the consent page of an OAuth 2.0
// provider. This page requests explicit permissions for the scopes required by
// the provided role, which could be Source or Destination.
//
// After granting permissions, the provider redirects the user to the URL
// specified by redirectURI.
func (this *Connector) AuthCodeURL(role Role, redirectURI string) (string, error) {
	this.apis.mustBeOpen()
	if this.connector.OAuth == nil {
		return "", errors.BadRequest("connector %s does not support OAuth", this.connector.Name)
	}
	if role != Source && role != Destination {
		return "", errors.BadRequest("role %q is not valid", role)
	}
	return this.apis.connectors.AuthorizationEndpoint(this.connector, state.Role(role), redirectURI)
}
