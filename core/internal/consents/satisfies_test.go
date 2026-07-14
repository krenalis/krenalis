// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import "testing"

func TestSatisfies(t *testing.T) {
	cases := []struct {
		name       string
		required   []string
		matchAll   bool
		attributes map[string]any
		want       bool
	}{
		{
			name:       "no required codes",
			required:   nil,
			matchAll:   true,
			attributes: map[string]any{},
			want:       true,
		},
		{
			name:     "AND: all required codes are true",
			required: []string{"marketing", "analytics"},
			matchAll: true,
			attributes: map[string]any{
				"context": map[string]any{
					"consent": map[string]any{
						"marketing": true,
						"analytics": true,
						"other":     false,
					},
				},
			},
			want: true,
		},
		{
			name:     "AND: one required code is false",
			required: []string{"marketing", "analytics"},
			matchAll: true,
			attributes: map[string]any{
				"context": map[string]any{
					"consent": map[string]any{
						"marketing": true,
						"analytics": false,
					},
				},
			},
			want: false,
		},
		{
			name:     "AND: one required code is missing",
			required: []string{"marketing", "analytics"},
			matchAll: true,
			attributes: map[string]any{
				"context": map[string]any{
					"consent": map[string]any{
						"marketing": true,
					},
				},
			},
			want: false,
		},
		{
			name:     "AND: required code is not a bool",
			required: []string{"marketing"},
			matchAll: true,
			attributes: map[string]any{
				"context": map[string]any{
					"consent": map[string]any{
						"marketing": "true",
					},
				},
			},
			want: false,
		},
		{
			name:       "AND: missing context",
			required:   []string{"marketing"},
			matchAll:   true,
			attributes: map[string]any{},
			want:       false,
		},
		{
			name:     "AND: missing consent",
			required: []string{"marketing"},
			matchAll: true,
			attributes: map[string]any{
				"context": map[string]any{},
			},
			want: false,
		},
		{
			name:     "OR: all required codes are true",
			required: []string{"marketing", "analytics"},
			matchAll: false,
			attributes: map[string]any{
				"context": map[string]any{
					"consent": map[string]any{
						"marketing": true,
						"analytics": true,
					},
				},
			},
			want: true,
		},
		{
			name:     "OR: one required code is true",
			required: []string{"marketing", "analytics"},
			matchAll: false,
			attributes: map[string]any{
				"context": map[string]any{
					"consent": map[string]any{
						"marketing": false,
						"analytics": true,
					},
				},
			},
			want: true,
		},
		{
			name:     "OR: no required code is true",
			required: []string{"marketing", "analytics"},
			matchAll: false,
			attributes: map[string]any{
				"context": map[string]any{
					"consent": map[string]any{
						"marketing": false,
						"analytics": false,
					},
				},
			},
			want: false,
		},
		{
			name:       "OR: missing context",
			required:   []string{"marketing"},
			matchAll:   false,
			attributes: map[string]any{},
			want:       false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Satisfies(c.required, c.matchAll, c.attributes)
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}
