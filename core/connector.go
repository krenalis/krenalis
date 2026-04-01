// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"

	"github.com/krenalis/krenalis/core/internal/connections"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// Connector represents a connector.
type Connector struct {
	core          *Core
	connector     *state.Connector
	Code          string                `json:"code"`
	Label         string                `json:"label"`
	Type          ConnectorType         `json:"type"`
	Categories    []string              `json:"categories"`
	AsSource      *SourceConnector      `json:"asSource"`
	AsDestination *DestinationConnector `json:"asDestination"`
	HasSheets     bool                  `json:"hasSheets"`
	FileExtension string                `json:"fileExtension"`
	OAuth         *ConnectorOAuth       `json:"oauth"`
	Terms         ConnectorTerms        `json:"terms"`
	Strategies    bool                  `json:"strategies"`
}

// AuthURL returns a URL that directs to the consent page of an OAuth 2.0
// provider. This page requests explicit permissions for the scopes required by
// the provided role, either Source or Destination.
//
// After granting permissions, the provider redirects the user to the URL
// specified by redirectURI.
//
// If the connector is not configured for OAuth (i.e., ClientID or ClientSecret
// is empty), it returns an errors.UnavailableError error.
func (this *Connector) AuthURL(role Role, redirectURI string) (string, error) {
	this.core.mustBeOpen()
	if this.connector.OAuth == nil {
		return "", errors.BadRequest("connector %s does not support OAuth", this.connector.Code)
	}
	if role != Source && role != Destination {
		return "", errors.BadRequest("role %q is not valid", role)
	}
	authURL, err := this.core.connections.AuthorizationEndpoint(this.connector, state.Role(role), redirectURI)
	if err != nil {
		if err, ok := err.(*connections.UnavailableError); ok {
			return "", errors.Unavailable("%s", err)
		}
		return "", err
	}
	return authURL, nil
}

type ConnectorOAuth struct {
	Configured        bool `json:"configured"`
	Disallow127_0_0_1 bool `json:"disallow127_0_0_1"`
	DisallowLocalhost bool `json:"disallowLocalhost"`
}

type ConnectorTerms struct {
	User   string `json:"user"`
	Users  string `json:"users"`
	UserID string `json:"userID"`
	// Group  string `json:"group"`  TODO(marco): Implement groups
	// Groups string `json:"groups"`
}

type SourceConnector struct {
	Targets     []Target    `json:"targets"`
	HasSettings bool        `json:"hasSettings"`
	SampleQuery string      `json:"sampleQuery"`
	WebhooksPer WebhooksPer `json:"webhooksPer"`
	Summary     string      `json:"summary"`
}

type DestinationConnector struct {
	Description string       `json:"description"`
	Targets     []Target     `json:"targets"`
	HasSettings bool         `json:"hasSettings"`
	SendingMode *SendingMode `json:"sendingMode"`
	Summary     string       `json:"summary"`
}

// ConnectorType represents a connector type.
type ConnectorType int

const (
	Application ConnectorType = iota + 1
	Database
	File
	FileStorage
	MessageBroker
	SDK
	Webhook
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
	case Application:
		return "Application"
	case Database:
		return "Database"
	case File:
		return "File"
	case FileStorage:
		return "FileStorage"
	case MessageBroker:
		return "MessageBroker"
	case SDK:
		return "SDK"
	case Webhook:
		return "Webhook"
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
		return fmt.Errorf("json: cannot scan a %T value into an core.ConnectorType value", v)
	}
	var t ConnectorType
	switch s {
	case "Application":
		t = Application
	case "Database":
		t = Database
	case "File":
		t = File
	case "FileStorage":
		t = FileStorage
	case "MessageBroker":
		t = MessageBroker
	case "SDK":
		t = SDK
	case "Webhook":
		t = Webhook
	default:
		return fmt.Errorf("json: invalid core.ConnectorType: %s", s)
	}
	*typ = t
	return nil
}

// SendingMode represents a sending mode.
type SendingMode string

const (
	Client          SendingMode = "Client"
	Server          SendingMode = "Server"
	ClientAndServer SendingMode = "ClientAndServer"
)

func isValidSendingMode(sm SendingMode) bool {
	switch sm {
	case "Client", "Server", "ClientAndServer":
		return true
	}
	return false
}
