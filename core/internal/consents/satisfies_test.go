// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import "testing"

func TestSatisfies(t *testing.T) {
	cases := []struct {
		name       string
		required   []string
		attributes map[string]any
		want       bool
	}{
		{
			name:       "no required codes",
			required:   nil,
			attributes: map[string]any{},
			want:       true,
		},
		{
			name:     "all required codes are true",
			required: []string{"marketing", "analytics"},
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
			name:     "one required code is false",
			required: []string{"marketing", "analytics"},
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
			name:     "one required code is missing",
			required: []string{"marketing", "analytics"},
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
			name:     "required code is not a bool",
			required: []string{"marketing"},
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
			name:       "missing context",
			required:   []string{"marketing"},
			attributes: map[string]any{},
			want:       false,
		},
		{
			name:     "missing consent",
			required: []string{"marketing"},
			attributes: map[string]any{
				"context": map[string]any{},
			},
			want: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Satisfies(c.required, c.attributes)
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}
