// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package lambda

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/core/internal/transformers"
	"github.com/krenalis/krenalis/tools/types"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/prometheus/client_golang/prometheus"
)

// egressBytesMetric is the name of the metric dialer exposes.
const egressBytesMetric = "krenalis_organization_network_egress_bytes_total"

// egressBytes returns the bytes counted, so far, as the egress traffic of the
// organization with the given ID. It returns zero both when the organization
// counted no bytes and when it has no counter at all, as is the case while
// counting is disabled.
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
// fake Lambda endpoint.
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

// newEndpoint returns a fake Lambda endpoint, responding to every invocation
// with callResponse, and the number of the invocations it has received so far,
// so that a test does not pass on a call that never left.
func newEndpoint(t *testing.T) (endpoint string, invocations func() int) {
	t.Helper()
	var mu sync.Mutex
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		n++
		mu.Unlock()
		w.Write([]byte(callResponse))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() int {
		mu.Lock()
		defer mu.Unlock()
		return n
	}
}

// call invokes, on behalf of the given organization, a function that leaves its
// record unchanged, and checks that the record comes back transformed.
func call(t *testing.T, fn transformers.FunctionProvider, organization string) {
	t.Helper()
	records := []transformers.Record{
		{Attributes: map[string]any{"name": "Krenalis"}},
	}
	err := fn.Call(context.Background(), organization, "arn:aws:lambda:eu-south-1:1:function:f.js", "1",
		callSchema, callSchema, false, records)
	if err != nil {
		t.Fatalf("cannot call the function: %s", err)
	}
	if err := records[0].Err; err != nil {
		t.Fatalf("the record has not been transformed: %s", err)
	}
	if name := records[0].Attributes["name"]; name != "Krenalis" {
		t.Fatalf("the transformed record has name %v, expecting Krenalis", name)
	}
}

// TestCallWithCountingDisabled tests that a function is invoked, and no bytes
// are counted for its organization, while counting is disabled.
//
// The counting of the bytes sent is what is disabled here, not the metrics of
// this package, which are always collected. Counting is disabled because
// dialer.EnableCounting is not called and counting is disabled by default.
//
// Being disabled, no organization has a counter to add bytes to, so the bytes
// counted can only be zero: what this test really guards is that the client
// dials on behalf of the organization, as TestWithDialer requires, because a
// dial whose context carries no organization fails. The counting itself is
// tested in the dialer package.
func TestCallWithCountingDisabled(t *testing.T) {

	endpoint, invocations := newEndpoint(t)
	fn := newFunction(t, endpoint)

	const organization = "test-call-disabled"

	call(t, fn, organization)

	if n := invocations(); n != 1 {
		t.Fatalf("the endpoint has been invoked %d times, expecting 1", n)
	}
	if sent := egressBytes(t, organization); sent != 0 {
		t.Fatalf("%d bytes have been counted with counting disabled, expecting 0", sent)
	}
}

// TestCallUsesASingleClient tests that the same client is used for every
// organization, so that no client has to be kept, and disposed of, per
// organization, and that it is released when the function is closed.
func TestCallUsesASingleClient(t *testing.T) {

	endpoint, invocations := newEndpoint(t)
	fn := newFunction(t, endpoint).(*function)

	// The first call creates the client.
	call(t, fn, "test-single-client-one")
	client := fn.client
	if client == nil {
		t.Fatal("no client has been created calling the function")
	}

	// A call on behalf of another organization reuses it.
	call(t, fn, "test-single-client-another")
	if fn.client != client {
		t.Fatal("a new client has been created for another organization, expecting the shared one")
	}

	if n := invocations(); n != 2 {
		t.Fatalf("the endpoint has been invoked %d times, expecting 2", n)
	}

	if err := fn.Close(context.Background()); err != nil {
		t.Fatalf("cannot close the function: %s", err)
	}
	if fn.client != nil {
		t.Fatal("the client has not been released closing the function")
	}
}

// TestWithDialer tests that the client establishes its connections with the
// dial function of the dialer package, and that it keeps them from being reused
// by another organization. A client created without withDialer would dial on
// its own, and silently so.
func TestWithDialer(t *testing.T) {

	fn := newFunction(t, "http://127.0.0.1:1").(*function)
	client, err := fn.lambdaClient(t.Context())
	if err != nil {
		t.Fatalf("cannot create the client: %s", err)
	}
	httpClient, ok := client.Options().HTTPClient.(*awshttp.BuildableClient)
	if !ok {
		t.Fatalf("the client dials with a %T, expecting a *awshttp.BuildableClient", client.Options().HTTPClient)
	}
	transport := httpClient.GetTransport()

	// The organization is resolved when a connection is dialed, so a pooled
	// connection would count for the organization that dialed it the bytes of
	// the requests it later serves for another one.
	if !transport.DisableKeepAlives {
		t.Fatal("the keep-alives are enabled, expecting them to be disabled")
	}

	// Only the dial function of the dialer package fails, instead of dialing,
	// when the context carries no organization.
	if transport.DialContext == nil {
		t.Fatal("the client dials with no dial function of its own, expecting the one of the dialer")
	}
	conn, err := transport.DialContext(t.Context(), "tcp", "127.0.0.1:1")
	if err == nil {
		conn.Close()
		t.Fatal("a connection has been dialed with no organization in its context, expecting an error")
	}
	if !strings.Contains(err.Error(), "no organization in the context") {
		t.Fatalf("the client does not dial with the dialer of the organization: %s", err)
	}
}
