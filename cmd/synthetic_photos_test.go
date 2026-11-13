// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestExternalPhotoOriginsConfiguration(t *testing.T) {

	t.Setenv("KRENALIS_KMS", "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("KRENALIS_DB_USERNAME", "u")
	t.Setenv("KRENALIS_DB_PASSWORD", "p")
	t.Setenv("KRENALIS_DB_DATABASE", "db")
	t.Setenv("KRENALIS_SYNTHETIC_PHOTOS_DIR", "")
	for _, setting := range []string{"", " , ", "https://example.test/assets, http://internal-assets:8080/"} {
		t.Setenv("KRENALIS_EXTERNAL_ASSETS_URLS", setting)
		config, err := loadConfig(t.Context(), "env:")
		if err != nil {
			t.Fatalf("expected asset configuration, got %v", err)
		}
		want := "https://assets.krenalis.com/"
		if strings.HasPrefix(setting, "https://example.test") {
			want = "https://example.test/assets,http://internal-assets:8080/"
		}
		if got := strings.Join(config.ExternalAssetsURLs, ","); got != want {
			t.Fatalf("expected origins %q, got %q", want, got)
		}
		if config.SyntheticPhotosDir != "" {
			t.Fatalf("expected external metadata configuration without local override, got %q", config.SyntheticPhotosDir)
		}
	}

}

func TestSyntheticPhotosConfiguration(t *testing.T) {

	t.Setenv("KRENALIS_KMS", "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("KRENALIS_DB_USERNAME", "u")
	t.Setenv("KRENALIS_DB_PASSWORD", "p")
	t.Setenv("KRENALIS_DB_DATABASE", "db")
	t.Setenv("KRENALIS_SYNTHETIC_PHOTOS_DIR", "")
	disabled, err := loadConfig(t.Context(), "env:")
	if err != nil {
		t.Fatalf("expected configuration without local photo override, got %v", err)
	}
	if disabled.SyntheticPhotosDir != "" {
		t.Fatalf("expected unset local photo directory, got %q", disabled.SyntheticPhotosDir)
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

func TestUSARepositoryOverrideRejected(t *testing.T) {
	t.Setenv("KRENALIS_USA_INTERNAL_DEMO_DIR", t.TempDir())
	_, err := loadConfig(t.Context(), "env:")
	if err != nil {
		if !strings.Contains(err.Error(), "KRENALIS_USA_INTERNAL_DEMO_DIR is no longer supported") {
			t.Fatalf("expected explicit legacy configuration error, got %v", err)
		}
		return
	}
	t.Fatal("expected legacy repository override rejected during configuration, got success")
}
