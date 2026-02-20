// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package meergotester

import (
	"bytes"
	"fmt"
	"time"

	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

	"github.com/google/uuid"
)

// These data types are copy-paste of the types defined within the APIs.

type PipelineToSet struct {
	Name               string          `json:"name"`
	Enabled            bool            `json:"enabled"`
	Filter             *Filter         `json:"filter"`
	InSchema           types.Type      `json:"inSchema"`
	OutSchema          types.Type      `json:"outSchema"`
	Transformation     *Transformation `json:"transformation"`
	Query              string          `json:"query"`
	Format             string          `json:"format"`
	Path               string          `json:"path"`
	Sheet              string          `json:"sheet"`
	Compression        Compression     `json:"compression"`
	OrderBy            string          `json:"orderBy"`
	FormatSettings     json.Value      `json:"formatSettings,omitempty"`
	ExportMode         ExportMode      `json:"exportMode,omitempty"`
	Matching           Matching        `json:"matching"`
	UpdateOnDuplicates bool            `json:"updateOnDuplicates"`
	TableName          string          `json:"tableName"`
	TableKey           string          `json:"tableKey"`
	UserIDColumn       string          `json:"userIDColumn"`
	UpdatedAtColumn    string          `json:"updatedAtColumn"`
	UpdatedAtFormat    string          `json:"updatedAtFormat"`
	Incremental        bool            `json:"incremental"`
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
	Name              string       `json:"name"`
	Role              Role         `json:"role"`
	Connector         string       `json:"connector"`
	Strategy          *Strategy    `json:"strategy"`
	LinkedConnections []int        `json:"linkedConnections"`
	SendingMode       *SendingMode `json:"sendingMode"`
	Settings          json.Value   `json:"settings"`
}

type DummySettings struct {
	URLForDispatchingEvents string `json:"urlForDispatchingEvents"`
}

type PipelineRun struct {
	ID        int        `json:"id"`
	Pipeline  int        `json:"pipeline"`
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
	OpIsEmpty                FilterOperator = "is empty"
	OpIsNotEmpty             FilterOperator = "is not empty"
	OpIsNull                 FilterOperator = "is null"
	OpIsNotNull              FilterOperator = "is not null"
	OpExists                 FilterOperator = "exists"
	OpDoesNotExist           FilterOperator = "does not exist"
)

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Profile struct {
	MPID       uuid.UUID      `json:"mpid"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	Attributes map[string]any `json:"attributes"`
}

type Identity struct {
	UserID       string    `json:"userId"`
	AnonymousIDs []string  `json:"anonymousIds"`
	UpdatedAt    time.Time `json:"updatedAt"`
	Connection   int       `json:"connection"`
	Pipeline     int       `json:"pipeline"`
}

type LabelValue struct { // copy-pasted from the not-exported type 'labelValue' within package 'core'.
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
		return fmt.Errorf("json: cannot scan a %T value into a SchedulePeriod value", v)
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
		return fmt.Errorf("json: invalid SchedulePeriod: %s", s)
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

type WarehouseMode string

const (
	Normal      WarehouseMode = "Normal"
	Inspection  WarehouseMode = "Inspection"
	Maintenance WarehouseMode = "Maintenance"
)

type Workspace struct {
	ID                             int            `json:"id"`
	Name                           string         `json:"name"`
	ProfileSchema                  types.Type     `json:"profileSchema"`
	PrimarySources                 map[string]int `json:"primarySources"`
	ResolveIdentitiesOnBatchImport bool           `json:"resolveIdentitiesOnBatchImport"`
	Identifiers                    []string       `json:"identifiers"`
	WarehouseMode                  WarehouseMode  `json:"warehouseMode"`
	UIPreferences                  UIPreferences  `json:"uiPreferences"`
}

type UIPreferences struct {
	Profile struct {
		Image     string `json:"image"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Extra     string `json:"extra"`
	} `json:"profile"`
}
