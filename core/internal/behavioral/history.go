// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import "fmt"

// History is a resolved descriptor of the queryable recorded-event domain.
// It neither certifies external completeness nor reconstructs deleted events.
type History struct {
	definition      string
	meaningRevision string
}

// ResolveHistory resolves registered-queryable-events at meaning revision 1.
// It covers retained, queryable recorded events before T without a lower bound.
func ResolveHistory(definition, meaningRevision string) (*History, error) {
	if definition != "registered-queryable-events" {
		return nil, fmt.Errorf("invalid recorded history definition")
	}
	if meaningRevision != "1" {
		return nil, fmt.Errorf("unresolved recorded history meaning revision")
	}
	return &History{definition: definition, meaningRevision: meaningRevision}, nil
}

// Reference returns the resolved definition and immutable meaning revision.
func (history *History) Reference() (definition, meaningRevision string) {
	if history == nil || history.definition == "" {
		panic("core/internal/behavioral: unresolved recorded history")
	}
	return history.definition, history.meaningRevision
}
