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
	"sort"
	"strings"
)

type Connectors struct {
	*APIs
	connectors map[int]*Connector
}

// Connector represents a Connector.
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

// Get returns a ConnectorInfo describing the connector with identifier id.
// Returns a ConnectorNotFoundError error if the connector does not exist.
func (this *Connectors) Get(id int) (*ConnectorInfo, error) {
	if id <= 0 || id > maxInt32 {
		return nil, errors.New("invalid connector identifier")
	}
	c, ok := this.connectors[id]
	if !ok {
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

// List returns a list of ConnectionInfo describing all connectors.
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
