// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"slices"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
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
// Kind is and, or, record, count, sum, average or experience.
// Result is MATCH, NO_MATCH or UNKNOWN.
// Record results do not identify which operand caused a SQL NULL.
type ProfileRuleExplanation struct {
	Kind       string                        `json:"kind"`
	Result     string                        `json:"result"`
	Rules      []ProfileRuleExplanation      `json:"rules,omitzero"`
	Count      *ProfileCountExplanation      `json:"count,omitzero"`
	Sum        *ProfileSumExplanation        `json:"sum,omitzero"`
	Average    *ProfileAverageExplanation    `json:"average,omitzero"`
	Experience *ProfileExperienceExplanation `json:"experience,omitzero"`
}

// ProfileExperienceExplanation describes selection before verification.
// Period metadata has the same meaning and omission rules as Count.
type ProfileExperienceExplanation struct {
	Start          time.Time                      `json:"start,omitzero"`
	End            time.Time                      `json:"end"`
	UnboundedStart bool                           `json:"unboundedStart,omitzero"`
	History        *FilterHistory                 `json:"history,omitzero"`
	Select         FilterExperienceSelector       `json:"select"`
	Selection      ProfileExperienceSelection     `json:"selection"`
	OccurredIn     []ProfileOccurredInExplanation `json:"occurredIn,omitzero"`
	Money          []ProfileMoneyExplanation      `json:"money,omitzero"`
}

// ProfileExperienceSelection retains ambiguity without inventing a witness.
// Kind is absent, unique or ambiguous. Absence is certain, possible or
// impossible. Candidates counts distinct possible selected-message identities.
type ProfileExperienceSelection struct {
	Kind       string `json:"kind"`
	Absence    string `json:"absence"`
	Candidates int64  `json:"candidates"`
}

// ProfileOccurredInExplanation reports a resolved half-open verify window.
// Path contains zero-based rule indexes relative to the experience verify root.
type ProfileOccurredInExplanation struct {
	Path  []int     `json:"path"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ProfileMoneyExplanation retains the requested and effective comparisons.
// Path contains zero-based rule indexes relative to the experience verify root.
// No candidate amount or identity is exposed.
type ProfileMoneyExplanation struct {
	Path           []int                  `json:"path"`
	Field          FilterSumField         `json:"field"`
	NativeCurrency string                 `json:"nativeCurrency"`
	Requested      ProfileMoneyComparison `json:"requested"`
	Normalized     ProfileMoneyComparison `json:"normalized"`
}

// ProfileMoneyComparison is an exact monetary operator and its JSON operands.
type ProfileMoneyComparison struct {
	Operator FilterOperator    `json:"operator"`
	Values   []decimal.Decimal `json:"values"`
}

// ProfileAverageExplanation reports exact inputs without a rounded quotient.
// Count is null for uncertain membership; Sum is null unless membership and
// amounts are determined. Certain/Uncertain retain the sample count bounds.
// Zero inputs on an empty population do not define a zero average.
type ProfileAverageExplanation struct {
	Start          time.Time        `json:"start,omitzero"`
	End            time.Time        `json:"end"`
	UnboundedStart bool             `json:"unboundedStart,omitzero"`
	History        *FilterHistory   `json:"history,omitzero"`
	Field          FilterSumField   `json:"field"`
	NativeCurrency string           `json:"nativeCurrency"`
	Sum            *decimal.Decimal `json:"sum"`
	Count          *int64           `json:"count"`
	Certain        int64            `json:"certain"`
	Uncertain      int64            `json:"uncertain"`
}

// ProfileCountExplanation describes independently qualified occurrences in
// [Start,End). Every integer from Certain to Certain+Uncertain is reachable.
// These are recorded occurrences, not deduplicated business entities.
// UnboundedStart means all queryable history before End; Start is then omitted.
// History identifies the resolved definition and meaning revision when present.
// CurrencyField and NativeCurrency identify the governed reading when used.
type ProfileCountExplanation struct {
	Start          time.Time            `json:"start,omitzero"`
	End            time.Time            `json:"end"`
	Certain        int64                `json:"certain"`
	Uncertain      int64                `json:"uncertain"`
	UnboundedStart bool                 `json:"unboundedStart,omitzero"`
	History        *FilterHistory       `json:"history,omitzero"`
	CurrencyField  *FilterCurrencyField `json:"currencyField,omitzero"`
	NativeCurrency string               `json:"nativeCurrency,omitzero"`
}

// ProfileSumExplanation describes monetary bounds with the resolved reading.
// Finite bounds are tight under independent occurrence qualifications.
// Certain counts certainly included occurrences; Uncertain counts only
// additional occurrences with uncertain inclusion. Unknown amounts still count.
type ProfileSumExplanation struct {
	Start          time.Time        `json:"start,omitzero"`
	End            time.Time        `json:"end"`
	UnboundedStart bool             `json:"unboundedStart,omitzero"`
	History        *FilterHistory   `json:"history,omitzero"`
	Field          FilterSumField   `json:"field"`
	NativeCurrency string           `json:"nativeCurrency"`
	Bounds         ProfileSumBounds `json:"bounds"`
	Certain        int64            `json:"certain"`
	Uncertain      int64            `json:"uncertain"`
}

// ProfileSumBounds contains tight extrema for independent qualification
// outcomes. Record groups keep Kleene logic without solving constraints
// between predicates on missing raw fields. Null means unbounded.
type ProfileSumBounds struct {
	Lower *decimal.Decimal `json:"lower"`
	Upper *decimal.Decimal `json:"upper"`
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
	explanation, err := explainProfileRule(result.Filter.Root, root, result.Nodes, &index, result.At)
	if err != nil {
		return nil, err
	}
	if index != len(result.Nodes) {
		return nil, errors.New("core: unexpected profile explanation nodes")
	}

	return &ProfileFilterExplanation{KPID: profile, At: result.At, Rule: explanation}, nil
}

// explainMoneyPredicates retains each monetary leaf and its normalized test.
func explainMoneyPredicates(verify *unified.ExperienceVerification, path []int) ([]ProfileMoneyExplanation, error) {
	if verify == nil {
		return nil, nil
	}
	if verify.Money != nil {
		money := verify.Money
		normalized, err := money.NormalizedTest()
		if err != nil {
			return nil, errors.New("core: unresolved money explanation comparison")
		}
		definition, revision := money.Field.Reference()
		operators := map[unified.Comparison]FilterOperator{
			unified.Greater: OpIsGreaterThan, unified.GreaterOrEqual: OpIsGreaterThanOrEqualTo,
			unified.Less: OpIsLessThan, unified.LessOrEqual: OpIsLessThanOrEqualTo,
		}
		return []ProfileMoneyExplanation{{Path: slices.Clone(path),
			Field:          FilterSumField{Definition: definition, MeaningRevision: revision},
			NativeCurrency: money.NativeCurrency,
			Requested: ProfileMoneyComparison{Operator: operators[money.Test.Operator],
				Values: []decimal.Decimal{money.Test.Value}},
			Normalized: ProfileMoneyComparison{Operator: operators[normalized.Operator],
				Values: []decimal.Decimal{normalized.Value}},
		}}, nil
	}
	var result []ProfileMoneyExplanation
	for i := range verify.Children {
		leaves, err := explainMoneyPredicates(&verify.Children[i], append(slices.Clone(path), i))
		if err != nil {
			return nil, err
		}
		result = append(result, leaves...)
	}
	return result, nil
}

// explainOccurrencePeriods attaches resolved windows to their input positions.
func explainOccurrencePeriods(verify *unified.ExperienceVerification, at time.Time, path []int) ([]ProfileOccurredInExplanation, error) {
	if verify == nil {
		return nil, nil
	}
	if verify.OccurredIn != nil {
		start, end, err := verify.OccurredIn.Bounds(at)
		if err != nil {
			return nil, errors.New("core: unresolved occurredIn explanation period")
		}
		return []ProfileOccurredInExplanation{{Path: slices.Clone(path), Start: start, End: end}}, nil
	}
	var result []ProfileOccurredInExplanation
	for i := range verify.Children {
		periods, err := explainOccurrencePeriods(&verify.Children[i], at, append(slices.Clone(path), i))
		if err != nil {
			return nil, err
		}
		result = append(result, periods...)
	}
	return result, nil
}

// explainProfileRule attaches measured outcomes and resolved references to the
// unchanged prepared filter shape, without reevaluating conditions.
func explainProfileRule(rule warehouses.FilterCondition, semantic unified.Rule, nodes []warehouses.FilterNodeOutcome, index *int, at time.Time) (ProfileRuleExplanation, error) {

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
		if len(rule.Children) != len(semantic.Children) {
			return ProfileRuleExplanation{}, errors.New("core: inconsistent profile explanation group")
		}
		result.Kind = "and"
		if rule.Operator == warehouses.OpOr {
			result.Kind = "or"
		}
		result.Rules = make([]ProfileRuleExplanation, len(rule.Children))
		for i, child := range rule.Children {
			var err error
			result.Rules[i], err = explainProfileRule(child, semantic.Children[i], nodes, index, at)
			if err != nil {
				return ProfileRuleExplanation{}, err
			}
		}
	case *warehouses.RecordFilter:
		result.Kind = "record"
	case *warehouses.SelectedOccurrence:
		if node.Experience == nil || semantic.Experience == nil {
			return ProfileRuleExplanation{}, errors.New("core: missing profile explanation experience")
		}
		selection := ProfileExperienceSelection{Kind: "ambiguous", Absence: "impossible",
			Candidates: node.Experience.Candidates}
		if node.Experience.AbsencePossible {
			selection.Absence = "possible"
		}
		if selection.Candidates == 0 {
			selection.Kind, selection.Absence = "absent", "certain"
		} else if selection.Candidates == 1 && !node.Experience.AbsencePossible {
			selection.Kind = "unique"
		}
		result.Kind = "experience"
		result.Experience = &ProfileExperienceExplanation{Start: rule.Start, End: rule.End,
			UnboundedStart: rule.UnboundedStart, Select: FilterExperienceSelector{Kind: "last"}, Selection: selection}
		if rule.First {
			result.Experience.Select.Kind = "first"
		}
		periods, err := explainOccurrencePeriods(semantic.Experience.Verify, at, nil)
		if err != nil {
			return ProfileRuleExplanation{}, err
		}
		result.Experience.OccurredIn = periods
		money, err := explainMoneyPredicates(semantic.Experience.Verify, nil)
		if err != nil {
			return ProfileRuleExplanation{}, err
		}
		result.Experience.Money = money
		if history := semantic.Experience.Scope.History; history != nil {
			definition, revision := history.Reference()
			result.Experience.History = &FilterHistory{Definition: definition, MeaningRevision: revision}
		}
	case *warehouses.DeterminedOccurrenceAverage:
		if node.Average == nil || semantic.Average == nil {
			return ProfileRuleExplanation{}, errors.New("core: missing profile explanation average")
		}
		definition, revision := semantic.Average.Field.Reference()
		result.Kind = "average"
		result.Average = &ProfileAverageExplanation{Start: rule.Start, End: rule.End,
			UnboundedStart: rule.UnboundedStart,
			Field:          FilterSumField{Definition: definition, MeaningRevision: revision},
			NativeCurrency: semantic.Average.NativeCurrency, Sum: node.Average.Sum, Count: node.Average.Count,
			Certain: node.Average.Certain, Uncertain: node.Average.Possible - node.Average.Certain}
		if history := semantic.Average.Scope.History; history != nil {
			definition, revision := history.Reference()
			result.Average.History = &FilterHistory{Definition: definition, MeaningRevision: revision}
		}
	case *warehouses.IndependentOccurrenceSum:
		if node.Sum == nil || semantic.Sum == nil {
			return ProfileRuleExplanation{}, errors.New("core: missing profile explanation sum")
		}
		definition, revision := semantic.Sum.Field.Reference()
		result.Kind = "sum"
		result.Sum = &ProfileSumExplanation{Start: rule.Start, End: rule.End, UnboundedStart: rule.UnboundedStart,
			Field:          FilterSumField{Definition: definition, MeaningRevision: revision},
			NativeCurrency: semantic.Sum.NativeCurrency, Bounds: ProfileSumBounds{Lower: node.Sum.Lower, Upper: node.Sum.Upper},
			Certain: node.Sum.Certain, Uncertain: node.Sum.Possible - node.Sum.Certain}
		if history := semantic.Sum.Scope.History; history != nil {
			definition, revision := history.Reference()
			result.Sum.History = &FilterHistory{Definition: definition, MeaningRevision: revision}
		}
	case *warehouses.IndependentOccurrenceCount:
		if node.Count == nil || semantic.Count == nil {
			return ProfileRuleExplanation{}, errors.New("core: missing profile explanation count")
		}
		result.Kind = "count"
		result.Count = &ProfileCountExplanation{Start: rule.Start, End: rule.End, Certain: node.Count.Certain,
			Uncertain: node.Count.Possible - node.Count.Certain, UnboundedStart: rule.UnboundedStart}
		if history := semantic.Count.Scope.History; history != nil {
			definition, revision := history.Reference()
			result.Count.History = &FilterHistory{Definition: definition, MeaningRevision: revision}
		}
		if currency := semantic.Count.Currency; currency != nil {
			result.Count.CurrencyField = &FilterCurrencyField{Definition: currency.Definition,
				MeaningRevision: currency.Revision}
			result.Count.NativeCurrency = currency.Value
		}
	default:
		return ProfileRuleExplanation{}, errors.New("core: invalid profile explanation condition")
	}

	return result, nil
}
