//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package meergotester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

// These data types are copy-paste of the types defined within the APIs.

type Action struct {
	ID                       int
	Connection               int
	Target                   *Target
	Name                     string
	Enabled                  bool
	EventType                *string
	Running                  bool
	ScheduleStart            *int
	SchedulePeriod           *SchedulePeriod
	InSchema                 types.Type
	OutSchema                types.Type
	Filter                   *Filter
	Transformation           Transformation
	Query                    *string
	Connector                int
	Path                     *string
	Sheet                    *string
	Compression              Compression
	Table                    *string
	TableKeyProperty         *string
	IdentityProperty         *string
	LastChangeTimeProperty   *string
	LastChangeTimeFormat     *string
	FileOrderingPropertyPath string
	ExportMode               *ExportMode
	MatchingProperties       *MatchingProperties
	ExportOnDuplicatedUsers  *bool
}

type ActionToSet struct {
	Name                     string
	Enabled                  bool
	Filter                   *Filter
	InSchema                 types.Type
	OutSchema                types.Type
	Transformation           Transformation
	Connector                string
	Query                    string
	Path                     string
	Sheet                    string
	Compression              Compression
	UIValues                 json.RawMessage `json:",omitempty"`
	TableName                string
	TableKeyProperty         string
	IdentityProperty         string
	LastChangeTimeProperty   string
	LastChangeTimeFormat     string
	FileOrderingPropertyPath string
	ExportMode               *ExportMode
	MatchingProperties       *MatchingProperties
	ExportOnDuplicatedUsers  *bool
}

type Compression string

const (
	NoCompression     Compression = ""
	ZipCompression    Compression = "Zip"
	GzipCompression   Compression = "Gzip"
	SnappyCompression Compression = "Snappy"
)

type Strategy string

type ConnectionToAdd struct {
	Name              string
	Role              Role
	Enabled           bool
	Connector         string
	Strategy          *Strategy
	WebsiteHost       string
	LinkedConnections []int
	SendingMode       *SendingMode
	UIValues          json.RawMessage
}

type DisplayedProperties struct {
	Image       string
	FirstName   string
	LastName    string
	Information string
}

type DummySettings struct {
	URLForDispatchingEvents string
}

type Execution struct {
	ID        int
	Action    int
	StartTime time.Time
	EndTime   *time.Time
	Passed    int
	Failed    int
	Error     string
}

type ExportMode string

// These variables have been introduced to simplify the writing of tests.
var (
	ExportModeCreateOnly     = &[]ExportMode{CreateOnly}[0]
	ExportModeUpdateOnly     = &[]ExportMode{UpdateOnly}[0]
	ExportModeCreateOrUpdate = &[]ExportMode{CreateOrUpdate}[0]
)

const (
	CreateOnly     ExportMode = "CreateOnly"
	UpdateOnly     ExportMode = "UpdateOnly"
	CreateOrUpdate ExportMode = "CreateOrUpdate"
)

type Filter struct {
	Logical    FilterLogical
	Conditions []FilterCondition
}

type FilterLogical string

type FilterCondition struct {
	Property string
	Operator string
	Values   []string
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type User struct {
	ID             uuid.UUID      `json:"id"`
	LastChangeTime time.Time      `json:"lastChangeTime"`
	Properties     map[string]any `json:"properties"`
}

type UserIdentity struct {
	Connection     int       // do not use in tests. Currently, this serves just for the UI.
	Action         int       `json:"action"`
	ID             string    `json:"id"`
	AnonymousIds   []string  `json:"anonymousIds"`
	LastChangeTime time.Time `json:"lastChangeTime"`
}

type LabelValue struct { // copy-pasted from the not-exported type 'labelValue' within package 'apis'.
	Label string
	Value string
}

type Language string

type MatchingProperties struct {
	Internal string
	External types.Property
}

type PrivacyRegion string

const (
	PrivacyRegionNotSpecified PrivacyRegion = ""
	PrivacyRegionEurope       PrivacyRegion = "Europe"
)

type Role int

const (
	Source Role = iota + 1
	Destination
)

func (role Role) MarshalJSON() ([]byte, error) {
	return []byte(`"` + role.String() + `"`), nil
}

func (role Role) String() string {
	switch role {
	case Source:
		return "Source"
	case Destination:
		return "Destination"
	}
	panic("invalid connection role")
}

// SendingMode represents a sending mode.
type SendingMode string

const (
	Cloud    SendingMode = "Cloud"
	Device   SendingMode = "Device"
	Combined SendingMode = "Combined"
)

type SchedulePeriod int

func (period SchedulePeriod) MarshalJSON() ([]byte, error) {
	return []byte(`"` + period.String() + `"`), nil
}

func (period SchedulePeriod) String() string {
	switch period {
	case 5:
		return "5m"
	case 15:
		return "15m"
	case 30:
		return "30m"
	case 60:
		return "1h"
	case 120:
		return "2h"
	case 180:
		return "3h"
	case 360:
		return "6h"
	case 480:
		return "8h"
	case 720:
		return "12h"
	case 1440:
		return "24h"
	}
	panic("invalid schedule period")
}

var null = []byte("null")

func (period *SchedulePeriod) UnmarshalJSON(data []byte) error {
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
		return fmt.Errorf("json: cannot scan a %T value into an api.SchedulePeriod value", v)
	}
	var p SchedulePeriod
	switch s {
	case "5m":
		p = 5
	case "15m":
		p = 15
	case "30m":
		p = 30
	case "1h":
		p = 60
	case "2h":
		p = 120
	case "3h":
		p = 180
	case "6h":
		p = 360
	case "8h":
		p = 480
	case "12h":
		p = 720
	case "24h":
		p = 1440
	default:
		return fmt.Errorf("json: invalid apis.SchedulePeriod: %s", s)
	}
	*period = p
	return nil
}

type Target int

const (
	Events Target = iota + 1
	Users
	Groups
)

func (at Target) MarshalJSON() ([]byte, error) {
	return []byte(`"` + at.String() + `"`), nil
}

func (at Target) String() string {
	switch at {
	case Events:
		return "Events"
	case Users:
		return "Users"
	case Groups:
		return "Groups"
	default:
		panic("invalid target")
	}
}

func (at *Target) UnmarshalJSON(data []byte) error {
	var v any
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("json: cannot scan a %T value into an api.Target value", v)
	}
	switch s {
	case "Events":
		*at = Events
	case "Users":
		*at = Users
	case "Groups":
		*at = Groups
	default:
		return fmt.Errorf("json: invalid apis.Target: %s", s)
	}
	return nil
}

type Transformation struct {
	Mapping  map[string]string
	Function *TransformationFunction
}

type TransformationFunction struct {
	Source        string
	Language      Language
	PreserveJSON  bool
	InProperties  []string
	OutProperties []string
}

type Workspace struct {
	ID                             int
	Name                           string
	UserSchema                     types.Type
	UserPrimarySources             map[string]int
	ResolveIdentitiesOnBatchImport bool
	Identifiers                    []string
	PrivacyRegion                  PrivacyRegion
	DisplayedProperties            DisplayedProperties
}
