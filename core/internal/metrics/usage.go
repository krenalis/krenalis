// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/errors"
)

// Usage collects, stores, and queries usage metrics.
type Usage struct {
	metrics *Metrics
	sync.Mutex
	// TODO: Bound event counts end-to-end or use types that ensure both pending
	// and persisted daily totals remain representable.
	events map[eventCountKey]int // Pending event count increments.
	buf    bytes.Buffer
}

// UsageSeries represents daily usage metrics for one scope. Workspace is empty
// for the organization series.
type UsageSeries struct {
	Workspace string
	Days      []UsageDay
}

// UsageDay represents one UTC day of usage.
type UsageDay struct {
	Day         time.Time
	Profiles    int
	ProfilesAvg int
	Events      int

	profileSeconds big.Int
	seconds        int
}

// newUsageSeries initializes a usage series for the requested interval and
// records the number of elapsed seconds available in each day.
func newUsageSeries(workspace string, start, calculatedAt time.Time, days int) (UsageSeries, error) {
	series := UsageSeries{
		Workspace: workspace,
		Days:      make([]UsageDay, days),
	}
	calculatedDay := calculatedAt.UTC().Truncate(24 * time.Hour)
	for i := range series.Days {
		day := start.AddDate(0, 0, i)
		var seconds int
		switch {
		case day.Before(calculatedDay):
			seconds = 24 * 60 * 60
		case day.Equal(calculatedDay):
			seconds = int(calculatedAt.Unix() - day.Unix())
		default:
			return UsageSeries{}, fmt.Errorf("usage metrics interval includes a future day")
		}
		series.Days[i].Day = day
		series.Days[i].seconds = seconds
	}
	return series, nil
}

// IngestedEvents records count events accepted by the collector for workspace
// at receivedAt. It only updates an in-memory buffer and is safe to call
// concurrently.
func (u *Usage) IngestedEvents(organization, workspace string, receivedAt time.Time, count int) {
	if count <= 0 {
		return
	}
	key := eventCountKey{
		organization: organization,
		workspace:    workspace,
		day:          receivedAt.UTC().Truncate(24 * time.Hour),
	}
	u.Lock()
	u.events[key] += count
	u.Unlock()
}

// RecordProfileObservation records an observed workspace profile count and
// accounts for how long the previous count remained active. The caller must
// hold a lock on the corresponding row in workspaces.
func (u *Usage) RecordProfileObservation(ctx context.Context, tx *db.Tx, organization, workspace string, profileCount int, observedAt time.Time) error {

	observedAt = observedAt.UTC().Truncate(time.Microsecond)
	day := observedAt.Truncate(24 * time.Hour)

	// Load the most recent profile observation up to this day.
	var (
		sameDay                bool
		previousProfileCount   int
		previousProfileSeconds int
		previousObservedAt     time.Time
	)
	err := tx.QueryRow(ctx,
		`SELECT day = $3, profiles, profile_seconds,
			(day + observed_at) AT TIME ZONE 'UTC'
		FROM usage_metrics
		WHERE organization = $1
			AND workspace = $2
			AND day <= $3
			AND observed_at IS NOT NULL
		ORDER BY day DESC
		LIMIT 1`, organization, workspace, day).Scan(
		&sameDay, &previousProfileCount, &previousProfileSeconds, &previousObservedAt)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	// Accumulate the time spent at the previously observed profile count.
	var profileSeconds int
	if hasPrevious := err == nil; hasPrevious {
		if observedAt.Before(previousObservedAt) {
			return fmt.Errorf("profile observation time precedes the previous state")
		}
		var elapsedSeconds int
		if sameDay {
			elapsedSeconds = int(observedAt.Unix() - previousObservedAt.Unix())
			profileSeconds = previousProfileSeconds
		} else {
			elapsedSeconds = int(observedAt.Unix() - day.Unix())
		}
		// previousProfileCount <= math.MaxInt32 and the total elapsed time in a day is
		// < 86,400 seconds, so profileSeconds remains below 2^48 and fits in a 64-bit int.
		profileSeconds += previousProfileCount * elapsedSeconds
	}

	// Store the new profile state while preserving any events already recorded for the day.
	_, err = tx.Exec(ctx,
		`INSERT INTO usage_metrics AS m
			(organization, workspace, day, profiles, profile_seconds, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (organization, workspace, day) DO UPDATE SET
			profiles = EXCLUDED.profiles,
			profile_seconds = EXCLUDED.profile_seconds,
			observed_at = EXCLUDED.observed_at`,
		organization, workspace, day, profileCount, profileSeconds, observedAt.Format("15:04:05.999999"))

	return err
}

// MetricsPerDate returns daily usage metrics for the interval [start,end).
// Start and end must be UTC day boundaries, and end must be after start.
// It returns one organization series when workspaces is nil, or one series
// for each workspace otherwise. Missing profile observations are carried
// forward, and missing event counts are treated as zero.
//
// It returns [ErrMetricResultTooLarge] when an organization total cannot be
// represented by the result type.
func (u *Usage) MetricsPerDate(ctx context.Context, organization string, start, end time.Time, workspaces []string) ([]UsageSeries, error) {

	if workspaces != nil && len(workspaces) == 0 {
		return []UsageSeries{}, nil
	}

	var scope string
	if workspaces != nil {
		var b strings.Builder
		b.WriteString(" AND workspace IN (")
		for i, workspace := range workspaces {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(db.Quote(workspace))
		}
		b.WriteByte(')')
		scope = b.String()
	}

	// Retrieve the last profile state before start for each workspace
	// and all rows in [start, end), using the same calculation timestamp.
	// Previous-state rows have a NULL day.
	query := `(SELECT DISTINCT ON (m.workspace)
	statement_timestamp(), m.workspace, NULL::date, m.profiles,
	0::bigint, NULL::timestamptz, 0::bigint
FROM usage_metrics AS m
WHERE m.organization = $1` + scope + `
	AND m.day < $2
	AND m.observed_at IS NOT NULL
ORDER BY m.workspace, m.day DESC)
UNION ALL
SELECT statement_timestamp(), m.workspace, m.day, m.profiles,
	m.profile_seconds, (m.day + m.observed_at) AT TIME ZONE 'UTC', m.events
FROM usage_metrics AS m
WHERE m.organization = $1` + scope + `
	AND m.day >= $2 AND m.day < $3`

	rows, err := u.metrics.db.Query(ctx, query, organization, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	days := int(end.Sub(start) / (24 * time.Hour))
	stored := make(map[string]*usageStoredWorkspace)
	calculatedAt := time.Now().UTC()
	for rows.Next() {
		var workspace string
		var day *time.Time
		var v usageStoredDay
		err := rows.Scan(&calculatedAt, &workspace, &day, &v.profiles, &v.profileSeconds, &v.observedAt, &v.events)
		if err != nil {
			return nil, err
		}
		if day == nil && v.profiles == 0 {
			continue
		}
		s := stored[workspace]
		if s == nil {
			s = &usageStoredWorkspace{days: make(map[int]usageStoredDay)}
			stored[workspace] = s
		}
		if day == nil {
			s.profiles = v.profiles
			continue
		}
		metricDay := day.UTC()
		dayIndex := int(metricDay.Sub(start) / (24 * time.Hour))
		if dayIndex < 0 || dayIndex >= days {
			return nil, fmt.Errorf("metrics: usage metrics contain a day outside the requested interval")
		}
		s.days[dayIndex] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	calculatedAt = calculatedAt.UTC()
	if workspaces == nil {
		organizationSeries, err := newUsageSeries("", start, calculatedAt, days)
		if err != nil {
			return nil, err
		}
		for _, s := range stored {
			workspaceSeries, err := buildUsageSeries("", start, calculatedAt, days, s)
			if err != nil {
				return nil, err
			}
			for i := range organizationSeries.Days {
				d := &organizationSeries.Days[i]
				workspaceDay := &workspaceSeries.Days[i]
				if workspaceDay.Profiles > math.MaxInt-d.Profiles ||
					workspaceDay.Events > math.MaxInt-d.Events {
					return nil, ErrMetricResultTooLarge
				}
				d.Profiles += workspaceDay.Profiles
				d.profileSeconds.Add(&d.profileSeconds, &workspaceDay.profileSeconds)
				d.Events += workspaceDay.Events
			}
		}
		for i := range organizationSeries.Days {
			d := &organizationSeries.Days[i]
			d.ProfilesAvg, err = usageAverage(&d.profileSeconds, d.seconds, d.Profiles)
			if err != nil {
				return nil, err
			}
		}
		return []UsageSeries{organizationSeries}, nil
	}

	result := make([]UsageSeries, len(workspaces))
	for i, workspace := range workspaces {
		s := stored[workspace]
		if s == nil {
			s = &usageStoredWorkspace{days: make(map[int]usageStoredDay)}
		}
		series, err := buildUsageSeries(workspace, start, calculatedAt, days, s)
		if err != nil {
			return nil, err
		}
		result[i] = series
	}

	return result, nil
}

// start periodically stores buffered event counts and attempts a final flush
// when the collector shuts down.
func (u *Usage) start() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	events := map[eventCountKey]int{}

	for {
		isShuttingDown := false
		select {
		case <-ticker.C:
		case <-u.metrics.close.stop:
			isShuttingDown = true
		case <-u.metrics.close.ctx.Done():
			return
		}
		u.Lock()
		events, u.events = u.events, events
		u.Unlock()
		if err := u.store(events); err != nil {
			return
		}
		clear(events)
		if isShuttingDown {
			return
		}
	}
}

// store persists a batch of event count increments, retrying after database
// errors until it succeeds. It returns an error if shutdown cancels storage
// before the batch has been stored.
func (u *Usage) store(eventCounts map[eventCountKey]int) error {

	if len(eventCounts) == 0 {
		return nil
	}

	b := &u.buf
	b.Reset()
	b.WriteString("WITH deltas(organization, workspace, day, events) AS (\n\tVALUES ")
	i := 0
	for key, count := range eventCounts {
		if count == 0 {
			continue
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('(')
		b.WriteString(db.Quote(key.organization))
		b.WriteByte(',')
		b.WriteString(db.Quote(key.workspace))
		b.WriteByte(',')
		b.WriteString(db.Quote(key.day.Format(time.DateOnly)))
		b.WriteString("::date,")
		b.WriteString(strconv.Itoa(count))
		b.WriteByte(')')
		i++
	}
	if i == 0 {
		return nil
	}
	b.WriteString(`
)
INSERT INTO usage_metrics AS m
	(organization, workspace, day, events)
SELECT d.organization, d.workspace, d.day, d.events
FROM deltas AS d
WHERE EXISTS (SELECT 1 FROM organizations AS o WHERE o.id = d.organization)
ON CONFLICT (organization, workspace, day) DO UPDATE SET
	events = m.events + EXCLUDED.events`)

	statement := b.String()
	var loggedMsg string
	bo := backoff.New(20)
	for bo.Next(u.metrics.close.ctx) {
		_, err := u.metrics.db.Exec(u.metrics.close.ctx, statement)
		if err == nil {
			return nil
		}
		if msg := err.Error(); msg != loggedMsg && u.metrics.close.ctx.Err() == nil {
			slog.Error("core/metrics: failed to store usage metrics", "error", msg)
			loggedMsg = msg
		}
	}
	if err := u.metrics.close.ctx.Err(); err != nil {
		return err
	}

	return nil
}

// eventCountKey identifies buffered event count increments for one organization,
// workspace, and UTC day.
type eventCountKey struct {
	organization string
	workspace    string
	day          time.Time
}

// usageStoredDay contains the stored usage values for a requested day.
// observedAt is nil when the row contains no profile observation.
type usageStoredDay struct {
	profiles       int
	profileSeconds int
	observedAt     *time.Time
	events         int
}

// usageStoredWorkspace contains the profile state before the requested
// interval and the stored days, indexed by offset from the start date.
type usageStoredWorkspace struct {
	profiles int
	days     map[int]usageStoredDay
}

// buildUsageSeries builds a daily usage series, carrying forward the latest
// profile count and calculating its time-weighted average for each day.
// It returns [ErrMetricResultTooLarge] when a rounded profile average cannot be
// represented by int.
func buildUsageSeries(workspace string, start, calculatedAt time.Time, days int, stored *usageStoredWorkspace) (UsageSeries, error) {
	series, err := newUsageSeries(workspace, start, calculatedAt, days)
	if err != nil {
		return UsageSeries{}, err
	}
	currentProfiles := stored.profiles
	for i := range series.Days {
		day := &series.Days[i]
		storedDay := stored.days[i]
		day.Events = storedDay.events
		var seconds big.Int
		day.profileSeconds.SetInt64(int64(currentProfiles))
		seconds.SetInt64(int64(day.seconds))
		day.profileSeconds.Mul(&day.profileSeconds, &seconds)
		if observedAt := storedDay.observedAt; observedAt != nil {
			secondsBeforeObservation := int(observedAt.Unix() - day.Day.Unix())
			if secondsBeforeObservation < 0 || secondsBeforeObservation > day.seconds {
				return UsageSeries{}, fmt.Errorf("usage metrics contain a profile observation outside its calculated interval")
			}
			secondsAfterObservation := day.seconds - secondsBeforeObservation
			day.profileSeconds.SetInt64(int64(storedDay.profileSeconds))
			var remainingProfileSeconds big.Int
			remainingProfileSeconds.SetInt64(int64(storedDay.profiles))
			seconds.SetInt64(int64(secondsAfterObservation))
			remainingProfileSeconds.Mul(&remainingProfileSeconds, &seconds)
			day.profileSeconds.Add(&day.profileSeconds, &remainingProfileSeconds)
			currentProfiles = storedDay.profiles
		}
		day.Profiles = currentProfiles
		day.ProfilesAvg, err = usageAverage(&day.profileSeconds, day.seconds, currentProfiles)
		if err != nil {
			return UsageSeries{}, err
		}
	}
	return series, nil
}

// usageAverage returns the time-weighted average profile count rounded to the
// nearest integer. For an empty interval, it returns the current profile count.
// It returns [ErrMetricResultTooLarge] when the rounded average cannot be
// represented by int.
func usageAverage(profileSeconds *big.Int, seconds, profiles int) (int, error) {
	if seconds <= 0 {
		return profiles, nil
	}
	var average, denominator, remainder big.Int
	denominator.SetInt64(int64(seconds))
	average.QuoRem(profileSeconds, &denominator, &remainder)
	var roundingThreshold big.Int
	roundingThreshold.SetInt64(int64((seconds + 1) / 2))
	if remainder.Cmp(&roundingThreshold) >= 0 {
		var one big.Int
		one.SetInt64(1)
		average.Add(&average, &one)
	}
	var minimum, maximum big.Int
	minimum.SetInt64(int64(math.MinInt))
	maximum.SetInt64(int64(math.MaxInt))
	if average.Cmp(&minimum) < 0 || average.Cmp(&maximum) > 0 {
		return 0, ErrMetricResultTooLarge
	}
	return int(average.Int64()), nil
}
