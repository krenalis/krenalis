//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package core

import (
	"bytes"
	"fmt"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/internal/connectors"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/json"
)

// Connector represents a connector.
type Connector struct {
	core            *Core
	connector       *state.Connector
	Code            string                `json:"code"`
	Label           string                `json:"label"`
	Type            ConnectorType         `json:"type"`
	Categories      []string              `json:"categories"`
	AsSource        *SourceConnector      `json:"asSource"`
	AsDestination   *DestinationConnector `json:"asDestination"`
	IdentityIDLabel string                `json:"identityIDLabel"`
	HasSheets       bool                  `json:"hasSheets"`
	FileExtension   string                `json:"fileExtension"`
	OAuth           *ConnectorOAuth       `json:"oauth"`
	Terms           ConnectorTerms        `json:"terms"`
	Icon            string                `json:"icon"`
	Strategies      bool                  `json:"strategies"`
}

type ConnectorOAuth struct {
	Configured        bool `json:"configured"`
	Disallow127_0_0_1 bool `json:"disallow127_0_0_1"`
	DisallowLocalhost bool `json:"disallowLocalhost"`
}

type ConnectorTerms struct {
	User  string `json:"user"`
	Users string `json:"users"`
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
	App ConnectorType = iota + 1
	Database
	File
	FileStorage
	SDK
	Stream
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
	case App:
		return "App"
	case Database:
		return "Database"
	case File:
		return "File"
	case FileStorage:
		return "FileStorage"
	case SDK:
		return "SDK"
	case Stream:
		return "Stream"
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
		t = App
	case "Database":
		t = Database
	case "File":
		t = File
	case "FileStorage":
		t = FileStorage
	case "SDK":
		t = SDK
	case "Stream":
		t = Stream
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

// AuthCodeURL returns a URL that directs to the consent page of an OAuth 2.0
// provider. This page requests explicit permissions for the scopes required by
// the provided role, either Source or Destination.
//
// After granting permissions, the provider redirects the user to the URL
// specified by redirectURI.
//
// If the connector is not configured for OAuth (i.e., ClientID or ClientSecret
// is empty), it returns an errors.UnavailableError error.
func (this *Connector) AuthCodeURL(role Role, redirectURI string) (string, error) {
	this.core.mustBeOpen()
	if this.connector.OAuth == nil {
		return "", errors.BadRequest("connector %s does not support OAuth", this.connector.Code)
	}
	if role != Source && role != Destination {
		return "", errors.BadRequest("role %q is not valid", role)
	}
	authCodeURL, err := this.core.connectors.AuthorizationEndpoint(this.connector, state.Role(role), redirectURI)
	if err != nil {
		if err, ok := err.(*connectors.UnavailableError); ok {
			return "", errors.Unavailable("%s", err)
		}
		return "", err
	}
	return authCodeURL, nil
}
