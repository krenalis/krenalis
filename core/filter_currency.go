// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterCurrencyField identifies a governed currency reading and revision.
type FilterCurrencyField struct {
	Definition      string `json:"definition"`
	MeaningRevision string `json:"meaningRevision"`
}

// UnmarshalJSON decodes a currency field reference without resolving it.
func (field *FilterCurrencyField) UnmarshalJSON(data []byte) error {
	if field == nil {
		return errors.New("core: cannot unmarshal a currency field into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("currencyField must be an object")
	}
	for name, value := range json.Value(data).Properties() {
		switch name {
		case "definition", "meaningRevision":
			if !value.IsString() || len(value) > 260 {
				return errors.New("invalid currency field reference")
			}
		default:
			return errors.New("unknown currency field reference property")
		}
	}
	type encoded FilterCurrencyField
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	*field = FilterCurrencyField(decoded)
	return nil
}
