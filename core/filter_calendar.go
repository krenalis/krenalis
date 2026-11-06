// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterCalendar refers to a built-in calendar and immutable meaning revision.
// Definitions are gregorian-monday/TIMEZONE, with an explicit IANA timezone.
type FilterCalendar struct {
	Definition      string `json:"definition"`
	MeaningRevision string `json:"meaningRevision"`
}

// UnmarshalJSON decodes a calendar reference without resolving its definition.
func (calendar *FilterCalendar) UnmarshalJSON(data []byte) error {
	if calendar == nil {
		return errors.New("core: cannot unmarshal a filter calendar into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("filter calendar must be an object")
	}
	for name, value := range json.Value(data).Properties() {
		switch name {
		case "definition", "meaningRevision":
			if !value.IsString() || len(value) > 260 {
				return errors.New("invalid calendar reference field")
			}
		default:
			return errors.New("unknown filter calendar property")
		}
	}
	type encoded FilterCalendar
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	*calendar = FilterCalendar(decoded)
	return nil
}
