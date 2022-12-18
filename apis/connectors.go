//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"

	"chichi/apis/errors"
)

type Connectors struct {
	*APIs
	state *connectorsState
}

// newConnectors returns a new *Connectors value.
func newConnectors(apis *APIs, state *connectorsState) *Connectors {
	return &Connectors{APIs: apis, state: state}
}

// Connector represents a connector.
type Connector struct {
	id          int
	name        string
	typ         ConnectorType
	logoURL     string
	webhooksPer WebhooksPer
	oAuth       *ConnectorOAuth
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

// UnmarshalJSON implements the json.Unmarshaler interface.
func (typ *ConnectorType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var s any
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a apis.ConnectorType value: %s", err)
	}
	return typ.Scan(s)
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

// Get returns a ConnectorInfo describing the connector with identifier id.
// Returns a ConnectorNotFoundError error if the connector does not exist.
func (this *Connectors) Get(id int) (*ConnectorInfo, error) {
	if id < 1 || id > maxInt32 {
		return nil, errors.BadRequest("connector identifier %d is not valid", id)
	}
	c, err := this.state.Get(id)
	if err != nil {
		return nil, errors.NotFound("connector %d does not exist", id)
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

// list returns all the connectors.
func (this *Connectors) list() []*Connector {
	this.state.Lock()
	connectors := make([]*Connector, len(this.state.ids))
	i := 0
	for _, c := range this.state.ids {
		connectors[i] = c
		i++
	}
	this.state.Unlock()
	return connectors
}

// List returns a list of ConnectorInfo describing all connectors.
func (this *Connectors) List() []*ConnectorInfo {
	connectors := this.state.List()
	var infos = make([]*ConnectorInfo, len(connectors))
	for i, c := range connectors {
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
		infos[i] = &info
	}
	sort.Slice(infos, func(i, j int) bool {
		a, b := infos[i], infos[j]
		return a.Name < b.Name || a.Name == b.Name && a.ID < b.ID
	})
	return infos
}
