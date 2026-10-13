// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestItalianWorldSourceAllOriginsFail also rejects physical SHA-256 pins.
func TestItalianWorldSourceAllOriginsFail(t *testing.T) {

	assets := internationalAssetFixture(t)
	for _, physical := range []bool{false, true} {
		t.Run(fmt.Sprintf("physical=%t", physical), func(t *testing.T) {

			var requests atomic.Int32
			descriptor := assets[ItalianReleasePath]
			if physical {
				release, err := decodeCountryReleaseData(t.Context(), bytes.NewReader(descriptor))
				if err != nil {
					t.Fatalf("expected fixture descriptor, got %v", err)
				}
				release.document.Base.LogicalSHA256 = fmt.Sprintf("%x", sha256.Sum256(assets["base.json"]))
				descriptor = internationalTestJSON(t, release.document)
			}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if !physical {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if r.URL.Path == "/"+ItalianReleasePath {
					_, _ = w.Write(descriptor)
					return
				}
				_, _ = w.Write(assets[strings.TrimPrefix(r.URL.Path, "/")])
			})
			first, second := httptest.NewServer(handler), httptest.NewServer(handler)
			defer first.Close()
			defer second.Close()
			source := mustItalianWorldSource(t, first.URL, second.URL)
			if requests.Load() != 0 {
				t.Fatal("expected lazy source construction, got startup acquisition")
			}
			for range 2 {
				data, err := source.Load(t.Context())
				if err != nil {
					if data != nil || source.prepared != nil || source.documents.base != nil {
						t.Fatal("expected no result after exhausted origins, got partial success")
					}
					if physical && !strings.Contains(err.Error(), "base logical digest mismatch") {
						t.Fatalf("expected physical hash rejected as logical pin, got %v", err)
					}
					continue
				}
				t.Fatal("expected all origins rejected, got success")
			}
			want := int32(4)
			if physical {
				want = 5 // one retained descriptor, two attempts per base retry
			}
			if requests.Load() != want {
				t.Fatalf("expected %d attempts without failure caching, got %d", want, requests.Load())
			}

		})
	}

}

// TestItalianWorldSourceCancellation checks owners and canceled waiters over
// real HTTP; channel handshakes avoid timer-dependent polling.
func TestItalianWorldSourceCancellation(t *testing.T) {

	assets := internationalAssetFixture(t)
	blocked := make(chan struct{})
	release := make(chan struct{})
	var first atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/base.json" && first.CompareAndSwap(false, true) {
			close(blocked)
			select {
			case <-r.Context().Done():
				return
			case <-release:
			}
		}
		_, _ = w.Write(assets[strings.TrimPrefix(r.URL.Path, "/")])
	}))
	defer server.Close()
	defer close(release)
	source := mustItalianWorldSource(t, server.URL)
	owner, cancelOwner := context.WithCancel(t.Context())
	defer cancelOwner()
	done := make(chan error, 1)
	go func() {
		_, err := source.Load(owner)
		done <- err
	}()
	<-blocked
	waiter, cancelWaiter := context.WithCancel(t.Context())
	cancelWaiter()
	_, err := source.Load(waiter)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected waiter cancellation, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected canceled waiter error, got success")
	}
	source.mu.Lock()
	if source.prepared != nil {
		t.Error("expected no incomplete prepared result, got published data")
	}
	source.mu.Unlock()
	cancelOwner()
	err = <-done
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected owner cancellation, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected canceled owner error, got success")
	}
	_, err = source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected retry after owner cancellation, got %v", err)
	}

}

// TestItalianWorldSourceFallback uses controlled HTTP fixtures for all roles.
func TestItalianWorldSourceFallback(t *testing.T) {

	assets := internationalAssetFixture(t)
	for _, test := range []struct{ name, role, fault string }{
		{"first valid", "", ""},
		{"unreachable first", "", "unreachable"},
		{"descriptor malformed", ItalianReleasePath, "malformed"},
		{"descriptor country", ItalianReleasePath, "profile"},
		{"base transport", "base.json", "transport"},
		{"base digest", "base.json", "digest"},
		{"base malformed", "base.json", "malformed"},
		{"base profile", "base.json", "profile"},
		{"model digest", "socioeconomic.json", "digest"},
		{"model malformed", "socioeconomic.json", "malformed"},
		{"model profile", "socioeconomic.json", "profile"},
		{"photos digest", "photos.json", "digest"},
		{"photos malformed", "photos.json", "malformed"},
	} {
		t.Run(test.name, func(t *testing.T) {

			var firstRequests, secondRequests atomic.Int32
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				firstRequests.Add(1)
				path := strings.TrimPrefix(r.URL.Path, "/")
				body := assets[path]
				if path == test.role {
					switch test.fault {
					case "transport":
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					case "digest":
						body = bytes.Replace(body, []byte(`"release":"`), []byte(`"release":"changed-`), 1)
					case "malformed":
						body = []byte(`{"schema":`)
					case "profile":
						body = bytes.Replace(body, []byte(`"country":"IT"`), []byte(`"country":"FR"`), 1)
					}
				}
				_, _ = w.Write(body)
			}))
			defer first.Close()
			if test.fault == "unreachable" {
				first.Close()
			}
			second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				secondRequests.Add(1)
				_, _ = w.Write(assets[strings.TrimPrefix(r.URL.Path, "/")])
			}))
			defer second.Close()

			source := mustItalianWorldSource(t, first.URL, second.URL)
			data, err := source.Load(t.Context())
			if err != nil {
				t.Fatalf("expected resolved combination, got %v", err)
			}
			wantSecond := int32(1)
			if test.fault == "" {
				wantSecond = 0
			}
			if test.fault == "unreachable" {
				wantSecond = 4
			}
			if secondRequests.Load() != wantSecond {
				t.Fatalf("expected %d fallback requests, got %d", wantSecond, secondRequests.Load())
			}
			wantFirst := int32(4)
			if test.fault == "unreachable" {
				wantFirst = 0
			}
			if firstRequests.Load() != wantFirst {
				t.Fatalf("expected %d first-origin JSON requests, got %d", wantFirst, firstRequests.Load())
			}
			photoURL, err := source.PhotoURL(data.catalog, "female-a", PhotoSize256)
			if err != nil {
				t.Fatalf("expected declared photo URL, got %v", err)
			}
			if photoURL != first.URL+"/fixture/256/female-a.webp" {
				t.Fatalf("expected first-origin image URL, got %s", photoURL)
			}

		})
	}

}

// TestItalianWorldSourceFinalTransport rejects a complete JSON followed by a
// corrupt gzip trailer and never exposes it as a successful catalog.
func TestItalianWorldSourceFinalTransport(t *testing.T) {

	assets := internationalAssetFixture(t)
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write(assets["photos.json"])
	if err != nil {
		t.Fatalf("expected compressed fixture, got %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("expected gzip trailer, got %v", err)
	}
	corrupt := bytes.Clone(compressed.Bytes())
	corrupt[len(corrupt)-8] ^= 0xff
	var fail atomic.Bool
	fail.Store(true)
	started := make(chan struct{})
	finish := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/photos.json" && fail.Load() {
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = w.Write(corrupt[:len(corrupt)-8])
			w.(http.Flusher).Flush()
			close(started)
			<-finish
			_, _ = w.Write(corrupt[len(corrupt)-8:])
			return
		}
		_, _ = w.Write(assets[strings.TrimPrefix(r.URL.Path, "/")])
	}))
	defer server.Close()
	source := mustItalianWorldSource(t, server.URL)
	done := make(chan error, 1)
	go func() {
		_, err := source.Load(t.Context())
		done <- err
	}()
	<-started
	source.mu.Lock()
	if source.prepared != nil || source.documents.photos != nil {
		t.Error("expected no success before transport completion, got published catalog")
	}
	source.mu.Unlock()
	close(finish)
	err = <-done
	if err != nil {
		if source.prepared != nil || source.documents.photos != nil {
			t.Fatal("expected corrupt-trailer candidate discarded, got cached catalog")
		}
	}
	if err == nil {
		t.Fatal("expected corrupt trailer rejection, got success")
	}
	fail.Store(false)
	_, err = source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected successful transport retry, got %v", err)
	}

}

// TestItalianWorldSourceIncompatiblePhotos keeps successful base/model data
// when an autonomously valid photo catalog cannot serve the Italian profile.
func TestItalianWorldSourceIncompatiblePhotos(t *testing.T) {

	assets := internationalAssetFixture(t)
	document := internationalPhotoFixture()
	document.Faces = document.Faces[:2] // valid format, but no male candidate
	photos := mustInternationalPhotos(t, document)
	assets["photos.json"] = internationalTestJSON(t, photos.document)
	release, err := decodeCountryReleaseData(t.Context(), bytes.NewReader(assets[ItalianReleasePath]))
	if err != nil {
		t.Fatalf("expected fixture descriptor, got %v", err)
	}
	release.document.Photos.LogicalSHA256 = fmt.Sprintf("%x", photos.digest)
	assets[ItalianReleasePath] = internationalTestJSON(t, release.document)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(assets[strings.TrimPrefix(r.URL.Path, "/")])
	}))
	defer server.Close()
	source := mustItalianWorldSource(t, server.URL)
	var model *SocioeconomicModel
	for range 2 {
		data, err := source.Load(t.Context())
		if err != nil {
			if data != nil || source.prepared != nil || source.documents.base == nil ||
				source.documents.socioeconomic == nil || !strings.Contains(err.Error(), "gender male") {
				t.Fatalf("expected photo compatibility failure with autonomous successes retained, got %v", err)
			}
			if model != nil && source.documents.socioeconomic.model != model {
				t.Fatal("expected retained autonomous model after incompatible combination, got replacement")
			}
			model = source.documents.socioeconomic.model
			continue
		}
		t.Fatal("expected incompatible combination rejected, got success")
	}
	_, err = PrepareItalianWorldData(source.documents.base, source.documents.socioeconomic, photos)
	if err != nil {
		if !errors.Is(err, ErrNoCompatiblePhoto) {
			t.Fatalf("expected local composition rejection, got %v", err)
		}
		return
	}

	t.Fatal("expected local composition to enforce photo coverage, got success")

}

// TestItalianWorldSourceMixedOrigins checks fallback per document and caching.
func TestItalianWorldSourceMixedOrigins(t *testing.T) {

	assets := internationalAssetFixture(t)
	var requests atomic.Int32
	servers := make([]*httptest.Server, 3)
	for i := range servers {
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			path := strings.TrimPrefix(r.URL.Path, "/")
			allowed := i == 0 && (path == ItalianReleasePath || path == "base.json") ||
				i == 1 && path == "socioeconomic.json" || i == 2 && path == "photos.json"
			if !allowed {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write(assets[path])
		}))
		defer servers[i].Close()
	}
	source := mustItalianWorldSource(t, servers[0].URL, servers[1].URL, servers[2].URL)
	data, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected mixed-origin exact documents, got %v", err)
	}
	if requests.Load() != 7 {
		t.Fatalf("expected seven ordered attempts, got %d", requests.Load())
	}
	for _, server := range servers {
		server.Close()
	}
	again, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected cached success with origins closed, got %v", err)
	}
	if again != data || requests.Load() != 7 {
		t.Fatal("expected immutable cached combination without requests, got reload")
	}

}

// TestItalianWorldSourceReferenceScope checks immutable owner bindings when
// the same paths are used for different expected logical contents.
func TestItalianWorldSourceReferenceScope(t *testing.T) {

	assets := internationalAssetFixture(t)
	changed := maps.Clone(assets)
	changed["photos.json"] = bytes.Replace(assets["photos.json"], []byte("fixture/256/"), []byte("changed/256/"), 1)
	photo, err := DecodePhotoCatalogData(t.Context(), bytes.NewReader(changed["photos.json"]))
	if err != nil {
		t.Fatalf("expected changed autonomous photo catalog, got %v", err)
	}
	old, err := DecodePhotoCatalogData(t.Context(), bytes.NewReader(assets["photos.json"]))
	if err != nil {
		t.Fatalf("expected original photo fixture, got %v", err)
	}
	changed[ItalianReleasePath] = bytes.Replace(assets[ItalianReleasePath],
		[]byte(fmt.Sprintf("%x", old.digest)), []byte(fmt.Sprintf("%x", photo.digest)), 1)
	var useChanged atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		documents := assets
		if useChanged.Load() {
			documents = changed
		}
		_, _ = w.Write(documents[strings.TrimPrefix(r.URL.Path, "/")])
	}))
	defer server.Close()
	source := mustItalianWorldSource(t, server.URL)
	first, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected original combination, got %v", err)
	}
	useChanged.Store(true)
	other := mustItalianWorldSource(t, server.URL)
	second, err := other.Load(t.Context())
	if err != nil {
		t.Fatalf("expected newly pinned combination, got %v", err)
	}
	again, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected retained first combination, got %v", err)
	}
	if first != again || second.photos.digest == first.photos.digest || second.packageDigest == first.packageDigest {
		t.Fatal("expected isolated reference digests and pinned owner success, got stale or replaced data")
	}

}

// TestItalianWorldSourceRetryConcurrent retains valid documents after failure.
func TestItalianWorldSourceRetryConcurrent(t *testing.T) {

	assets := internationalAssetFixture(t)
	var fail atomic.Bool
	fail.Store(true)
	var mu sync.Mutex
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		mu.Lock()
		requests[path]++
		mu.Unlock()
		if path == "photos.json" && fail.Load() {
			_, _ = w.Write([]byte(`{"invalid":0}`))
			return
		}
		_, _ = w.Write(assets[path])
	}))
	defer server.Close()
	source := mustItalianWorldSource(t, server.URL)
	for range 2 {
		data, err := source.Load(t.Context())
		if err != nil {
			if data != nil || source.prepared != nil || source.documents.photos != nil ||
				source.documents.base == nil || source.documents.socioeconomic == nil {
				t.Fatal("expected only autonomous successes retained, got partial combination or discarded success")
			}
			continue
		}
		t.Fatal("expected failed photo acquisition, got success")
	}
	model := source.documents.socioeconomic.model
	fail.Store(false)
	const count = 16
	results := make(chan *ItalianWorldData, count)
	errors := make(chan error, count)
	var group sync.WaitGroup
	for range count {
		group.Go(func() {
			data, err := source.Load(t.Context())
			results <- data
			errors <- err
		})
	}
	group.Wait()
	close(results)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("expected concurrent retry success, got %v", err)
		}
	}
	for data := range results {
		if data != source.prepared || data.socioeconomic.model != model {
			t.Fatal("expected one shared immutable combination and model, got divergent ownership")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	for path, count := range requests {
		want := 1
		if path == "photos.json" {
			want = 3
		}
		if count != want {
			t.Fatalf("expected %d requests for %s, got %d", want, path, count)
		}
	}

}

func internationalAssetFixture(t *testing.T) map[string][]byte {

	t.Helper()
	// Only offline fixture preparation reads original CSVs; no fixture is a
	// distributable release.
	base, err := ConvertItalianBaseData(t.Context(), os.DirFS("data/it-2026a"))
	if err != nil {
		t.Fatalf("expected base fixture, got %v", err)
	}
	photos := mustInternationalPhotos(t, internationalPhotoFixture())
	assets := map[string][]byte{
		"base.json":          internationalTestJSON(t, base.document),
		"socioeconomic.json": internationalTestJSON(t, internationalSocioeconomicFixture(t)),
		"photos.json":        internationalTestJSON(t, photos.document),
	}
	baseDigest, err := base.LogicalDigest()
	if err != nil {
		t.Fatalf("expected base fixture digest, got %v", err)
	}
	modelDigest, err := internationalDigest("fakedata/socioeconomic-data/json-v1", internationalSocioeconomicFixture(t))
	if err != nil {
		t.Fatalf("expected model fixture digest, got %v", err)
	}
	assets[ItalianReleasePath] = internationalTestJSON(t, internationalRelease{
		Schema: "sw-country-release-v1", Release: "artificial-combination", Country: "IT",
		Base:          internationalAssetReference{"base.json", fmt.Sprintf("%x", baseDigest)},
		Socioeconomic: internationalAssetReference{"socioeconomic.json", fmt.Sprintf("%x", modelDigest)},
		Photos:        internationalAssetReference{"photos.json", fmt.Sprintf("%x", photos.digest)},
	})

	return assets
}

func internationalTestJSON(t *testing.T, document any) []byte {
	t.Helper()
	encoded, err := internationalCanonical(document)
	if err != nil {
		t.Fatalf("expected fixture C1, got %v", err)
	}
	return encoded
}

func mustItalianWorldSource(t *testing.T, origins ...string) *ItalianWorldSource {
	t.Helper()
	source, err := NewItalianWorldSource(origins)
	if err != nil {
		t.Fatalf("expected configured Italian source, got %v", err)
	}
	return source
}
