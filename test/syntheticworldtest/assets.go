// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package syntheticworldtest prepares portable international JSON test assets.
package syntheticworldtest

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
)

// Assets converts the frozen base offline and adds artificial L2/photo data.
// The fixtures contain no published model or image bytes.
func Assets(t *testing.T) map[string][]byte {

	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("expected fixture source location, got none")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "tools", "fakedata")
	base, err := fakedata.ConvertItalianBaseData(t.Context(), os.DirFS(filepath.Join(root, "data", "it-2026a")))
	if err != nil {
		t.Fatalf("expected offline base conversion, got %v", err)
	}
	baseJSON, err := base.CanonicalJSON()
	if err != nil {
		t.Fatalf("expected base JSON, got %v", err)
	}
	assets := map[string][]byte{"base.json": baseJSON}
	for role, name := range map[string]string{
		"socioeconomic.json": "international-socioeconomic-v1.json", "photos.json": "international-photos-v1.json",
	} {
		assets[role], err = os.ReadFile(filepath.Join(root, "testdata", name))
		if err != nil {
			t.Fatalf("expected portable %s fixture, got %v", role, err)
		}
	}
	setModel(t, assets, assets["socioeconomic.json"])

	return assets
}

// Handler serves only the declared JSON assets below prefix, rejecting images.
func Handler(t *testing.T, assets map[string][]byte, prefix string) http.Handler {
	return handler(t, assets, prefix)
}

// Load acquires the fixtures through HTTP, then closes their origin.
func Load(t *testing.T, assets map[string][]byte) *fakedata.ItalianWorldData {

	t.Helper()
	server := httptest.NewServer(handler(t, assets, ""))
	defer server.Close()
	source, err := fakedata.NewItalianWorldSource([]string{server.URL})
	if err != nil {
		t.Fatalf("expected Italian World source, got %v", err)
	}
	data, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected prepared Italian World, got %v", err)
	}

	return data
}

// SetModel replaces an artificial model and rebuilds the coordinated descriptor.
func SetModel(t *testing.T, assets map[string][]byte, model []byte) {
	setModel(t, assets, model)
}

func handler(t *testing.T, assets map[string][]byte, prefix string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := assets[strings.TrimPrefix(r.URL.Path, prefix+"/")]
		if !ok {
			t.Errorf("expected declared JSON request, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(data)
	})
}

func setModel(t *testing.T, assets map[string][]byte, model []byte) {

	t.Helper()
	base, err := fakedata.DecodeItalianBaseData(t.Context(), bytes.NewReader(assets["base.json"]))
	if err != nil {
		t.Fatalf("expected base document, got %v", err)
	}
	baseDigest, err := base.LogicalDigest()
	if err != nil {
		t.Fatalf("expected base digest, got %v", err)
	}
	socioeconomic, err := fakedata.DecodeItalianSocioeconomicData(t.Context(), bytes.NewReader(model))
	if err != nil {
		t.Fatalf("expected model document, got %v", err)
	}
	modelDigest, err := socioeconomic.LogicalDigest()
	if err != nil {
		t.Fatalf("expected model digest, got %v", err)
	}
	photos, err := fakedata.DecodePhotoCatalogData(t.Context(), bytes.NewReader(assets["photos.json"]))
	if err != nil {
		t.Fatalf("expected photo document, got %v", err)
	}
	photoDigest, err := photos.LogicalDigest()
	if err != nil {
		t.Fatalf("expected photo digest, got %v", err)
	}
	references := map[string]any{"schema": "sw-country-release-v1", "release": "artificial-test", "country": "IT"}
	for role, digest := range map[string][32]byte{"base": baseDigest, "socioeconomic": modelDigest, "photos": photoDigest} {
		references[role] = map[string]string{"path": role + ".json", "logicalSha256": fmt.Sprintf("%x", digest)}
	}
	descriptor, err := json.Marshal(references)
	if err != nil {
		t.Fatalf("expected descriptor JSON, got %v", err)
	}
	assets[fakedata.ItalianReleasePath] = descriptor
	assets["socioeconomic.json"] = model

}
