// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// ProfileFilterExplanation describes one complete evaluation for a profile.
// At anchors periods, without selecting a historical snapshot of the data.
type ProfileFilterExplanation struct {
	KPID string                 `json:"kpid"`
	At   time.Time              `json:"at"`
	Rule ProfileRuleExplanation `json:"rule"`
}

// ProfileRuleExplanation follows the input filter's group and rule order.
// Kind is and, or, record or count; Result is MATCH, NO_MATCH or UNKNOWN.
// Record results do not identify which operand caused a SQL NULL.
type ProfileRuleExplanation struct {
	Kind   string                   `json:"kind"`
	Result string                   `json:"result"`
	Rules  []ProfileRuleExplanation `json:"rules,omitzero"`
	Count  *ProfileCountExplanation `json:"count,omitzero"`
}

// ProfileCountExplanation describes independently qualified occurrences in
// [Start,End). Every integer from Certain to Certain+Uncertain is reachable.
// These are recorded occurrences, not deduplicated business entities.
type ProfileCountExplanation struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Certain   int64     `json:"certain"`
	Uncertain int64     `json:"uncertain"`
}

// ExplainProfileFilter explains a supported filter for the KPID in the URL.
// It returns errors.BadRequestError for invalid arguments,
// errors.NotFoundError for a missing profile, errors.UnavailableError for a
// warehouse failure, and errors.UnprocessableError with the following codes:
//   - PropertyNotExist for an unresolved property path.
//   - FilterCapabilityNotImplemented for an unavailable language capability.
//   - UnifiedFilteringUnavailable for an unverified workspace backend.
//   - MaintenanceMode when warehouse access is in maintenance.
func (this *Workspace) ExplainProfileFilter(ctx context.Context, kpid string, filter *Filter, at string) (*ProfileFilterExplanation, error) {

	this.core.mustBeOpen()

	profile, ok := types.NormalizeUUID(kpid)
	if !ok {
		return nil, errors.BadRequest("invalid KPID")
	}
	if filter == nil {
		return nil, errors.BadRequest("filter is required")
	}
	now := time.Now().UTC()
	instant := now.Truncate(time.Microsecond)
	if at != "" {
		parsed, err := parseFilterInstant(at)
		if err != nil {
			return nil, errors.BadRequest("invalid at: %s", err)
		}
		instant = parsed.UTC()
		if instant.IsZero() || instant.Year() < 1 || instant.Year() > 9999 ||
			instant.Nanosecond()%1000 != 0 || instant.After(now) {
			return nil, errors.BadRequest("at must be a nonfuture timestamp with microsecond precision")
		}
	}
	nodes := 0
	root, err := prepareUnifiedRule(filter, this.workspace.ProfileSchema, 0, &nodes)
	if err != nil {
		if missing, ok := errors.AsType[types.PathNotExistError](err); ok {
			return nil, errors.Unprocessable(PropertyNotExist, "filter's property %s does not exist", missing.Path)
		}
		if unavailable, ok := errors.AsType[*unified.UnsupportedError](err); ok {
			return nil, errors.Unprocessable(FilterCapabilityNotImplemented, "%s", unavailable)
		}
		return nil, errors.BadRequest("filter is not valid: %s", err)
	}

	result, err := this.store.ExplainUnifiedProfile(ctx, &root, unified.Bindings{}, instant, profile)
	if err != nil {
		if errors.Is(err, datastore.ErrUnifiedUnavailable) {
			return nil, errors.Unprocessable(UnifiedFilteringUnavailable, "Unified filtering is unavailable on this warehouse")
		}
		if unavailable, ok := errors.AsType[*unified.UnsupportedError](err); ok {
			return nil, errors.Unprocessable(FilterCapabilityNotImplemented, "%s", unavailable)
		}
		if errors.Is(err, datastore.ErrMaintenanceMode) {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if _, ok := errors.AsType[*datastore.UnavailableError](err); ok {
			return nil, errors.Unavailable("data warehouse profile explanation failed")
		}
		return nil, errors.New("core: profile explanation failed")
	}
	if result == nil {
		return nil, errors.NotFound("profile does not exist")
	}
	index := 0
	explanation, err := explainProfileRule(result.Filter.Root, result.Nodes, &index)
	if err != nil {
		return nil, err
	}
	if index != len(result.Nodes) {
		return nil, errors.New("core: unexpected profile explanation nodes")
	}

	return &ProfileFilterExplanation{KPID: profile, At: result.At, Rule: explanation}, nil
}

// explainProfileRule attaches measured outcomes to the prepared filter shape.
// It neither reevaluates conditions nor infers reasons from raw SQL NULLs.
func explainProfileRule(rule warehouses.FilterCondition, nodes []warehouses.FilterNodeOutcome, index *int) (ProfileRuleExplanation, error) {

	if *index >= len(nodes) {
		return ProfileRuleExplanation{}, errors.New("core: incomplete profile explanation")
	}
	node := nodes[*index]
	*index++
	result := ProfileRuleExplanation{}
	switch node.Truth {
	case warehouses.FilterMatch:
		result.Result = "MATCH"
	case warehouses.FilterNoMatch:
		result.Result = "NO_MATCH"
	case warehouses.FilterUnknown:
		result.Result = "UNKNOWN"
	default:
		return ProfileRuleExplanation{}, errors.New("core: invalid profile explanation truth")
	}

	switch rule := rule.(type) {
	case *warehouses.FilterGroup:
		result.Kind = "and"
		if rule.Operator == warehouses.OpOr {
			result.Kind = "or"
		}
		result.Rules = make([]ProfileRuleExplanation, len(rule.Children))
		for i, child := range rule.Children {
			var err error
			result.Rules[i], err = explainProfileRule(child, nodes, index)
			if err != nil {
				return ProfileRuleExplanation{}, err
			}
		}
	case *warehouses.RecordFilter:
		result.Kind = "record"
	case *warehouses.IndependentOccurrenceCount:
		if node.Count == nil {
			return ProfileRuleExplanation{}, errors.New("core: missing profile explanation count")
		}
		result.Kind = "count"
		result.Count = &ProfileCountExplanation{Start: rule.Start, End: rule.End, Certain: node.Count.Certain,
			Uncertain: node.Count.Possible - node.Count.Certain}
	default:
		return ProfileRuleExplanation{}, errors.New("core: invalid profile explanation condition")
	}

	return result, nil
}
