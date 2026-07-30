// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
)

func TestTranslateRateLimitError(t *testing.T) {
	t.Run("capacity exceeded", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", state.CapacityExceededError{RetryAfter: 1500 * time.Millisecond})
		err = translateRateLimitError(err)
		responseErr, ok := err.(*errors.TooManyRequestsError)
		if !ok {
			t.Fatalf("expected TooManyRequestsError, got %T", err)
		}
		if responseErr.RetryAfter != 1500*time.Millisecond {
			t.Fatalf("expected retry-after 1.5s, got %v", responseErr.RetryAfter)
		}
	})

	t.Run("limiter unavailable", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", state.ErrRateLimiterUnavailable)
		if err := translateRateLimitError(err); err == nil {
			t.Fatal("expected UnavailableError, got nil")
		} else if _, ok := err.(*errors.UnavailableError); !ok {
			t.Fatalf("expected UnavailableError, got %T", err)
		}
	})

	t.Run("other error", func(t *testing.T) {
		err := stderrors.New("other")
		if got := translateRateLimitError(err); got != err {
			t.Fatalf("expected original error, got %v", got)
		}
	})

	if err := translateRateLimitError(nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// TestValidateOrganizationRateLimits verifies the accepted boundaries and
// rejects values outside the configured organization rate-limit ranges.
func TestValidateOrganizationRateLimits(t *testing.T) {
	valid := func() OrganizationLimits {
		return OrganizationLimits{
			Members: 1,
			Rates: RateLimits{
				OrganizationSpecific: RateLimit{RatePerMinute: minRequestRatePerMinute, MaxCapacity: minRequestMaxCapacity},
				WorkspaceSpecific:    RateLimit{RatePerMinute: minRequestRatePerMinute, MaxCapacity: minRequestMaxCapacity},
				EventsSpecific:       RateLimit{RatePerMinute: minEventRatePerMinute, MaxCapacity: minEventMaxCapacity},
			},
		}
	}

	tests := []struct {
		name    string
		update  func(*OrganizationLimits)
		wantErr bool
	}{
		{
			name: "accepts maximum values",
			update: func(limits *OrganizationLimits) {
				limits.Rates.OrganizationSpecific = RateLimit{RatePerMinute: maxRequestRatePerMinute, MaxCapacity: maxRequestMaxCapacity}
				limits.Rates.WorkspaceSpecific = RateLimit{RatePerMinute: maxRequestRatePerMinute, MaxCapacity: maxRequestMaxCapacity}
				limits.Rates.EventsSpecific = RateLimit{RatePerMinute: maxEventRatePerMinute, MaxCapacity: maxEventMaxCapacity}
			},
		},
		{
			name: "rejects organization request maximum capacity above the allowed range",
			update: func(limits *OrganizationLimits) {
				limits.Rates.OrganizationSpecific.MaxCapacity = maxRequestMaxCapacity + 1
			},
			wantErr: true,
		},
		{
			name: "rejects workspace request rate below minimum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.WorkspaceSpecific.RatePerMinute = minRequestRatePerMinute - 1
			},
			wantErr: true,
		},
		{
			name: "rejects workspace request rate above maximum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.WorkspaceSpecific.RatePerMinute = maxRequestRatePerMinute + 1
			},
			wantErr: true,
		},
		{
			name: "rejects workspace request maximum capacity below the allowed range",
			update: func(limits *OrganizationLimits) {
				limits.Rates.WorkspaceSpecific.MaxCapacity = minRequestMaxCapacity - 1
			},
			wantErr: true,
		},
		{
			name: "rejects event rate below minimum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.EventsSpecific.RatePerMinute = minEventRatePerMinute - 1
			},
			wantErr: true,
		},
		{
			name: "rejects event rate above maximum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.EventsSpecific.RatePerMinute = maxEventRatePerMinute + 1
			},
			wantErr: true,
		},
		{
			name: "rejects event maximum capacity above the allowed range",
			update: func(limits *OrganizationLimits) {
				limits.Rates.EventsSpecific.MaxCapacity = maxEventMaxCapacity + 1
			},
			wantErr: true,
		},
		{
			name: "rejects event maximum capacity below the allowed range",
			update: func(limits *OrganizationLimits) {
				limits.Rates.EventsSpecific.MaxCapacity = minEventMaxCapacity - 1
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limits := valid()
			test.update(&limits)
			err := validateOrganizationLimits(&limits)
			if (err != nil) != test.wantErr {
				t.Fatalf("expected error %t, got %v", test.wantErr, err)
			}
		})
	}
}
