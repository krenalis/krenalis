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

// EventDefinitions returns a copy of the workspace's event definitions.
func (workspace *Workspace) EventDefinitions() []EventDefinition {
	return slices.Clone(eventDefinitions)
}

// LookupEventDefinition returns the definition matching typ and eventOrName.
// eventOrName is the event field for track, the name field for page and screen,
// or empty for the other event types. Generic page and screen definitions
// match any name. Named definitions take precedence over generic definitions.
// Matching is exact and case-sensitive.
// It returns the zero EventDefinition if no definition matches.
func (workspace *Workspace) LookupEventDefinition(typ, eventOrName string) EventDefinition {
	var generic EventDefinition
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
			if name == "" {
				generic = definition
			}
		default:
			// Other event types have no event or name.
		}
		if name == eventOrName {
			return definition
		}
	}
	return generic
}

func init() {
	for _, definition := range eventDefinitions {
		if definition.Schemas.Properties.Valid() &&
			definition.Type != "track" && definition.Type != "page" && definition.Type != "screen" {
			panic(fmt.Sprintf("core/internal/state: unexpected properties schema for type %q", definition.Type))
		}
	}
}
