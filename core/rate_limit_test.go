// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import "testing"

// TestValidateOrganizationRateLimits verifies the accepted boundaries and
// rejects values outside the configured organization rate-limit ranges.
func TestValidateOrganizationRateLimits(t *testing.T) {
	valid := func() OrganizationLimits {
		return OrganizationLimits{
			Members: 1,
			Rates: RateLimits{
				Workspace:    RateLimit{RatePerMinute: minRequestRatePerMinute, BurstCapacity: minRequestBurstCapacity},
				Events:       RateLimit{RatePerMinute: minEventRatePerMinute, BurstCapacity: minEventBurstCapacity},
				Organization: RateLimit{RatePerMinute: minRequestRatePerMinute, BurstCapacity: minRequestBurstCapacity},
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
				limits.Rates.Workspace = RateLimit{RatePerMinute: maxRequestRatePerMinute, BurstCapacity: maxRequestBurstCapacity}
				limits.Rates.Events = RateLimit{RatePerMinute: maxEventRatePerMinute, BurstCapacity: maxEventBurstCapacity}
				limits.Rates.Organization = RateLimit{RatePerMinute: maxRequestRatePerMinute, BurstCapacity: maxRequestBurstCapacity}
			},
		},
		{
			name: "rejects workspace request rate below minimum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.Workspace.RatePerMinute = minRequestRatePerMinute - 1
			},
			wantErr: true,
		},
		{
			name: "rejects workspace request rate above maximum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.Workspace.RatePerMinute = maxRequestRatePerMinute + 1
			},
			wantErr: true,
		},
		{
			name: "rejects workspace request burst below minimum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.Workspace.BurstCapacity = minRequestBurstCapacity - 1
			},
			wantErr: true,
		},
		{
			name: "rejects event rate below minimum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.Events.RatePerMinute = minEventRatePerMinute - 1
			},
			wantErr: true,
		},
		{
			name: "rejects event rate above maximum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.Events.RatePerMinute = maxEventRatePerMinute + 1
			},
			wantErr: true,
		},
		{
			name: "rejects organization request burst above maximum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.Organization.BurstCapacity = maxRequestBurstCapacity + 1
			},
			wantErr: true,
		},
		{
			name: "rejects event burst above maximum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.Events.BurstCapacity = maxEventBurstCapacity + 1
			},
			wantErr: true,
		},
		{
			name: "rejects event burst below minimum",
			update: func(limits *OrganizationLimits) {
				limits.Rates.Events.BurstCapacity = minEventBurstCapacity - 1
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
