// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"unicode/utf8"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// EventDefinition describes a catalog entry, not an observed event or a
// guarantee about the schemas of historical warehouse data.
type EventDefinition struct {
	ID          string       `json:"id"`
	Type        string       `json:"type"`
	Event       string       `json:"event,omitzero"`
	Name        string       `json:"name,omitzero"`
	DisplayName string       `json:"displayName"`
	Description string       `json:"description"`
	Trigger     string       `json:"trigger,omitzero"`
	Schemas     EventSchemas `json:"schemas"`
}

// EventSchemas contains native catalog schemas. An unspecified schema is null
// when serialized; it does not mean that the corresponding data is absent.
type EventSchemas struct {
	Traits     types.Type `json:"traits"`
	Properties types.Type `json:"properties"`
}

// EventDefinition returns the catalog entry with the exact ID. It returns an
// errors.BadRequestError for invalid IDs and an errors.UnprocessableError
// with code EventDefinitionNotExist for unknown IDs.
func (this *Workspace) EventDefinition(id string) (EventDefinition, error) {
	this.core.mustBeOpen()

	if id == "" || len(id) > 128 || !utf8.ValidString(id) {
		return EventDefinition{}, errors.BadRequest("event definition ID must contain 1 to 128 UTF-8 bytes")
	}
	for _, definition := range this.workspace.EventDefinitions() {
		if definition.ID == id {
			return EventDefinition{
				ID: definition.ID, Type: definition.Type, Event: definition.Event, Name: definition.Name,
				DisplayName: definition.DisplayName, Description: definition.Description, Trigger: definition.Trigger,
				Schemas: EventSchemas{Traits: definition.Schemas.Traits, Properties: definition.Schemas.Properties},
			}, nil
		}
	}
	return EventDefinition{}, errors.Unprocessable(EventDefinitionNotExist, "event definition does not exist")
}

// EventDefinitions returns catalog entries in catalog order. Query is an
// optional case-insensitive substring of an ID, technical name, display name
// or description; it is limited to 256 UTF-8 bytes. An empty query lists all.
// It returns errors.BadRequestError for invalid queries.
func (this *Workspace) EventDefinitions(query string) ([]EventDefinition, error) {

	this.core.mustBeOpen()

	if len(query) > 256 || !utf8.ValidString(query) {
		return nil, errors.BadRequest("catalog query must contain at most 256 UTF-8 bytes")
	}
	query = strings.ToLower(strings.TrimSpace(query))
	definitions := []EventDefinition{}
	for _, definition := range this.workspace.EventDefinitions() {
		if query != "" && !strings.Contains(strings.ToLower(definition.ID), query) &&
			!strings.Contains(strings.ToLower(definition.Event), query) &&
			!strings.Contains(strings.ToLower(definition.Name), query) &&
			!strings.Contains(strings.ToLower(definition.DisplayName), query) &&
			!strings.Contains(strings.ToLower(definition.Description), query) {
			continue
		}
		definitions = append(definitions, EventDefinition{
			ID: definition.ID, Type: definition.Type, Event: definition.Event, Name: definition.Name,
			DisplayName: definition.DisplayName, Description: definition.Description, Trigger: definition.Trigger,
			Schemas: EventSchemas{Traits: definition.Schemas.Traits, Properties: definition.Schemas.Properties},
		})
	}

	return definitions, nil
}
