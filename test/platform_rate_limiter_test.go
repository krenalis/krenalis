// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestPlatformRateLimiter verifies that platform management API endpoints
// consume the global platform budget and expose retry guidance when it is
// exhausted.
func TestPlatformRateLimiter(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	k.ExecQueryTestDatabase(t.Context(), `
		UPDATE metadata
		SET requests_rate_per_minute = 60,
			requests_burst_capacity = 1
		WHERE singleton`)

	organizations, err := k.TryOrganizations(0, 1)
	if err != nil {
		t.Fatalf("consume initial platform capacity: %v", err)
	}
	if len(organizations) != 1 {
		t.Fatalf("expected one organization, got %d", len(organizations))
	}

	var availableUnits int
	k.QueryRowTestDatabase(t.Context(), &availableUnits, `
		SELECT available_units
		FROM rate_limit_buckets
		WHERE subject_kind = 'platform'
			AND subject_id = 'platform'`)
	if availableUnits != 0 {
		t.Fatalf("expected the initial request to exhaust the platform bucket, got %d available units", availableUnits)
	}

	// Prevent capacity from refilling while the second HTTP request is made, so
	// the response and retry duration are deterministic.
	k.ExecQueryTestDatabase(t.Context(), `
		UPDATE rate_limit_buckets
		SET last_refill_at = clock_timestamp() + INTERVAL '1 minute',
			refill_remainder = 0
		WHERE subject_kind = 'platform'
			AND subject_id = 'platform'`)

	err = k.CanGetOrganization(organizations[0].ID)
	statusErr, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError, got %T: %v", err, err)
	}
	if statusErr.Response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP status %d, got %d", http.StatusTooManyRequests, statusErr.Response.Code)
	}
	if retryAfter := statusErr.Response.Header.Get("Retry-After"); retryAfter != "1" {
		t.Fatalf("expected Retry-After 1, got %q", retryAfter)
	}
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(statusErr.Response.Text), &response); err != nil {
		t.Fatalf("expected JSON error response, got %q", statusErr.Response.Text)
	}
	if response.Error.Code != "TooManyRequests" {
		t.Fatalf("expected error code %q, got %q", "TooManyRequests", response.Error.Code)
	}
}
