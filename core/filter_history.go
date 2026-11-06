// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterHistory refers to a built-in history and immutable meaning revision.
// registered-queryable-events at revision 1 means retained, queryable recorded
// events before T, without an asserted beginning or external completeness.
type FilterHistory struct {
	Definition      string `json:"definition"`
	MeaningRevision string `json:"meaningRevision"`
}

// UnmarshalJSON decodes a history reference without resolving its definition.
func (history *FilterHistory) UnmarshalJSON(data []byte) error {
	if history == nil {
		return errors.New("core: cannot unmarshal a filter history into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("filter history must be an object")
	}
	for name, value := range json.Value(data).Properties() {
		switch name {
		case "definition", "meaningRevision":
			if !value.IsString() || len(value) > 260 {
				return errors.New("invalid history reference field")
			}
		default:
			return errors.New("unknown filter history property")
		}
	}
	type encoded FilterHistory
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	*history = FilterHistory(decoded)
	return nil
}
