// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"fmt"
	"slices"

	"github.com/krenalis/krenalis/tools/types"
)

// EventDefinition describes an event in the workspace's catalog.
// Catalog entries define properties schemas only for track, page and screen
// events.
type EventDefinition struct {
	ID          string
	Type        string
	Event       string // event field of a track event
	Name        string // name field of a page or screen event
	DisplayName string
	Description string
	Trigger     string
	Schemas     EventSchemas
}

// EventSchemas holds the optional schemas of an event's traits and properties.
// Each schema is either an object type or an invalid type when unspecified.
type EventSchemas struct {
	Traits     types.Type
	Properties types.Type
}

// eventDefinitions is shared by all workspaces and is never modified.
// These examples will be replaced by the production catalog.
var eventDefinitions = []EventDefinition{
	{
		ID:          "example-action",
		Type:        "track",
		Event:       "Example Action",
		DisplayName: "Example action",
		Description: "Demonstration event covering Krenalis property types. It does not represent a business action.",
		Trigger:     "Emitted by a test integration to exercise the event catalog.",
		Schemas: EventSchemas{
			Traits: types.Object([]types.Property{
				{
					Name: "email", Type: types.String().AsEmail(), ReadOptional: true,
					DisplayName: "Email", Description: "Contact email address supplied by the test integration.",
				},
				{
					Name: "country", Type: types.String().AsCountry(types.ISO3166Alpha2), ReadOptional: true,
					DisplayName: "Country", Description: "Country represented by its two-letter ISO code.",
				},
			}),
			Properties: types.Object([]types.Property{
				{
					Name: "label", Type: types.String().WithMaxLength(100), ReadOptional: true,
					DisplayName: "Label", Description: "Descriptive text of at most 100 characters.",
				},
				{
					Name: "enabled", Type: types.Boolean(), ReadOptional: true,
					DisplayName: "Enabled", Description: "Whether the sample feature is enabled.",
				},
				{
					Name: "quantity", Type: types.Int(32), ReadOptional: true,
					DisplayName: "Quantity", Description: "Sample signed integer quantity.",
				},
				{
					Name: "sequence", Type: types.Int(64).Unsigned(), ReadOptional: true,
					DisplayName: "Sequence", Description: "Sample unsigned 64-bit sequence number.",
				},
				{
					Name: "score", Type: types.Float(64).Real(), ReadOptional: true,
					DisplayName: "Score", Description: "Finite floating-point sample score.",
				},
				{
					Name: "amount", Type: types.Decimal(18, 4).AsMoney().WithCurrency("EUR"), ReadOptional: true,
					DisplayName: "Amount", Description: "Sample monetary amount in euros, not cents.",
				},
				{
					Name: "occurredAt", Type: types.DateTime(), ReadOptional: true,
					DisplayName: "Occurrence time", Description: "Sample instant, independent of the event timestamp.",
				},
				{
					Name: "date", Type: types.Date(), ReadOptional: true,
					DisplayName: "Date", Description: "Sample calendar date without a time of day.",
				},
				{
					Name: "time", Type: types.Time(), ReadOptional: true,
					DisplayName: "Time", Description: "Sample time of day without a calendar date.",
				},
				{
					Name: "year", Type: types.Year(), ReadOptional: true,
					DisplayName: "Year", Description: "Sample calendar year.",
				},
				{
					Name: "referenceId", Type: types.UUID(), ReadOptional: true,
					DisplayName: "Reference ID", Description: "Sample reference represented as a UUID.",
				},
				{
					Name: "address", Type: types.IP(), ReadOptional: true,
					DisplayName: "IP address", Description: "Sample IPv4 or IPv6 address.",
				},
				{
					Name: "payload", Type: types.JSON(), ReadOptional: true,
					DisplayName: "Payload", Description: "Additional JSON data without a declared internal structure.",
				},
				{
					Name: "tags", Type: types.Array(types.String()), ReadOptional: true,
					DisplayName: "Tags", Description: "Ordered collection of sample string tags.",
				},
				{
					Name: "metadata", Type: types.Map(types.String()), ReadOptional: true,
					DisplayName: "Metadata", Description: "Sample string values indexed by arbitrary keys.",
				},
				{
					Name: "details",
					Type: types.Object([]types.Property{
						{
							Name: "source", Type: types.String(), ReadOptional: true,
							DisplayName: "Source", Description: "Name of the sample integration.",
						},
					}),
					ReadOptional: true,
					DisplayName:  "Details",
					Description:  "Sample nested object with a declared structure.",
				},
			}),
		},
	},
	{
		ID:          "example-page",
		Type:        "page",
		Name:        "Example Page",
		DisplayName: "Example page",
		Description: "Demonstration page event without specialized traits or properties schemas.",
		Trigger:     "Emitted by a test integration when displaying the example page.",
	},
}

// EventDefinitions returns a copy of the workspace's event definitions.
func (workspace *Workspace) EventDefinitions() []EventDefinition {
	return slices.Clone(eventDefinitions)
}

// LookupEventDefinition returns the definition matching typ and eventOrName.
// eventOrName is the event field for track, the name field for page and screen,
// or empty for the other event types. Matching is exact and case-sensitive.
// It returns the zero EventDefinition if no definition matches.
func (workspace *Workspace) LookupEventDefinition(typ, eventOrName string) EventDefinition {
	for _, definition := range eventDefinitions {
		if definition.Type != typ {
			continue
		}
		var name string
		switch typ {
		case "track":
			name = definition.Event
		case "page", "screen":
			name = definition.Name
		default:
			// Other event types have no event or name.
		}
		if name == eventOrName {
			return definition
		}
	}
	return EventDefinition{}
}

func init() {
	for _, definition := range eventDefinitions {
		if definition.Schemas.Properties.Valid() &&
			definition.Type != "track" && definition.Type != "page" && definition.Type != "screen" {
			panic(fmt.Sprintf("core/internal/state: unexpected properties schema for type %q", definition.Type))
		}
	}
}
