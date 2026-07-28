// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/krenalis/krenalis/core"
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

	t.Run("propagates rate-limit errors", func(t *testing.T) {
		rateLimitErr := stderrors.New("rate-limit capacity unavailable")
		subject := &rateLimitSubjectStub{err: rateLimitErr}

		err := (authenticatedRequest{}).applyRateLimitTo(context.Background(), subject, 3)
		if err != rateLimitErr {
			t.Fatalf("expected rate-limit error %v, got %v", rateLimitErr, err)
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
		t.Fatalf("expected unscoped rate-limit subject organization, got %T", got)
	}
	if got := (authenticatedRequest{organization: organization, workspace: workspace}).scopedRateLimitSubject(); got != workspace {
		t.Fatalf("expected scoped rate-limit subject workspace, got %T", got)
	}
}
