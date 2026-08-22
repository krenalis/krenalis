// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"slices"
	"time"

	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/tools/errors"
)

// UsageMetrics contains daily usage metrics over a UTC time range.
type UsageMetrics struct {
	Start   time.Time           `json:"start"`
	End     time.Time           `json:"end"`
	Metrics []UsageMetricSeries `json:"metrics"`
}

// UsageMetricSeries contains organization or workspace daily usage metrics.
// Workspace is omitted for the organization series.
type UsageMetricSeries struct {
	Workspace string           `json:"workspace,omitzero"`
	Days      []UsageMetricDay `json:"days"`
}

// UsageMetricDay contains usage metrics for one UTC day.
type UsageMetricDay struct {
	Day         string `json:"day"`
	Profiles    int    `json:"profiles"`
	ProfilesAvg int    `json:"profilesAvg"`
	Events      int    `json:"events"`
}

// UsageMetricsPerDate returns daily usage metrics in [start,end). If workspaces
// is nil, it returns the organization-level series unless workspaceScope
// restricts the request to a single workspace. If workspaces is non-nil, it
// returns one series for each selected workspace within the request scope.
//
// It returns an errors.UnprocessableError with code MetricResultTooLarge
// when an organization total cannot be represented by the result type.
func (this *Organization) UsageMetricsPerDate(ctx context.Context, start, end time.Time, workspaceScope string, workspaces []string) (UsageMetrics, error) {

	this.core.mustBeOpen()

	// Normalize start and end to UTC days.
	start = start.UTC().Truncate(24 * time.Hour)
	end = end.UTC().Truncate(24 * time.Hour)

	// Validate start and end.
	if start.Before(metrics.MinTime) {
		return UsageMetrics{}, errors.NotFound("start date is too far in the past")
	}
	if end.After(metrics.MaxTime) {
		return UsageMetrics{}, errors.NotFound("end date is too far in the future")
	}
	if end.After(time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)) {
		return UsageMetrics{}, errors.NotFound("end date is too far in the future")
	}
	if !end.After(start) {
		return UsageMetrics{}, errors.NotFound("day of the end date must be after the day of the start date")
	}

	// Validate workspace scope.
	if workspaceScope != "" && !IsValidID(workspaceScope) {
		return UsageMetrics{}, errors.BadRequest("workspace %q is not valid", workspaceScope)
	}

	// Validate workspace selection.
	var selectedWorkspaces []string
	if workspaces != nil {
		if len(workspaces) == 0 {
			return UsageMetrics{}, errors.BadRequest("workspaces must not be empty when provided")
		}
		if len(workspaces) > 1000 {
			return UsageMetrics{}, errors.BadRequest("workspaces must contain at most 1,000 entries")
		}
		for _, workspace := range workspaces {
			if !IsValidID(workspace) {
				return UsageMetrics{}, errors.BadRequest("workspace %q is not valid", workspace)
			}
		}
		selectedWorkspaces = slices.Clone(workspaces)
		selectedWorkspaces = slices.DeleteFunc(selectedWorkspaces, func(id string) bool {
			_, ok := this.organization.Workspace(id)
			return !ok || workspaceScope != "" && id != workspaceScope
		})
		if len(selectedWorkspaces) == 0 {
			return UsageMetrics{Start: start, End: end, Metrics: []UsageMetricSeries{}}, nil
		}
		slices.Sort(selectedWorkspaces)
		selectedWorkspaces = slices.Compact(selectedWorkspaces)
	} else if workspaceScope != "" {
		if _, ok := this.organization.Workspace(workspaceScope); !ok {
			return UsageMetrics{Start: start, End: end, Metrics: []UsageMetricSeries{}}, nil
		}
		selectedWorkspaces = []string{workspaceScope}
	}

	// Validate the number of series and days in the response.
	seriesCount := max(1, len(selectedWorkspaces))
	days := int(end.Sub(start) / (24 * time.Hour))
	if days*seriesCount > maxEntryDays {
		return UsageMetrics{}, errors.BadRequest("requested metrics exceed the maximum: %d series × %d days is more than 60,000", seriesCount, days)
	}

	// Retrieve usage metrics for the organization or selected workspaces.
	usageSeries, err := this.core.metrics.Usage.MetricsPerDate(ctx, this.organization.ID, start, end, selectedWorkspaces)
	if err != nil {
		if errors.Is(err, metrics.ErrMetricResultTooLarge) {
			return UsageMetrics{}, errors.Unprocessable(MetricResultTooLarge,
				"requested usage metric total exceeds the supported range; request metrics for individual workspaces instead")
		}
		return UsageMetrics{}, err
	}

	// Convert internal usage metrics to the public representation.
	result := UsageMetrics{
		Start:   start,
		End:     end,
		Metrics: make([]UsageMetricSeries, len(usageSeries)),
	}
	for i, sourceSeries := range usageSeries {
		series := &result.Metrics[i]
		series.Workspace = sourceSeries.Workspace
		series.Days = make([]UsageMetricDay, len(sourceSeries.Days))
		for dayIndex, sourceDay := range sourceSeries.Days {
			day := &series.Days[dayIndex]
			day.Day = sourceDay.Day.Format(time.DateOnly)
			day.Profiles = sourceDay.Profiles
			day.ProfilesAvg = sourceDay.ProfilesAvg
			day.Events = sourceDay.Events
		}
	}

	return result, nil
}
