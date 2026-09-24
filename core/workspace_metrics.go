// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/tools/errors"
)

// IdentityMetric contains the latest observed identity metrics for the
// workspace.
type IdentityMetric struct {
	ObservedAt time.Time `json:"observedAt"`

	Total          int `json:"total"`
	Anonymous      int `json:"anonymous"`
	Recognized     int `json:"recognized"`
	WithoutProfile int `json:"withoutProfile"`

	Connections []IdentityConnectionMetric `json:"connections"`
}

// IdentityConnectionMetric contains identity metrics for one source
// connection.
type IdentityConnectionMetric struct {
	Connection     string `json:"connection"`
	Anonymous      int    `json:"anonymous"`
	Recognized     int    `json:"recognized"`
	WithoutProfile int    `json:"withoutProfile"`
}

// IdentityMetricDay contains a known identity state for one UTC day.
type IdentityMetricDay struct {
	Day        string `json:"day"`
	Total      int    `json:"total"`
	Anonymous  int    `json:"anonymous"`
	Recognized int    `json:"recognized"`
}

// IdentityMetricsPerDate returns known daily identity states in [start,end),
// ordered by day. Days without a known state may be omitted, and the result may
// be empty. A nil connectionSelection selects workspace totals, a connection
// identifier selects that connection, and "deleted" selects the aggregate
// contribution of deleted connections.
//
// IdentityMetricsPerDate returns an [errors.BadRequestError] when the
// connection selection is invalid or the interval exceeds the maximum size. It
// returns an [errors.NotFoundError] when the interval is invalid or the
// selected connection does not exist.
func (this *Workspace) IdentityMetricsPerDate(ctx context.Context, start, end time.Time, connectionSelection *string) ([]IdentityMetricDay, error) {

	this.core.mustBeOpen()

	// Validate start and end.
	start, end, err := validateWorkspaceMetricsRange(start, end, time.Now())
	if err != nil {
		return nil, errors.NotFound("%s", err)
	}
	days := int(end.Sub(start) / (24 * time.Hour))
	if days > maxEntryDays {
		return nil, errors.BadRequest("requested metrics exceed the maximum: %d days is more than 60,000", days)
	}

	// Validate the connection selection.
	if connectionSelection != nil && *connectionSelection != "deleted" && !IsValidID(*connectionSelection) {
		return nil, errors.BadRequest(`value %q is neither a valid connection identifier nor "deleted"`, *connectionSelection)
	}

	source, err := this.core.metrics.Identities.MetricsPerDate(ctx, this.workspace.ID, start, end, connectionSelection)
	if err != nil {
		if errors.Is(err, metrics.ErrConnectionNotFound) {
			return nil, errors.NotFound("connection %s does not exist", *connectionSelection)
		}
		return nil, err
	}

	result := make([]IdentityMetricDay, len(source))
	for index, day := range source {
		result[index] = IdentityMetricDay{
			Day:        day.Day.Format(time.DateOnly),
			Total:      day.Total,
			Anonymous:  day.Anonymous,
			Recognized: day.Recognized,
		}
	}

	return result, nil
}

// LatestIdentityMetric returns the latest observed workspace identity state
// with its live-connection breakdown. It returns an [errors.NotFoundError] if
// the workspace does not exist.
func (this *Workspace) LatestIdentityMetric(ctx context.Context) (IdentityMetric, error) {

	this.core.mustBeOpen()

	source, err := this.core.metrics.Identities.Latest(ctx, this.workspace.ID)
	if err != nil {
		if errors.Is(err, metrics.ErrWorkspaceNotFound) {
			return IdentityMetric{}, errors.NotFound("workspace %s does not exist", this.workspace.ID)
		}
		return IdentityMetric{}, err
	}
	result := IdentityMetric{
		ObservedAt:     source.ObservedAt,
		Total:          source.Total,
		Anonymous:      source.Anonymous,
		Recognized:     source.Recognized,
		WithoutProfile: source.WithoutProfile,
		Connections:    make([]IdentityConnectionMetric, len(source.Connections)),
	}
	for i, connection := range source.Connections {
		result.Connections[i] = IdentityConnectionMetric(connection)
	}

	return result, nil
}

// RefreshIdentityMetrics observes and stores identity metrics for the
// workspace.
//
// It returns an [errors.NotFoundError] if the workspace does not exist, an
// [errors.UnprocessableError] with code [MaintenanceMode] if the data warehouse
// is in maintenance mode, or an [errors.UnavailableError] if the data warehouse
// cannot be accessed.
func (this *Workspace) RefreshIdentityMetrics(ctx context.Context) error {

	this.core.mustBeOpen()

	err := this.core.metrics.Identities.Refresh(ctx, this.workspace.ID)
	if err != nil {
		if errors.Is(err, metrics.ErrWorkspaceNotFound) {
			return errors.NotFound("workspace %s does not exist", this.workspace.ID)
		}
		if err == datastore.ErrMaintenanceMode {
			return errors.Unprocessable(MaintenanceMode, "data warehouse is in maintenance mode")
		}
		if _, ok := errors.AsType[*datastore.UnavailableError](err); ok {
			return errors.Unavailable("data warehouse is unavailable")
		}
		return err
	}

	return nil
}

// validateWorkspaceMetricsRange normalizes and validates a workspace metric
// daily interval against the supplied current time.
func validateWorkspaceMetricsRange(start, end, now time.Time) (time.Time, time.Time, error) {

	// Normalize start and end.
	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)

	// Validate start and end.
	if start.Before(metrics.MinTime) {
		return time.Time{}, time.Time{}, errors.New("start date is too far in the past")
	}
	if end.After(metrics.MaxTime) {
		return time.Time{}, time.Time{}, errors.New("end date is too far in the future")
	}
	if end.After(now.UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)) {
		return time.Time{}, time.Time{}, errors.New("end date is too far in the future")
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, errors.New("day of the end date must be after the day of the start date")
	}

	return start, end, nil
}
