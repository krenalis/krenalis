// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"math"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// ProfileQuery selects a profile page. At optionally fixes the Unified
// reference instant; it does not select a historical snapshot of profile data.
type ProfileQuery struct {
	Properties   []string
	Filter       *Filter
	Order        string
	OrderDesc    bool
	First, Limit int
	At           string
}

// ProfileQueryResult is a complete profile page and its matching total.
// Evaluation is present only when the request contains a behavioral condition.
type ProfileQueryResult struct {
	Profiles   []Profile
	Schema     types.Type
	Total      int
	Evaluation *ProfileEvaluation
}

// ProfileEvaluation contains exact outcomes from the page's own evaluation.
// At can anchor later requests without freezing their data view.
type ProfileEvaluation struct {
	At      time.Time `json:"at"`
	Match   int64     `json:"match"`
	NoMatch int64     `json:"noMatch"`
	Unknown int64     `json:"unknown"`
}

// QueryProfiles returns matching profiles and Unified evaluation metadata.
// It returns errors.BadRequestError for invalid request values,
// errors.NotFoundError if the workspace no longer exists,
// errors.UnavailableError on warehouse failure, and
// errors.UnprocessableError with the following codes:
//   - PropertyNotExist, OrderNotExist or OrderTypeNotSortable
//     for schema errors.
//   - FilterCapabilityNotImplemented for an unavailable language capability.
//   - UnifiedFilteringUnavailable for an unverified workspace backend.
//   - ProfileProjectionTooWide or ProfileResultTooLarge for page limits.
//   - MaintenanceMode when warehouse access is in maintenance.
func (this *Workspace) QueryProfiles(ctx context.Context, query ProfileQuery) (*ProfileQueryResult, error) {

	this.core.mustBeOpen()

	if !query.Filter.RequiresUnified() {
		if query.At != "" {
			return nil, errors.BadRequest("at requires a behavioral filter")
		}
		filter := query.Filter
		if filter != nil && filter.DefaultCalendar != nil {
			_, err := unified.ResolveCalendar(filter.DefaultCalendar.Definition, filter.DefaultCalendar.MeaningRevision)
			if err != nil {
				return nil, errors.BadRequest("filter is not valid: %s", err)
			}
			// Retain the public metadata; the Record executor has no use for it.
			record := *filter
			record.DefaultCalendar = nil
			filter = &record
		}
		profiles, schema, total, err := this.profiles(ctx, query.Properties, filter,
			query.Order, query.OrderDesc, query.First, query.Limit)
		if err != nil {
			return nil, err
		}
		return &ProfileQueryResult{Profiles: profiles, Schema: schema, Total: total}, nil
	}

	// Validate the projection and ordering against the current profile schema.
	properties := query.Properties
	schema := this.workspace.ProfileSchema
	if properties == nil {
		properties = schema.Properties().Names()
	}
	if len(properties) == 0 {
		return nil, errors.BadRequest("properties is empty")
	}
	if len(properties) > schema.Properties().Len() {
		return nil, errors.BadRequest("too many requested properties")
	}
	resultProperties := make([]types.Property, len(properties))
	seen := map[string]bool{}
	for i, name := range properties {
		if seen[name] {
			return nil, errors.BadRequest("duplicate property name")
		}
		seen[name] = true
		p, ok := schema.Properties().ByName(name)
		if !ok {
			if !types.IsValidPropertyName(name) {
				return nil, errors.BadRequest("property name %q is not valid", name)
			}
			return nil, errors.Unprocessable(PropertyNotExist, "property name %s does not exist", name)
		}
		resultProperties[i] = p
	}
	order := query.Order
	if order == "" {
		order = "_updated_at"
	} else {
		p, ok := schema.Properties().ByName(order)
		if !ok {
			if !types.IsValidPropertyName(order) {
				return nil, errors.BadRequest("order %q is not valid", order)
			}
			return nil, errors.Unprocessable(OrderNotExist, "order %s does not exist in schema", order)
		}
		switch p.Type.Kind() {
		case types.JSONKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return nil, errors.Unprocessable(OrderTypeNotSortable, "cannot sort by %s: property has type %s", order, p.Type)
		default:
			// The remaining scalar kinds have a warehouse order.
		}
	}
	if query.First < 0 || int64(query.First) > math.MaxInt32 || query.Limit < 1 || query.Limit > 1000 {
		return nil, errors.BadRequest("invalid first or limit")
	}

	// Fix T once, retaining microseconds without rounding to a future instant.
	now := time.Now().UTC()
	at := now.Truncate(time.Microsecond)
	if query.At != "" {
		parsed, err := parseFilterInstant(query.At)
		if err != nil {
			return nil, errors.BadRequest("invalid at: %s", err)
		}
		at = parsed.UTC()
		if at.IsZero() || at.Year() < 1 || at.Year() > 9999 || at.Nanosecond()%1000 != 0 || at.After(now) {
			return nil, errors.BadRequest("at must be a nonfuture timestamp with microsecond precision")
		}
	}
	nodes := 0
	root, err := prepareUnifiedRule(query.Filter, schema, 0, &nodes)
	if err != nil {
		if missing, ok := errors.AsType[types.PathNotExistError](err); ok {
			return nil, errors.Unprocessable(PropertyNotExist, "filter's property %s does not exist", missing.Path)
		}
		if unavailable, ok := errors.AsType[*unified.UnsupportedError](err); ok {
			return nil, errors.Unprocessable(FilterCapabilityNotImplemented, "%s", unavailable)
		}
		return nil, errors.BadRequest("filter is not valid: %s", err)
	}
	result, err := this.store.UnifiedProfiles(ctx, &root, unified.Bindings{}, at, unified.Page{
		Properties: append([]string{"_updated_at"}, properties...), OrderBy: order, OrderDesc: query.OrderDesc,
		First: query.First, Limit: query.Limit, MatchingOnly: true})
	if err != nil {
		if errors.Is(err, datastore.ErrProfileProjectionTooWide) {
			return nil, errors.Unprocessable(ProfileProjectionTooWide,
				"profile projection exceeds %d physical columns including metadata; request fewer properties",
				warehouses.MaxProfileProjectionColumns)
		}
		if errors.Is(err, datastore.ErrProfileResultTooLarge) {
			return nil, errors.Unprocessable(ProfileResultTooLarge,
				"profile page exceeds the supported size; reduce limit or request fewer properties")
		}
		if errors.Is(err, datastore.ErrUnifiedUnavailable) {
			return nil, errors.Unprocessable(UnifiedFilteringUnavailable, "Unified filtering is unavailable on this warehouse")
		}
		if unavailable, ok := errors.AsType[*unified.UnsupportedError](err); ok {
			return nil, errors.Unprocessable(FilterCapabilityNotImplemented, "%s", unavailable)
		}
		if errors.Is(err, datastore.ErrMaintenanceMode) {
			return nil, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if _, ok := errors.AsType[*datastore.WarehouseError](err); ok {
			return nil, errors.Unavailable("data warehouse profile query failed")
		}
		return nil, errors.New("core: profile query failed")
	}
	if result.Match > math.MaxInt {
		return nil, errors.New("core: profile total exceeds representation")
	}
	profiles := make([]Profile, len(result.Profiles))
	for i, profile := range result.Profiles {
		updated, ok := profile.Attributes["_updated_at"].(time.Time)
		if !ok {
			return nil, errors.New("core: missing profile update timestamp")
		}
		delete(profile.Attributes, "_updated_at")
		profiles[i] = Profile{KPID: profile.ID, UpdatedAt: updated, Attributes: profile.Attributes}
	}

	return &ProfileQueryResult{Profiles: profiles, Schema: types.Object(resultProperties), Total: int(result.Match),
		Evaluation: &ProfileEvaluation{At: result.At, Match: result.Match,
			NoMatch: result.NoMatch, Unknown: result.Unknown}}, nil
}

func (this *Workspace) profiles(ctx context.Context, properties []string, filter *Filter, order string, orderDesc bool, first, limit int) ([]Profile, types.Type, int, error) {

	ws := this.workspace

	profileProperties := ws.ProfileSchema.Properties()

	// Validate the properties.
	if properties == nil {
		properties = profileProperties.Names()
	} else {
		if len(properties) == 0 {
			return nil, types.Type{}, 0, errors.BadRequest("properties is empty")
		}
		for _, name := range properties {
			if _, ok := profileProperties.ByName(name); !ok {
				if name == "" {
					return nil, types.Type{}, 0, errors.BadRequest("a property name is empty")
				}
				if !types.IsValidPropertyName(name) {
					return nil, types.Type{}, 0, errors.BadRequest("property name %q is not valid", name)
				}
				return nil, types.Type{}, 0, errors.Unprocessable(PropertyNotExist, "property name %s does not exist", name)
			}
		}
	}

	// Validate the filter.
	var where *state.Where
	if filter != nil {
		prepared, _, err := prepareFilter(filter, ws.ProfileSchema)
		if err != nil {
			if err, ok := err.(types.PathNotExistError); ok {
				return nil, types.Type{}, 0, errors.Unprocessable(PropertyNotExist, "filter's property %s does not exist", err.Path)
			}
			return nil, types.Type{}, 0, errors.BadRequest("filter is not valid: %w", err)
		}
		where = prepared
	}

	// Validate the order.
	if order != "" {
		orderProperty, ok := profileProperties.ByName(order)
		if !ok {
			if !types.IsValidPropertyName(order) {
				return nil, types.Type{}, 0, errors.BadRequest("order %q is not a valid property name", order)
			}
			return nil, types.Type{}, 0, errors.Unprocessable(OrderNotExist, "order %s does not exist in schema", order)
		}
		switch orderProperty.Type.Kind() {
		case types.JSONKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return nil, types.Type{}, 0, errors.Unprocessable(OrderTypeNotSortable,
				"cannot sort by %s: property has type %s", order, orderProperty.Type)
		}
	} else {
		order = "_updated_at"
	}

	// Validate first and limit.
	if first < 0 || first > maxInt32 {
		return nil, types.Type{}, 0, errors.BadRequest("first %d in not valid", first)
	}
	if limit < 1 || limit > 1000 {
		return nil, types.Type{}, 0, errors.BadRequest("limit %d is not valid", limit)
	}

	// Read the profiles.
	rows, total, err := this.store.Profiles(ctx, datastore.Query{
		Properties: append([]string{"_kpid", "_updated_at"}, properties...),
		Where:      where,
		OrderBy:    order,
		OrderDesc:  orderDesc,
		First:      first,
		Limit:      limit,
	})
	if err != nil {
		if err == datastore.ErrMaintenanceMode {
			return nil, types.Type{}, 0, errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if err, ok := err.(*datastore.WarehouseError); ok {
			return nil, types.Type{}, 0, errors.Unavailable("%s", err)
		}
		return nil, types.Type{}, 0, err
	}

	// Create the schema to return, with only the requested properties.
	props := make([]types.Property, len(properties))
	for i, name := range properties {
		props[i], _ = profileProperties.ByName(name)
	}
	schema := types.Object(props)

	profiles := make([]Profile, len(rows))
	for i, row := range rows {
		profiles[i].KPID = row["_kpid"].(string)
		profiles[i].UpdatedAt = row["_updated_at"].(time.Time)
		profiles[i].Attributes = row
		delete(row, "_kpid")
		delete(row, "_updated_at")
	}

	return profiles, schema, total, nil
}
