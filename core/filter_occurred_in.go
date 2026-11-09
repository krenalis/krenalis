// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterOccurredInCondition tests the selected occurrence's timestamp.
// It is allowed only inside an experience verify, without operator or values.
type FilterOccurredInCondition struct {
	OccurredIn FilterPeriod `json:"occurredIn"`
}

// MarshalJSON returns the condition without resolving its period against T.
func (condition *FilterOccurredInCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter occurredIn condition cannot be nil")
	}
	type encoded FilterOccurredInCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes a standalone selected-occurrence time condition.
func (condition *FilterOccurredInCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal an occurredIn condition into nil")
	}
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, selectedFilterRule)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterOccurredInCondition)
	if !ok {
		return errors.New("expected an occurredIn condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterOccurredInCondition) filterRule() {}
