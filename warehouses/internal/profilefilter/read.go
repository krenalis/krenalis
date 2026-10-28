// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"context"
	"fmt"
	"math"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// Read consumes and closes the normalized outcome cursor. Any read, decode,
// cancellation or Close failure discards the whole operation's result. It
// returns warehouses.ErrProfileResultTooLarge when a page exceeds its budget.
func Read(ctx context.Context, rows warehouses.Rows, request warehouses.ProfileOutcomeRequest) (result *warehouses.ProfileOutcomes, err error) {

	if rows == nil {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: missing result cursor")
	}
	defer func() {
		closeErr := rows.Close()
		if closeErr != nil && err == nil {
			result, err = nil, closeErr
		}
	}()
	if request.Limit < 1 || request.Limit > 1000 {
		return nil, fmt.Errorf("invalid internal Unified page")
	}

	result = &warehouses.ProfileOutcomes{Profiles: []warehouses.ProfileOutcome{}}
	seen, metadata := 0, false
	var previous string
	identities := map[string]bool{}
	bytesRead := 0
	for rows.Next() {
		if seen > request.Limit {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: result row limit exceeded")
		}
		values := make([]any, 8+len(request.Columns))
		err := rows.Scan(values...)
		if err != nil {
			return nil, err
		}
		failures, ok := values[6].(int)
		if !ok || failures < 0 {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid source diagnostic")
		}
		if failures != 0 {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: occurrence source violates the prepared Count contract")
		}
		oversized, ok := values[7].(bool)
		if !ok {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid payload diagnostic")
		}
		if oversized {
			return nil, warehouses.ErrProfileResultTooLarge
		}
		m, mok := values[3].(int)
		f, fok := values[4].(int)
		u, uok := values[5].(int)
		if !mok || !fok || !uok || m < 0 || f < 0 || u < 0 || int64(m) > math.MaxInt64-int64(f) ||
			int64(m)+int64(f) > math.MaxInt64-int64(u) {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid profile totals")
		}
		if seen == 0 {
			result.Match, result.NoMatch, result.Unknown = int64(m), int64(f), int64(u)
		}
		hasProfile, ok := values[2].(bool)
		if !ok || (metadata && hasProfile) {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid metadata marker")
		}
		if !hasProfile {
			if seen != 0 || values[0] != nil || values[1] != nil {
				return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid empty page")
			}
			metadata = true
		} else {
			id, ok := values[0].(string)
			if !ok || len(id) != 36 {
				return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid profile identity")
			}
			_, err := warehouses.ValidateUUID("profile_id", id)
			if err != nil {
				return nil, err
			}
			if (len(request.OrderBy) == 0 && id <= previous) || identities[id] ||
				len(result.Profiles) >= request.Limit {
				return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid profile page order or size")
			}
			truth := warehouses.FilterUnknown
			if values[1] != nil {
				value, ok := values[1].(bool)
				if !ok {
					return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid truth value")
				}
				truth = warehouses.FilterNoMatch
				if value {
					truth = warehouses.FilterMatch
				}
			}
			if request.MatchingOnly && truth != warehouses.FilterMatch {
				return nil, fmt.Errorf("warehouses/internal/profilefilter: unexpected non-matching profile")
			}
			for i, value := range values[8:] {
				if value == nil && !request.Columns[i].Nullable {
					return nil, fmt.Errorf("warehouses/internal/profilefilter: missing required profile value")
				}
				switch value := value.(type) {
				case string:
					bytesRead += len(value)
				case json.Value:
					bytesRead += len(value)
				default:
					// Other normalized values are bounded by their scalar/schema type.
				}
				if bytesRead > warehouses.MaxProfilePageBytes {
					return nil, warehouses.ErrProfileResultTooLarge
				}
			}
			result.Profiles = append(result.Profiles, warehouses.ProfileOutcome{ID: id, Truth: truth,
				Values: values[8:]})
			identities[id] = true
			previous = id
		}
		seen++
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if seen == 0 {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: missing profile totals")
	}
	if request.MatchingOnly && int64(len(result.Profiles)) > result.Match {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: matching page exceeds its total")
	}

	return result, nil
}
