//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package chichitester

import (
	"encoding/json"
	"time"

	"chichi/connector/types"
)

// These data types are copy-paste of the types defined within the APIs.

type ActionToSet struct {
	Name                    string
	Enabled                 bool
	Filter                  *Filter
	InSchema                types.Type
	OutSchema               types.Type
	Transformation          Transformation
	Query                   string
	Path                    string
	TableName               string
	Sheet                   string
	IdentityColumn          string
	TimestampColumn         string
	TimestampFormat         string
	ExportMode              *ExportMode
	MatchingProperties      *MatchingProperties
	ExportOnDuplicatedUsers *bool
}

type BusinessID struct {
	Name  string
	Label string
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
	Name        string
	Role        Role
	Enabled     bool
	Connector   int
	Storage     int
	Compression Compression
	Strategy    *Strategy
	WebsiteHost string
	BusinessID  BusinessID
	Settings    json.RawMessage
}

type DisplayedProperties struct {
	Image       string
	FirstName   string
	LastName    string
	Information string
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
	Value    string
}

type UserIdentity struct { // copy-pasted from the not-exported type 'identity' within package 'apis'.
	Connection   int
	ExternalId   LabelValue
	BusinessId   LabelValue
	AnonymousIds []string
	UpdatedAt    time.Time
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

type Transformation struct {
	Mapping  map[string]string
	Function *TransformationFunction
}

type TransformationFunction struct {
	Source   string
	Language Language
}

type Workspace struct {
	ID                  int
	Name                string
	UsersSchema         types.Type
	Identifiers         []string
	PrivacyRegion       PrivacyRegion
	DisplayedProperties DisplayedProperties
}
