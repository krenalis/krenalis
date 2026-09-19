// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestSyntheticPhotosConfiguration(t *testing.T) {

	t.Setenv("KRENALIS_KMS", "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("KRENALIS_DB_USERNAME", "u")
	t.Setenv("KRENALIS_DB_PASSWORD", "p")
	t.Setenv("KRENALIS_DB_DATABASE", "db")
	t.Setenv("KRENALIS_SYNTHETIC_PHOTOS_DIR", "")
	t.Setenv("KRENALIS_SYNTHETIC_CONFIG", "")
	disabled, err := loadConfig(t.Context(), "env:")
	if err != nil {
		t.Fatalf("expected disabled photo configuration, got %v", err)
	}
	if disabled.SyntheticPhotosDir != "" {
		t.Fatalf("expected disabled photo directory, got %q", disabled.SyntheticPhotosDir)
	}

	directory := t.TempDir()
	t.Setenv("KRENALIS_SYNTHETIC_PHOTOS_DIR", directory)
	enabled, err := loadConfig(t.Context(), "env:")
	if err != nil {
		t.Fatalf("expected enabled photo configuration, got %v", err)
	}
	if enabled.SyntheticPhotosDir != directory {
		t.Fatalf("expected photo directory %q, got %q", directory, enabled.SyntheticPhotosDir)
	}
	err = Run(t.Context(), enabled, nil, false, false)
	if err != nil {
		if !strings.Contains(err.Error(), "load KRENALIS_SYNTHETIC_PHOTOS_DIR") {
			t.Fatalf("expected clear startup error for invalid catalog, got %v", err)
		}
		return
	}
	t.Fatal("expected startup error for invalid catalog, got nil")

}

func TestSyntheticScenarioConfiguration(t *testing.T) {
	t.Setenv("KRENALIS_KMS", "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("KRENALIS_DB_USERNAME", "u")
	t.Setenv("KRENALIS_DB_PASSWORD", "p")
	t.Setenv("KRENALIS_DB_DATABASE", "db")
	for _, value := range []string{"null", "{", strings.Repeat("x", 8193)} {
		t.Setenv("KRENALIS_SYNTHETIC_CONFIG", value)
		_, err := loadConfig(t.Context(), "env:")
		if err == nil {
			t.Fatalf("expected rejected scenario %q, got nil", value[:min(len(value), 20)])
		}
	}
	t.Setenv("KRENALIS_SYNTHETIC_CONFIG", `{}`)
	conf, err := loadConfig(t.Context(), "env:")
	if err != nil || conf.Synthetic == nil {
		t.Fatalf("expected parsed scenario, got %v, %v", conf, err)
	}
	err = Run(t.Context(), conf, nil, false, false)
	if err == nil || !strings.Contains(err.Error(), "KRENALIS_SYNTHETIC_PHOTOS_DIR") {
		t.Fatalf("expected missing catalog startup error, got %v", err)
	}
}
