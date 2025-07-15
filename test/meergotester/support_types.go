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
	"errors"
	"fmt"
	"time"

	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

// These data types are copy-paste of the types defined within the APIs.

type Action struct {
	ID                   int             `json:"id"`
	Connection           int             `json:"connection"`
	Target               *Target         `json:"target"`
	Name                 string          `json:"name"`
	Enabled              bool            `json:"enabled"`
	EventType            *string         `json:"eventType"`
	Running              bool            `json:"running"`
	ScheduleStart        *int            `json:"scheduleStart"`
	SchedulePeriod       *SchedulePeriod `json:"schedulePeriod"`
	InSchema             types.Type      `json:"inSchema"`
	OutSchema            types.Type      `json:"outSchema"`
	Filter               *Filter         `json:"filter"`
	Transformation       Transformation  `json:"transformation"`
	Query                *string         `json:"query"`
	Connector            int             `json:"connector"`
	Path                 *string         `json:"path"`
	Sheet                *string         `json:"sheet"`
	Compression          Compression     `json:"compression"`
	OrderBy              string          `json:"orderBy"`
	ExportMode           *ExportMode     `json:"exportMode"`
	Matching             *Matching       `json:"matching"`
	UpdateOnDuplicates   *bool           `json:"updateOnDuplicates"`
	TableName            *string         `json:"tableName"`
	TableKey             *string         `json:"tableKey"`
	IdentityColumn       *string         `json:"identityColumn"`
	LastChangeTimeColumn *string         `json:"lastChangeTimeColumn"`
	LastChangeTimeFormat *string         `json:"lastChangeTimeFormat"`
}

type ActionToSet struct {
	Name                 string          `json:"name"`
	Enabled              bool            `json:"enabled"`
	Filter               *Filter         `json:"filter"`
	InSchema             types.Type      `json:"inSchema"`
	OutSchema            types.Type      `json:"outSchema"`
	Transformation       *Transformation `json:"transformation"`
	Format               string          `json:"format"`
	Query                string          `json:"query"`
	Path                 string          `json:"path"`
	Sheet                string          `json:"sheet"`
	Compression          Compression     `json:"compression"`
	FormatSettings       json.RawMessage `json:"formatSettings,omitempty"`
	ExportMode           ExportMode      `json:"exportMode,omitempty"`
	Matching             Matching        `json:"matching"`
	UpdateOnDuplicates   bool            `json:"updateOnDuplicates"`
	TableName            string          `json:"tableName"`
	TableKey             string          `json:"tableKey"`
	IdentityColumn       string          `json:"identityColumn"`
	LastChangeTimeColumn string          `json:"lastChangeTimeColumn"`
	LastChangeTimeFormat string          `json:"lastChangeTimeFormat"`
	OrderBy              string          `json:"orderBy"`
}

type Compression string

const (
	NoCompression     Compression = ""
	ZipCompression    Compression = "Zip"
	GzipCompression   Compression = "Gzip"
	SnappyCompression Compression = "Snappy"
)

type Strategy string

type ConnectionToCreate struct {
	Name              string          `json:"name"`
	Role              Role            `json:"role"`
	Connector         string          `json:"connector"`
	Strategy          *Strategy       `json:"strategy"`
	LinkedConnections []int           `json:"linkedConnections"`
	SendingMode       *SendingMode    `json:"sendingMode"`
	Settings          json.RawMessage `json:"settings"`
}

type DisplayedProperties struct {
	Image       string `json:"image"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Information string `json:"information"`
}

type DummySettings struct {
	URLForDispatchingEvents string
}

type Execution struct {
	ID        int        `json:"id"`
	Action    int        `json:"action"`
	StartTime time.Time  `json:"startTime"`
	EndTime   *time.Time `json:"endTime"`
	Passed    [6]int     `json:"passed"`
	Failed    [6]int     `json:"failed"`
	Error     string     `json:"error"`
}

type ExportMode string

const (
	CreateOnly     ExportMode = "CreateOnly"
	UpdateOnly     ExportMode = "UpdateOnly"
	CreateOrUpdate ExportMode = "CreateOrUpdate"
)

type Filter struct {
	Logical    FilterLogical     `json:"logical"`
	Conditions []FilterCondition `json:"conditions"`
}

type FilterLogical string

const (
	OpAnd FilterLogical = "and"
	OpOr  FilterLogical = "or"
)

type FilterCondition struct {
	Property string         `json:"property"`
	Operator FilterOperator `json:"operator"`
	Values   []string       `json:"values"`
}

type FilterOperator string

const (
	OpIs                     FilterOperator = "is"
	OpIsNot                  FilterOperator = "is not"
	OpIsLessThan             FilterOperator = "is less than"
	OpIsLessThanOrEqualTo    FilterOperator = "is less than or equal to"
	OpIsGreaterThan          FilterOperator = "is greater than"
	OpIsGreaterThanOrEqualTo FilterOperator = "is greater than or equal to"
	OpIsBetween              FilterOperator = "is between"
	OpIsNotBetween           FilterOperator = "is not between"
	OpContains               FilterOperator = "contains"
	OpDoesNotContain         FilterOperator = "does not contain"
	OpIsOneOf                FilterOperator = "is one of"
	OpIsNotOneOf             FilterOperator = "is not one of"
	OpStartsWith             FilterOperator = "starts with"
	OpEndsWith               FilterOperator = "ends with"
	OpIsBefore               FilterOperator = "is before"
	OpIsOnOrBefore           FilterOperator = "is on or before"
	OpIsAfter                FilterOperator = "is after"
	OpIsOnOrAfter            FilterOperator = "is on or after"
	OpIsTrue                 FilterOperator = "is true"
	OpIsFalse                FilterOperator = "is false"
	OpIsNull                 FilterOperator = "is null"
	OpIsNotNull              FilterOperator = "is not null"
	OpExists                 FilterOperator = "exists"
	OpDoesNotExist           FilterOperator = "does not exist"
)

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type User struct {
	ID                uuid.UUID      `json:"id"`
	SourcesLastUpdate time.Time      `json:"sourcesLastUpdate"`
	Traits            map[string]any `json:"traits"`
}

type UserIdentity struct {
	Connection     int       // do not use in tests. Currently, this serves just for the UI.
	Action         int       `json:"action"`
	ID             string    `json:"id"`
	AnonymousIds   []string  `json:"anonymousIds"`
	LastChangeTime time.Time `json:"lastChangeTime"`
}

type LabelValue struct { // copy-pasted from the not-exported type 'labelValue' within package 'apis'.
	Label string `json:"label"`
	Value string `json:"value"`
}

type Language string

type Matching struct {
	In  string `json:"in"`
	Out string `json:"out"`
}

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
	Client          SendingMode = "Client"
	Server          SendingMode = "Server"
	ClientAndServer SendingMode = "ClientAndServer"
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
	TargetEvent Target = iota + 1
	TargetUser
	TargetGroup
)

func (at Target) MarshalJSON() ([]byte, error) {
	return []byte(`"` + at.String() + `"`), nil
}

func (at Target) String() string {
	switch at {
	case TargetEvent:
		return "Event"
	case TargetUser:
		return "User"
	case TargetGroup:
		return "Group"
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
		return fmt.Errorf("json: cannot scan a %T value into an meergotester.Target value", v)
	}
	switch s {
	case "Event":
		*at = TargetEvent
	case "User":
		*at = TargetUser
	case "Group":
		*at = TargetGroup
	default:
		return fmt.Errorf("json: invalid meergotester.Target: %s", s)
	}
	return nil
}

type Transformation struct {
	Mapping  map[string]string       `json:"mapping"`
	Function *TransformationFunction `json:"function"`
}

type TransformationFunction struct {
	Language     Language `json:"language"`
	Source       string   `json:"source"`
	PreserveJSON bool     `json:"preserveJSON"`
	InPaths      []string `json:"inPaths"`
	OutPaths     []string `json:"outPaths"`
}

type WarehouseMode int

const (
	Normal WarehouseMode = iota
	Inspection
	Maintenance
)

// MarshalJSON returns mode as the JSON encoding of mode.
func (mode WarehouseMode) MarshalJSON() ([]byte, error) {
	switch mode {
	case Normal:
		return []byte(`"Normal"`), nil
	case Inspection:
		return []byte(`"Inspection"`), nil
	case Maintenance:
		return []byte(`"Maintenance"`), nil
	}
	return nil, errors.New("invalid warehouse mode")
}

type Workspace struct {
	ID                             int                 `json:"id"`
	Name                           string              `json:"name"`
	UserSchema                     types.Type          `json:"userSchema"`
	UserPrimarySources             map[string]int      `json:"userPrimarySources"`
	ResolveIdentitiesOnBatchImport bool                `json:"ResolveIdentitiesOnBatchImport"`
	Identifiers                    []string            `json:"identifiers"`
	DisplayedProperties            DisplayedProperties `json:"displayedProperties"`
}
