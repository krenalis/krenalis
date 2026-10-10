// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/krenalis/krenalis/tools/json"
)

// TestSocioeconomicModel verifies compilation without generating any facts.
func TestSocioeconomicModel(t *testing.T) {

	model, err := decodeSocioeconomicModel(strings.NewReader(socioeconomicFixture(t)))
	if err != nil {
		t.Fatalf("expected compiled model, got %v", err)
	}
	if model.companyCount != 65536 || model.employer.cap != 4096 || len(model.employeeCount) != 14 {
		t.Fatalf("expected company parameters, got count %d and cap %d", model.companyCount, model.employer.cap)
	}
	if !slices.Equal(model.companyIndustry.cumulative, []uint64{1, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3}) {
		t.Fatalf("expected exact cumulative weights with zero slots, got %v", model.companyIndustry.cumulative)
	}
	for i, age := range model.ages {
		if age != [55]uint8{} {
			t.Fatalf("expected one complete age bucket in mapping %d, got %v", i, age)
		}
	}
	for branch, income := range model.income {

		law := income.laws[0]
		if !slices.Equal(law.values, []uint32{0, 1, math.MaxUint32}) ||
			!slices.Equal(law.cumulative, []uint64{1, 4294967296, 4294967298}) {
			t.Fatalf("expected exact zero/positive/maximum supports and uint64 totals, got %+v", law)
		}

		for slot, id := range income.tupleMap {
			want := uint16(0)
			if branch == 0 && slot%5 == 0 {
				want = math.MaxUint16
			}
			if id != want {
				t.Fatalf("expected law %d in branch %d slot %d, got %d", want, branch, slot, id)
			}
		}

	}

	data, err := loadLayer1ItalyData()
	if err != nil {
		t.Fatalf("expected frozen Italian data, got %v", err)
	}
	// Region ordinals follow D2-A and must cover the frozen Layer 1 labels.
	regions := [...]string{
		"Abruzzo", "Basilicata", "Calabria", "Campania", "Emilia-Romagna", "Friuli-Venezia Giulia", "Lazio", "Liguria",
		"Lombardia", "Marche", "Molise", "Piemonte", "Puglia", "Sardegna", "Sicilia", "Toscana",
		"Trentino-Alto Adige/Südtirol", "Umbria", "Valle d'Aosta/Vallée d'Aoste", "Veneto",
	}
	seen := map[string]bool{}
	for _, place := range data.places {
		if !slices.Contains(regions[:], place.region) {
			t.Fatalf("expected a model Region for every frozen place, got %q", place.region)
		}
		seen[place.region] = true
	}
	if len(seen) != len(regions) {
		t.Fatalf("expected %d Regions, got %d", len(regions), len(seen))
	}

}

// TestSocioeconomicModelAgeBoundaries checks the supplied compiled mapping
// against every age, including the boundary and both adjacent ages.
func TestSocioeconomicModelAgeBoundaries(t *testing.T) {

	fixture := socioeconomicFixture(t)
	old := `"lowerBounds":[21],"compiled":` + socioeconomicNumbers(t, 55, 0)
	compiled := append(make([]uint32, 14), slices.Repeat([]uint32{1}, 41)...)
	encoded, err := json.Marshal(compiled)
	if err != nil {
		t.Fatalf("expected age JSON, got %v", err)
	}
	fixture = strings.Replace(fixture, old, `"lowerBounds":[21,35],"compiled":`+string(encoded), 1)
	oldWeights := `"education":{"weights":` + socioeconomicNumbers(t, 140, 1)
	fixture = strings.Replace(fixture, oldWeights, `"education":{"weights":`+socioeconomicNumbers(t, 280, 1), 1)

	model, err := decodeSocioeconomicModel(strings.NewReader(fixture))
	if err != nil {
		t.Fatalf("expected two-bucket model, got %v", err)
	}
	for offset, bucket := range model.ages[0] {
		want := uint8(0)
		if offset+21 >= 35 {
			want = 1
		}
		if bucket != want {
			t.Fatalf("expected bucket %d at age %d, got %d", want, offset+21, bucket)
		}
	}

}

// TestSocioeconomicModelBounds accepts the frozen ranges beyond today's asset.
func TestSocioeconomicModelBounds(t *testing.T) {

	input := strings.NewReplacer(
		`"companyCount":65536`, `"companyCount":104855`,
		`"cap":4096`, `"cap":65535`,
		`"industryFactors":[1`, `"industryFactors":[4095`,
		`"sizeFactors":[1`, `"sizeFactors":[4095`,
		`"values":[0,9]`, `"values":[0,4294967295]`,
	).Replace(socioeconomicFixture(t))

	model, err := decodeSocioeconomicModel(strings.NewReader(input))
	if err != nil {
		t.Fatalf("expected maximum numerical bounds accepted, got %v", err)
	}
	if model.companyCount != 104855 || model.employer.cap != math.MaxUint16 ||
		model.employer.industryFactors[0] != 4095 || model.employer.sizeFactors[0] != 4095 ||
		model.employeeCount[0].values[1] != math.MaxUint32 {
		t.Fatalf("expected maximum values retained exactly, got altered model parameters")
	}

}

// TestSocioeconomicModelIncomeLaws checks canonical pooling and support capacity.
func TestSocioeconomicModelIncomeLaws(t *testing.T) {

	valid := socioeconomicFixture(t)
	law := `{"values":[0,1,4294967295],"weights":[1,4294967295,2]}`
	second := `{"values":[0,2],"weights":[1,1]}`
	twoLaws := strings.Replace(valid, law, law+","+second, 1)
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{"duplicate", strings.Replace(valid, law, law+","+law, 1), "duplicate canonical laws"},
		{"unreferenced", twoLaws, "unreferenced law"},
		{"first occurrence",
			strings.Replace(twoLaws, `"tupleMap":[65535,0,0`, `"tupleMap":[65535,1,0`, 1), "noncanonical"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeSocioeconomicModel(strings.NewReader(test.input))
			if err != nil {
				if !strings.Contains(err.Error(), test.want) {
					t.Fatalf("expected %q rejection, got %v", test.want, err)
				}
				return
			}
			t.Fatalf("expected invalid law rejection, got success")
		})
	}
	input := strings.Replace(twoLaws, `"tupleMap":[65535,0,0`, `"tupleMap":[65535,0,1`, 1)
	model, err := decodeSocioeconomicModel(strings.NewReader(input))
	if err != nil {
		t.Fatalf("expected canonical first-occurrence references, got %v", err)
	}
	if model.income[0].tupleMap[2] != 1 || len(model.income[0].laws) != 2 {
		t.Fatalf("expected two laws and reference 1, got %d laws and reference %d",
			len(model.income[0].laws), model.income[0].tupleMap[2])
	}

	values := make([]uint32, 512)
	for i := range values {
		values[i] = uint32(i)
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("expected support JSON, got %v", err)
	}
	denseLaw := `{"values":` + string(encoded) + `,"weights":` + socioeconomicNumbers(t, 512, 1) + `}`
	model, err = decodeSocioeconomicModel(strings.NewReader(strings.Replace(valid, law, denseLaw, 1)))
	if err != nil {
		t.Fatalf("expected one 512-point law to fit, got %v", err)
	}
	if len(model.income[0].laws[0].values) != 512 {
		t.Fatalf("expected 512 retained points, got %d", len(model.income[0].laws[0].values))
	}

	// All simultaneous law/point maxima exceed the retained-byte envelope even
	// though their encoded input and individual cardinalities fit. Preflight
	// must reject that volume before compiling or comparing the law pools.
	parts := strings.Split(valid, law)
	var full strings.Builder
	for i, count := range []int{512, 256, 64, 64, 64, 64} {
		full.WriteString(parts[i])
		full.WriteString(strings.TrimSuffix(strings.Repeat(denseLaw+",", count), ","))
	}
	full.WriteString(parts[6])
	_, err = decodeSocioeconomicModel(strings.NewReader(full.String()))
	if err != nil {
		if !strings.Contains(err.Error(), "retained model charge") {
			t.Fatalf("expected aggregate retained-byte rejection, got %v", err)
		}
		return
	}

	t.Fatalf("expected aggregate retained-byte rejection, got success")

}

// TestSocioeconomicModelInvalid covers closed schema, numeric domains, complete
// grids, canonical laws and branch-local references through the real decoder.
func TestSocioeconomicModelInvalid(t *testing.T) {

	valid := socioeconomicFixture(t)
	cases := []struct {
		name string
		old  string
		new  string
		want string
	}{
		{"schema", `"SW-L2-D2B2-logical-v0"`, `"future"`, "schema"},
		{"unknown section", `"companyIndustry":`, `"other":`, "unknown field"},
		{"missing section", `"companyIndustry":{"weights":[1,2,0,0,0,0,0,0,0,0,0,0,0,0]},`,
			``, "missing field companyIndustry"},
		{"missing scalar", `"companyCount":65536,`, ``, "missing field companyCount"},
		{"duplicate key", `"companyCount":65536,`, `"companyCount":65536,"companyCount":65536,`, "JSON"},
		{"escaped duplicate key", `"companyCount":65536,`, `"companyCount":65536,"company\u0043ount":65536,`, "JSON"},
		{"null scalar", `"companyCount":65536`, `"companyCount":null`, "unsigned integer"},
		{"zero count", `"companyCount":65536`, `"companyCount":0`, "positive"},
		{"count bound", `"companyCount":65536`, `"companyCount":104856`, "exceeds"},
		{"decimal", `"companyCount":65536`, `"companyCount":65536.0`, "integer"},
		{"exponent", `"companyCount":65536`, `"companyCount":65536e0`, "integer"},
		{"negative", `"companyCount":65536`, `"companyCount":-1`, "integer"},
		{"uint64 overflow", `"companyCount":65536`, `"companyCount":18446744073709551616`, "integer"},
		{"boolean", `"companyCount":65536`, `"companyCount":true`, "integer"},
		{"unknown branch", `"i":{"tupleMap"`, `"x":{"tupleMap"`, "unknown field"},
		{"unknown age domain", `"Income[i]":`, `"Income[x]":`, "unknown field"},
		{"missing law field", `"values":[0,1,4294967295],`, ``, "missing field values"},
		{"empty law", `"values":[0,1,4294967295]`, `"values":[]`, "positive length"},
		{"null law", `"values":[0,1,4294967295]`, `"values":null`, "expected array"},
		{"support width", `"values":[0,1,4294967295]`, `"values":[0,1,4294967296]`, "exceeds"},
		{"weight width", `"weights":[1,4294967295,2]`, `"weights":[1,4294967296,2]`, "exceeds"},
		{"support order", `"values":[0,1,4294967295]`, `"values":[1,0,4294967295]`, "increasing"},
		{"duplicate support", `"values":[0,1,4294967295]`, `"values":[0,0,4294967295]`, "increasing"},
		{"length mismatch", `"values":[0,1,4294967295]`, `"values":[0,1]`, "equal positive length"},
		{"zero support weight", `"weights":[1,4294967295,2]`, `"weights":[0,4294967295,2]`, "zero weight"},
		{"GCD", `"weights":[1,4294967295,2]`, `"weights":[2,4,6]`, "GCD"},
		{"categorical GCD", `"companyIndustry":{"weights":[1,2`, `"companyIndustry":{"weights":[2,4`, "GCD"},
		{"no workforce support",
			`"employeeCount":{"laws":[{"values":[0,9],"weights":[1,1]},{"values":[0,9],"weights":[1,1]}`,
			`"employeeCount":{"laws":[{"values":[0],"weights":[1]},{"values":[0],"weights":[1]}`,
			"no positive-workforce"},
		{"cap zero", `"cap":4096`, `"cap":0`, "positive"},
		{"cap width", `"cap":4096`, `"cap":65536`, "exceeds"},
		{"factor zero", `"industryFactors":[1`, `"industryFactors":[0`, "positive"},
		{"factor range", `"sizeFactors":[1`, `"sizeFactors":[4096`, "exceeds"},
		{"age coverage", `"lowerBounds":[21]`, `"lowerBounds":[22]`, "coverage"},
		{"age order", `"lowerBounds":[21]`, `"lowerBounds":[21,21]`, "increasing"},
		{"age cap", `"lowerBounds":[21]`, `"lowerBounds":[21,22,23,24,25,26,27]`, "exceeds 6 entries"},
		{"compiled age", `"compiled":[0`, `"compiled":[1`, "inconsistent bucket"},
		{"sentinel", `"tupleMap":[65535`, `"tupleMap":[0`, "sentinel"},
		{"broken reference", `"tupleMap":[65535,0`, `"tupleMap":[65535,1`, "reference"},
		{"reachable sentinel", `"tupleMap":[65535,0`, `"tupleMap":[65535,65535`, "reference"},
		{"unknown nested field", `"cap":4096`, `"extra":0,"cap":4096`, "unknown field"},
	}
	for _, test := range cases {

		t.Run(test.name, func(t *testing.T) {

			input := strings.Replace(valid, test.old, test.new, 1)
			if input == valid {
				t.Fatalf("expected fixture mutation, got unchanged input")
			}
			model, err := decodeSocioeconomicModel(strings.NewReader(input))
			if err != nil {
				if model != nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("expected nil model and %q error, got model %p and %v", test.want, model, err)
				}
				return
			}

			t.Fatalf("expected rejected model, got success")

		})

	}
	for _, input := range []string{
		"{", "null", valid + "{}", valid + "!", strings.Replace(valid, `"income":`, `"income":null,"other":`, 1),
	} {
		_, err := decodeSocioeconomicModel(strings.NewReader(input))
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid JSON/schema rejection, got success")
	}

}

// TestSocioeconomicModelLimits exercises bounds before excessive array growth.
func TestSocioeconomicModelLimits(t *testing.T) {

	valid := socioeconomicFixture(t)
	countLaw := `{"values":[0,9],"weights":[1,1]}`
	t.Run("decoded byte boundary", func(t *testing.T) {

		var padded strings.Builder
		remaining := maxSocioeconomicModelBytes - len(valid)
		for _, b := range valid {
			padded.WriteRune(b)
			if b == ',' && remaining > 0 {
				count := min(192, remaining)
				padded.WriteString(strings.Repeat(" ", count))
				remaining -= count
			}
		}
		if padded.Len() != maxSocioeconomicModelBytes {
			t.Fatalf("expected %d-byte fixture, got %d", maxSocioeconomicModelBytes, padded.Len())
		}
		_, err := decodeSocioeconomicModel(strings.NewReader(padded.String()))
		if err != nil {
			t.Fatalf("expected exact byte limit accepted, got %v", err)
		}
		_, err = decodeSocioeconomicModel(io.MultiReader(strings.NewReader(padded.String()), strings.NewReader(" ")))
		if err != nil {
			if !strings.Contains(err.Error(), "decoded bytes") {
				t.Fatalf("expected decoded-byte rejection, got %v", err)
			}
			return
		}

		t.Fatalf("expected byte limit plus one rejected, got success")

	})
	cases := []struct {
		name  string
		input string
	}{
		{"bytes", valid + strings.Repeat(" ", maxSocioeconomicModelBytes)},
		{"whitespace", strings.Repeat(" ", 193) + valid},
		{"token", `{"` + strings.Repeat("x", 193) + `":0}`},
		{"categorical cardinality",
			strings.Replace(valid, `"companyIndustry":{"weights":[`, `"companyIndustry":{"weights":[1,`, 1)},
		{"count points", strings.Replace(valid, `"values":[0,9]`, `"values":`+socioeconomicNumbers(t, 129, 0), 1)},
		{"income points",
			strings.Replace(valid, `"values":[0,1,4294967295]`, `"values":`+socioeconomicNumbers(t, 513, 0), 1)},
		{"tuple cardinality", strings.Replace(valid, `"tupleMap":[65535`, `"tupleMap":[65535,65535`, 1)},
		{"law cardinality", strings.Replace(valid, countLaw, countLaw+","+countLaw, 1)},
		{"missing grid cell", strings.Replace(valid, `"education":{"weights":[1,`, `"education":{"weights":[`, 1)},
		{"zero vector",
			strings.Replace(valid, `"companyIndustry":{"weights":[1,2`, `"companyIndustry":{"weights":[0,0`, 1)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := decodeSocioeconomicModel(strings.NewReader(test.input))
			if err != nil {
				return
			}
			t.Fatalf("expected limit or completeness rejection, got success")
		})
	}

}

// TestSocioeconomicModelSourceCompression checks all supported encodings through
// model-specific consumption, rather than just the external resolver.
func TestSocioeconomicModelSourceCompression(t *testing.T) {

	fixture := socioeconomicFixture(t)
	for _, encoding := range []string{"identity", "gzip", "br"} {

		t.Run(encoding, func(t *testing.T) {

			var data bytes.Buffer
			switch encoding {
			case "identity":
				data.WriteString(fixture)
			case "gzip":
				writer := gzip.NewWriter(&data)
				_, _ = io.WriteString(writer, fixture)
				err := writer.Close()
				if err != nil {
					t.Fatalf("expected encoded fixture, got %v", err)
				}
			case "br":
				writer := brotli.NewWriter(&data)
				_, _ = io.WriteString(writer, fixture)
				err := writer.Close()
				if err != nil {
					t.Fatalf("expected encoded fixture, got %v", err)
				}
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Encoding", encoding)
				_, _ = w.Write(data.Bytes())
			}))
			defer server.Close()
			source, err := NewSocioeconomicModelSource([]string{server.URL})
			if err != nil {
				t.Fatalf("expected source, got %v", err)
			}
			_, err = source.Load(t.Context())
			if err != nil {
				t.Fatalf("expected compressed model, got %v", err)
			}

		})

	}

}

// TestSocioeconomicModelSourceFallback exercises complete attempts, including
// asset semantics, compressed input, path joining and successful retention.
func TestSocioeconomicModelSourceFallback(t *testing.T) {

	fixture := socioeconomicFixture(t)
	for _, failure := range []string{
		"HTTP", "transport", "JSON", "schema", "semantic", "size", "encoding", "gzip trailer",
	} {

		t.Run(failure, func(t *testing.T) {

			var firstCalls, secondCalls atomic.Int32
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				firstCalls.Add(1)
				switch failure {
				case "HTTP":
					w.WriteHeader(http.StatusServiceUnavailable)
				case "JSON":
					_, _ = io.WriteString(w, "{")
				case "schema":
					_, _ = io.WriteString(w, strings.Replace(fixture, "SW-L2-D2B2-logical-v0", "future", 1))
				case "semantic":
					_, _ = io.WriteString(w, strings.Replace(fixture, `"cap":4096`, `"cap":0`, 1))
				case "size":
					_, _ = io.WriteString(w, strings.ReplaceAll(fixture, ",", ","+strings.Repeat(" ", 150)))
				case "encoding":
					w.Header().Set("Content-Encoding", "unsupported")
					_, _ = io.WriteString(w, fixture)
				case "gzip trailer":
					var compressed bytes.Buffer
					writer := gzip.NewWriter(&compressed)
					_, _ = io.WriteString(writer, fixture)
					_ = writer.Close()
					data := compressed.Bytes()
					data[len(data)-8] ^= 1
					w.Header().Set("Content-Encoding", "gzip")
					_, _ = w.Write(data)
				}
			}))
			defer first.Close()
			if failure == "transport" {
				first.Close()
			}

			second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				secondCalls.Add(1)
				if r.URL.Path != "/assets/synthetic-world/it-2026a/model.json" {
					t.Errorf("expected production path with base prefix, got %q", r.URL.Path)
				}
				_, _ = io.WriteString(w, fixture)
			}))
			defer second.Close()
			source, err := NewSocioeconomicModelSource([]string{first.URL, second.URL + "/assets"})
			if err != nil {
				t.Fatalf("expected source, got %v", err)
			}
			model, err := source.Load(t.Context())
			if err != nil {
				t.Fatalf("expected fallback model, got %v", err)
			}
			again, err := source.Load(t.Context())
			if err != nil {
				t.Fatalf("expected retained model, got %v", err)
			}
			wantFirst := int32(1)
			if failure == "transport" {
				wantFirst = 0
			}
			if again != model || firstCalls.Load() != wantFirst || secondCalls.Load() != 1 {
				t.Fatalf("expected retained second-origin model, got pointers %p/%p and calls %d/%d",
					model, again, firstCalls.Load(), secondCalls.Load())
			}

		})

	}

}

// TestSocioeconomicModelSourceOwnerCancellationRetry checks cancellation of
// the active HTTP acquisition owner, successful retry and cached reuse.
func TestSocioeconomicModelSourceOwnerCancellationRetry(t *testing.T) {

	fixture := socioeconomicFixture(t)
	expected, err := decodeSocioeconomicModel(strings.NewReader(fixture))
	if err != nil {
		t.Fatalf("expected complete canonical fixture, got %v", err)
	}
	ctx, stop := context.WithTimeout(t.Context(), 10*time.Second)
	defer stop()
	started, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
			select {
			case <-r.Context().Done():
				close(canceled)
			case <-release:
			}
			return
		}
		_, _ = io.WriteString(w, fixture)
	}))
	defer server.Close()
	defer close(release)
	source, err := NewSocioeconomicModelSource([]string{server.URL})
	if err != nil {
		t.Fatalf("expected source, got %v", err)
	}

	ownerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var firstModel *SocioeconomicModel
	var firstErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		firstModel, firstErr = source.Load(ownerCtx)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("expected acquisition goroutine to finish, got timeout")
		}
	})
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatalf("expected owner request to start, got %v", ctx.Err())
	}
	cancel()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("expected canceled owner to finish, got %v", ctx.Err())
	}
	if firstErr != nil {
		if firstModel != nil {
			t.Fatalf("expected nil model after owner cancellation, got %p", firstModel)
		}
	}
	if firstErr == nil {
		t.Fatal("expected canceled owner error, got success")
	}
	select {
	case <-canceled:
	case <-ctx.Done():
		t.Fatalf("expected canceled HTTP handler to finish, got %v", ctx.Err())
	}

	model, err := source.Load(ctx)
	if err != nil {
		t.Fatalf("expected successful retry after owner cancellation, got %v", err)
	}
	if !reflect.DeepEqual(model, expected) {
		t.Fatalf("expected complete immutable fixture after retry, got model %p", model)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected canceled request and successful retry, got %d requests", calls.Load())
	}
	again, err := source.Load(ctx)
	if err != nil {
		t.Fatalf("expected cached model after retry, got %v", err)
	}
	if again != model || calls.Load() != 2 {
		t.Fatalf("expected cached model %p and two requests, got %p and %d requests", model, again, calls.Load())
	}

}

// TestSocioeconomicModelSourceRetryConcurrent checks failed loads are not cached
// and concurrent callers all observe the same immutable successful result.
func TestSocioeconomicModelSourceRetryConcurrent(t *testing.T) {

	fixture := socioeconomicFixture(t)
	var calls atomic.Int32
	var available atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !available.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, fixture)
	}))
	defer server.Close()
	source, err := NewSocioeconomicModelSource([]string{server.URL, server.URL + "/backup"})
	if err != nil {
		t.Fatalf("expected source, got %v", err)
	}
	model, err := source.Load(t.Context())
	if err != nil {
		if model != nil || !strings.Contains(err.Error(), "/backup") || calls.Load() != 2 {
			t.Fatalf("expected aggregated failures and nil model, got %p, %v and %d calls", model, err, calls.Load())
		}
	}
	if err == nil {
		t.Fatalf("expected failed load, got success")
	}
	available.Store(true)

	var group sync.WaitGroup
	models := make([]*SocioeconomicModel, 12)
	for i := range models {
		group.Go(func() {
			var err error
			models[i], err = source.Load(t.Context())
			if err != nil {
				t.Errorf("expected concurrent model load, got %v", err)
				return
			}
			if models[i].income[0].laws[0].values[2] != math.MaxUint32 {
				t.Errorf("expected complete immutable model, got incomplete support")
			}
		})
	}
	group.Wait()
	for _, loaded := range models {
		if loaded == nil || loaded != models[0] {
			t.Fatalf("expected same retained model, got %p and %p", loaded, models[0])
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("expected two failed attempts and one successful fetch, got %d", calls.Load())
	}

}

// TestSocioeconomicModelSourceWaitCancellation uses channels to coordinate real
// HTTP I/O; canceled waiters must not poison the successful in-flight load.
func TestSocioeconomicModelSourceWaitCancellation(t *testing.T) {

	fixture := socioeconomicFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, fixture)
	}))
	defer server.Close()
	source, err := NewSocioeconomicModelSource([]string{server.URL})
	if err != nil {
		t.Fatalf("expected source, got %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := source.Load(t.Context())
		done <- err
	}()
	<-started

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = source.Load(ctx)
	close(release)
	loadErr := <-done
	if loadErr != nil {
		t.Fatalf("expected in-flight load to succeed, got %v", loadErr)
	}
	if err != nil {
		if err != context.Canceled {
			t.Fatalf("expected canceled waiter, got %v", err)
		}
		return
	}
	t.Fatalf("expected canceled waiter, got success")

}

// socioeconomicFixture constructs the smallest complete grids rather than
// checking in 56,000 repetitive employee tuple-map entries. Each branch has one
// law; two positive Industry weights and full-width Income exercise exactness.
func socioeconomicFixture(t *testing.T) string {

	t.Helper()
	var out strings.Builder
	out.WriteString(`{"schema":"SW-L2-D2B2-logical-v0","companyCount":65536,"ages":{`)
	for i, name := range []string{
		"Education", "PrimaryActivity", "Employee.JobLevel",
		"Income[s]", "Income[u]", "Income[t]", "Income[r]", "Income[i]",
	} {
		if i > 0 {
			out.WriteByte(',')
		}
		fmt.Fprintf(&out, `%q:{"lowerBounds":[21],"compiled":%s}`, name, socioeconomicNumbers(t, 55, 0))
	}
	out.WriteString(`},"companyIndustry":{"weights":[1,2,0,0,0,0,0,0,0,0,0,0,0,0]},"employeeCount":{"laws":[`)
	out.WriteString(strings.TrimSuffix(strings.Repeat(`{"values":[0,9],"weights":[1,1]},`, 14), ","))
	out.WriteString(`]}`)
	for _, table := range []struct {
		name  string
		count int
	}{
		{"education", 140}, {"primaryActivity", 840}, {"employeeJobFamily", 1400},
		{"employeeJobLevel", 280}, {"selfEmployedJobFamily", 1400}, {"selfEmployedIndustry", 140},
	} {
		fmt.Fprintf(&out, `,%q:{"weights":%s}`, table.name, socioeconomicNumbers(t, table.count, 1))
	}
	fmt.Fprintf(&out, `,"employer":{"cap":4096,"industryFactors":%s,"sizeFactors":%s},"income":{`,
		socioeconomicNumbers(t, 140, 1), socioeconomicNumbers(t, 16, 1))
	for branch, name := range []string{"e", "s", "u", "t", "r", "i"} {

		if branch > 0 {
			out.WriteByte(',')
		}
		count := 140
		if branch == 0 {
			count = 56000
		}
		if branch == 1 {
			count = 2800
		}
		tupleMap := make([]uint16, count)
		if branch == 0 {
			for i := 0; i < count; i += 5 {
				tupleMap[i] = 65535
			}
		}

		encoded, err := json.Marshal(tupleMap)
		if err != nil {
			t.Fatalf("expected tuple JSON, got %v", err)
		}
		fmt.Fprintf(&out, `%q:{"tupleMap":%s,"laws":[{"values":[0,1,4294967295],"weights":[1,4294967295,2]}]}`,
			name, encoded)

	}
	out.WriteString(`}}`)

	return out.String()
}

func socioeconomicNumbers(t *testing.T, count int, value uint32) string {
	t.Helper()
	encoded, err := json.Marshal(slices.Repeat([]uint32{value}, count))
	if err != nil {
		t.Fatalf("expected numeric fixture JSON, got %v", err)
	}
	return string(encoded)
}
