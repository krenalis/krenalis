// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestIngestionRateLimiterRejectsBatchBeforePublishing verifies that an
// over-limit batch is rejected without publishing any of its events.
func TestIngestionRateLimiterRejectsBatchBeforePublishing(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	organizations := k.Organizations(0, 1)
	if len(organizations) != 1 {
		t.Fatalf("expected one organization, got %d", len(organizations))
	}
	organization := organizations[0]
	limits := organization.Limits
	limits.API.Ingestion.BurstCapacity = 1
	k.UpdateOrganization(organization.ID, organization.Name, limits)

	connectionID := k.CreateJavaScriptSource("Rate-limited source", nil)
	writeKeys := k.EventWriteKeys(connectionID)
	if len(writeKeys) != 1 {
		t.Fatalf("expected one event write key, got %d", len(writeKeys))
	}
	k.CreatePipeline(connectionID, "Event", krenalistester.PipelineToSet{
		Name:    "Store rate-limited events",
		Enabled: true,
	})

	err := k.TryCall(http.MethodPost, "/v1/events",
		http.Header{"Authorization": []string{"Bearer " + writeKeys[0]}},
		[]map[string]any{
			{"type": "track", "userId": "user-1", "event": "first"},
			{"type": "track", "userId": "user-1", "event": "second"},
		}, nil)
	statusErr, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError, got %T", err)
	}
	if statusErr.Response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP status %d, got %d", http.StatusTooManyRequests, statusErr.Response.Code)
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

	time.Sleep(2 * time.Second)
	if count := k.CountEventsInWarehouse(t.Context()); count != 0 {
		t.Fatalf("expected no events stored after the rejected batch, got %d", count)
	}
}
