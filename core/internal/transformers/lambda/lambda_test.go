// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package lambda

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krenalis/krenalis/core/internal/countdial"
	"github.com/krenalis/krenalis/core/internal/transformers"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/prometheus/client_golang/prometheus"
)

// egressBytesMetric is the name of the metric countdial exposes.
const egressBytesMetric = "krenalis_organization_network_egress_bytes_total"

// egressBytes returns the bytes counted, so far, as the egress traffic of the
// organization with the given ID.
func egressBytes(t *testing.T, organization string) uint64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("cannot gather the metrics: %s", err)
	}
	for _, family := range families {
		if family.GetName() != egressBytesMetric {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "organization" && label.GetValue() == organization {
					return uint64(metric.GetCounter().GetValue())
				}
			}
		}
	}
	return 0
}

// newFunction returns a function provider invoking the functions on the given
// fake Lambda endpoint, and the records to transform.
func newFunction(t *testing.T, endpoint string) transformers.FunctionProvider {
	t.Helper()
	t.Setenv("AWS_ENDPOINT_URL", endpoint)
	t.Setenv("AWS_REGION", "eu-south-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "access-key-id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret-access-key")
	return New(Settings{})
}

var (
	// callSchema is the schema of the records passed to and returned by the
	// invoked function.
	callSchema = types.Object([]types.Property{
		{Name: "name", Type: types.String()},
	})

	// callResponse is the response of a function that leaves its record
	// unchanged, as a Node.js function returns it.
	callResponse = `{"records":[{"value":{"name":"Krenalis"}}]}`
)

// TestCallCountsEgress tests that the bytes sent invoking a function are
// counted as the egress traffic of the organization the function belongs to.
func TestCallCountsEgress(t *testing.T) {

	countdial.Enabled(true)
	t.Cleanup(func() { countdial.Enabled(false) })

	var invocations int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		invocations++
		w.Write([]byte(callResponse))
	}))
	t.Cleanup(srv.Close)

	fn := newFunction(t, srv.URL)

	const organization = "test-call-egress"
	before := egressBytes(t, organization)

	records := []transformers.Record{
		{Attributes: map[string]any{"name": "Krenalis"}},
	}
	err := fn.Call(context.Background(), organization, "arn:aws:lambda:eu-south-1:1:function:f.js", "1",
		callSchema, callSchema, false, records)
	if err != nil {
		t.Fatalf("cannot call the function: %s", err)
	}

	if invocations != 1 {
		t.Fatalf("the function has been invoked %d times, expecting 1", invocations)
	}
	if got := records[0].Attributes["name"]; got != "Krenalis" {
		t.Fatalf("the record has not been transformed, got name %v", got)
	}

	// The invocation sends, at least, the request line, the headers, and the
	// payload with the record, so its bytes are far more than the ones of the
	// payload alone.
	sent := egressBytes(t, organization) - before
	if sent <= uint64(len(`"[{\"name\":\"Krenalis\"}]"`)) {
		t.Fatalf("the bytes sent invoking the function are %d, expecting more", sent)
	}
}

// TestCallDoesNotCountEgressOfOtherOrganizations tests that the bytes sent
// invoking a function are only attributed to the organization it belongs to.
func TestCallDoesNotCountEgressOfOtherOrganizations(t *testing.T) {

	countdial.Enabled(true)
	t.Cleanup(func() { countdial.Enabled(false) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(callResponse))
	}))
	t.Cleanup(srv.Close)

	fn := newFunction(t, srv.URL)

	const organization = "test-call-egress-one"
	const other = "test-call-egress-another"
	before := egressBytes(t, other)

	records := []transformers.Record{
		{Attributes: map[string]any{"name": "Krenalis"}},
	}
	err := fn.Call(context.Background(), organization, "arn:aws:lambda:eu-south-1:1:function:f.js", "1",
		callSchema, callSchema, false, records)
	if err != nil {
		t.Fatalf("cannot call the function: %s", err)
	}

	if sent := egressBytes(t, other) - before; sent != 0 {
		t.Fatalf("%d bytes have been attributed to the organization %s, expecting 0", sent, other)
	}
	if sent := egressBytes(t, organization); sent == 0 {
		t.Fatalf("no bytes have been attributed to the organization %s", organization)
	}
}

// TestCallWithMetricsDisabled tests that, when the metrics are disabled, the
// bytes sent invoking a function are not counted.
func TestCallWithMetricsDisabled(t *testing.T) {

	countdial.Enabled(false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(callResponse))
	}))
	t.Cleanup(srv.Close)

	fn := newFunction(t, srv.URL)

	const organization = "test-call-disabled"

	records := []transformers.Record{
		{Attributes: map[string]any{"name": "Krenalis"}},
	}
	err := fn.Call(context.Background(), organization, "arn:aws:lambda:eu-south-1:1:function:f.js", "1",
		callSchema, callSchema, false, records)
	if err != nil {
		t.Fatalf("cannot call the function: %s", err)
	}

	if sent := egressBytes(t, organization); sent != 0 {
		t.Fatalf("%d bytes have been counted with the metrics disabled, expecting 0", sent)
	}
}

// TestCallUsesASingleClient tests that the same client is used for every
// organization, so that no client has to be kept, and disposed of, per
// organization, and that its connections are not pooled between them, so that
// the bytes of each call are attributed to the organization that made it.
func TestCallUsesASingleClient(t *testing.T) {

	countdial.Enabled(true)
	t.Cleanup(func() { countdial.Enabled(false) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(callResponse))
	}))
	t.Cleanup(srv.Close)

	fn := newFunction(t, srv.URL).(*function)

	var client any
	for _, organization := range []string{"test-single-client-one", "test-single-client-another"} {
		records := []transformers.Record{
			{Attributes: map[string]any{"name": "Krenalis"}},
		}
		err := fn.Call(context.Background(), organization, "arn:aws:lambda:eu-south-1:1:function:f.js", "1",
			callSchema, callSchema, false, records)
		if err != nil {
			t.Fatalf("cannot call the function: %s", err)
		}
		if client == nil {
			client = fn.client
		} else if any(fn.client) != client {
			t.Fatalf("a new client has been created for the organization %s, expecting the shared one", organization)
		}
		// The call must be counted for the organization that made it, and not
		// for the one that dialed the connection the client would otherwise
		// have reused.
		if sent := egressBytes(t, organization); sent == 0 {
			t.Fatalf("no bytes have been attributed to the organization %s", organization)
		}
	}

	if err := fn.Close(context.Background()); err != nil {
		t.Fatalf("cannot close the function: %s", err)
	}
	if fn.client != nil {
		t.Fatal("the client has not been released closing the function")
	}
}
