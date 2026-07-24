// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
)

// rateLimitSubjectStub records rate-limit capacity consumption in tests.
type rateLimitSubjectStub struct {
	calls int
	cost  int
	err   error
}

// ConsumeRateLimitCapacity records the requested test cost.
func (subject *rateLimitSubjectStub) ConsumeRateLimitCapacity(_ context.Context, cost int) error {
	subject.calls++
	subject.cost = cost
	return subject.err
}

// TestApplyRateLimitTo verifies rate-limit exemptions, consumption, and errors.
func TestApplyRateLimitTo(t *testing.T) {

	t.Run("skips Admin authentication", func(t *testing.T) {
		subject := &rateLimitSubjectStub{}
		authenticated := authenticatedRequest{rateLimitExempt: true}

		err := authenticated.applyRateLimitTo(context.Background(), subject, 3)
		if err != nil {
			t.Fatalf("apply rate limit: %v", err)
		}
		if subject.calls != 0 {
			t.Fatalf("expected no consumption, got %d calls", subject.calls)
		}
	})

	t.Run("consumes the selected organization budget", func(t *testing.T) {
		organization := &rateLimitSubjectStub{}
		authenticated := authenticatedRequest{}

		err := authenticated.applyRateLimitTo(context.Background(), organization, 3)
		if err != nil {
			t.Fatalf("apply rate limit: %v", err)
		}
		if organization.calls != 1 || organization.cost != 3 {
			t.Fatalf("expected organization consumption with cost 3, got calls=%d cost=%d", organization.calls, organization.cost)
		}
	})

	t.Run("consumes the selected workspace budget", func(t *testing.T) {
		workspace := &rateLimitSubjectStub{}
		authenticated := authenticatedRequest{}

		err := authenticated.applyRateLimitTo(context.Background(), workspace, 3)
		if err != nil {
			t.Fatalf("apply rate limit: %v", err)
		}
		if workspace.calls != 1 || workspace.cost != 3 {
			t.Fatalf("expected workspace consumption with cost 3, got calls=%d cost=%d", workspace.calls, workspace.cost)
		}
	})

	t.Run("maps exhausted capacity to Too Many Requests", func(t *testing.T) {
		subject := &rateLimitSubjectStub{err: core.ErrAPICapacityExceeded}

		err := (authenticatedRequest{}).applyRateLimitTo(context.Background(), subject, 3)
		rateLimitError, ok := err.(*errors.TooManyRequestsError)
		if !ok {
			t.Fatalf("expected TooManyRequestsError, got %T", err)
		}
		response := httptest.NewRecorder()
		if err := rateLimitError.WriteTo(response); err != nil {
			t.Fatalf("write response: %v", err)
		}
		if response.Code != http.StatusTooManyRequests {
			t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, response.Code)
		}
	})

	t.Run("propagates invalid API costs", func(t *testing.T) {
		subject := &rateLimitSubjectStub{err: core.ErrInvalidAPICost}

		err := (authenticatedRequest{}).applyRateLimitTo(context.Background(), subject, 3)
		if !stderrors.Is(err, core.ErrInvalidAPICost) {
			t.Fatalf("expected invalid API cost, got %v", err)
		}
	})

	t.Run("propagates operational errors", func(t *testing.T) {
		operationalError := stderrors.New("rate limiter unavailable")
		subject := &rateLimitSubjectStub{err: operationalError}

		err := (authenticatedRequest{}).applyRateLimitTo(context.Background(), subject, 3)
		if !stderrors.Is(err, operationalError) {
			t.Fatalf("expected operational error, got %v", err)
		}
	})
}

// TestScopedRateLimitSubject verifies budget selection for scope-aware requests.
func TestScopedRateLimitSubject(t *testing.T) {
	organization := &core.Organization{}
	workspace := &core.Workspace{}

	if got := (authenticatedRequest{organization: organization}).scopedRateLimitSubject(); got != organization {
		t.Fatalf("unscoped rate-limit subject = %T, want organization", got)
	}
	if got := (authenticatedRequest{organization: organization, workspace: workspace}).scopedRateLimitSubject(); got != workspace {
		t.Fatalf("scoped rate-limit subject = %T, want workspace", got)
	}
}
