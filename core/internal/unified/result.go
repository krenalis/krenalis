// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/krenalis/krenalis/warehouses"
)

// Read consumes and closes the normalized outcome cursor. Any read, decode,
// cancellation or Close failure discards the whole operation's result.
func Read(ctx context.Context, rows warehouses.Rows, at time.Time, page Page) (result *Result, err error) {

	if rows == nil {
		return nil, fmt.Errorf("core/internal/unified: missing result cursor")
	}
	defer func() {
		closeErr := rows.Close()
		if closeErr != nil && err == nil {
			result, err = nil, closeErr
		}
	}()
	if page.Limit < 1 || page.Limit > 1000 {
		return nil, fmt.Errorf("invalid internal Unified page")
	}

	result = &Result{Profiles: []Profile{}, At: at.UTC()}
	seen, metadata := 0, false
	var previous string
	for rows.Next() {
		if seen > page.Limit {
			return nil, fmt.Errorf("core/internal/unified: result row limit exceeded")
		}
		values := make([]any, 6)
		err := rows.Scan(values...)
		if err != nil {
			return nil, err
		}
		m, mok := values[3].(int)
		f, fok := values[4].(int)
		u, uok := values[5].(int)
		if !mok || !fok || !uok || m < 0 || f < 0 || u < 0 || int64(m) > math.MaxInt64-int64(f) ||
			int64(m)+int64(f) > math.MaxInt64-int64(u) {
			return nil, fmt.Errorf("core/internal/unified: invalid profile totals")
		}
		if seen == 0 {
			result.Match, result.NoMatch, result.Unknown = int64(m), int64(f), int64(u)
		}
		hasProfile, ok := values[2].(bool)
		if !ok || (metadata && hasProfile) {
			return nil, fmt.Errorf("core/internal/unified: invalid metadata marker")
		}
		if !hasProfile {
			if seen != 0 || values[0] != nil || values[1] != nil {
				return nil, fmt.Errorf("core/internal/unified: invalid empty page")
			}
			metadata = true
		} else {
			id, ok := values[0].(string)
			if !ok || len(id) != 36 {
				return nil, fmt.Errorf("core/internal/unified: invalid profile identity")
			}
			_, err := warehouses.ValidateUUID("profile_id", id)
			if err != nil {
				return nil, err
			}
			if id <= previous || len(result.Profiles) >= page.Limit {
				return nil, fmt.Errorf("core/internal/unified: invalid profile page order or size")
			}
			truth := Unknown
			if values[1] != nil {
				value, ok := values[1].(bool)
				if !ok {
					return nil, fmt.Errorf("core/internal/unified: invalid truth value")
				}
				truth = NoMatch
				if value {
					truth = Match
				}
			}
			if page.MatchingOnly && truth != Match {
				return nil, fmt.Errorf("core/internal/unified: unexpected non-matching profile")
			}
			result.Profiles = append(result.Profiles, Profile{ID: id, Truth: truth})
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
		return nil, fmt.Errorf("core/internal/unified: missing profile totals")
	}

	return result, nil
}
