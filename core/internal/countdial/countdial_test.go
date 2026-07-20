// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package countdial

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	client "github.com/prometheus/client_model/go"
)

// egress returns a function that reports the bytes counted for the organization
// with the given ID since egress was called. The counters are global and
// accumulate for the whole life of the process, so the tests can only rely on
// how much they increase.
func egress(t *testing.T, organizationID string) func() uint64 {
	t.Helper()
	before := counted(t, organizationID)
	return func() uint64 {
		return counted(t, organizationID) - before
	}
}

// counted returns the value of the egress counter of the organization with the
// given ID. It returns 0 if the organization has no counter.
func counted(t *testing.T, organizationID string) uint64 {
	t.Helper()
	countersMu.Lock()
	c, ok := counters[organizationID]
	countersMu.Unlock()
	if !ok {
		return 0
	}
	ch := make(chan prometheus.Metric, 1)
	c.Collect(ch)
	close(ch)
	m := &client.Metric{}
	err := (<-ch).Write(m)
	if err != nil {
		t.Fatalf("cannot read the counter of organization %q: %s", organizationID, err)
	}
	return uint64(m.GetCounter().GetValue())
}

// echoServer starts a server that echoes back what it is written, and returns
// its address. The server is closed when the test ends.
func echoServer(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	t.Cleanup(func() {
		_ = l.Close()
		wg.Wait()
	})
	return l.Addr().String()
}

// enable enables the metrics for the duration of the test.
func enable(t *testing.T) {
	t.Helper()
	Enabled(true)
	t.Cleanup(func() { Enabled(false) })
}

// write writes b to the connection established by dial to addr, reads the echo
// back, and closes the connection. It returns the established connection.
func write(t *testing.T, dial DialFunc, addr, s string) net.Conn {
	t.Helper()
	conn, err := dial(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	n, err := conn.Write([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	if n != len(s) {
		t.Fatalf("written %d bytes, expecting %d", n, len(s))
	}
	// Read the echo back, so that the bytes received are not counted.
	_, err = io.ReadFull(conn, make([]byte, len(s)))
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestDialDisabled(t *testing.T) {
	// The metrics are disabled, so the dialer is transparent and the bytes are
	// not counted.
	addr := echoServer(t)
	egress := egress(t, "org-disabled")
	conn := write(t, Dial("org-disabled"), addr, "hello")
	if _, ok := conn.(*instrumentedConn); ok {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	if n := egress(); n != 0 {
		t.Fatalf("counted %d bytes, expecting 0", n)
	}
}

func TestDialWithoutOrganization(t *testing.T) {
	// The organization is unknown, so the bytes are not counted even if the
	// metrics are enabled.
	enable(t)
	addr := echoServer(t)
	egress := egress(t, "")
	conn := write(t, Dial(""), addr, "hello")
	if _, ok := conn.(*instrumentedConn); ok {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	if n := egress(); n != 0 {
		t.Fatalf("counted %d bytes, expecting 0", n)
	}
}

func TestDial(t *testing.T) {
	// Only the bytes sent are counted, and they are attributed to the
	// organization the dialer was created for.
	enable(t)
	addr := echoServer(t)
	egressA := egress(t, "org-a")
	egressB := egress(t, "org-b")
	conn := write(t, Dial("org-a"), addr, "hello")
	if _, ok := conn.(*instrumentedConn); !ok {
		t.Fatalf("the connection is a %T, expecting an instrumented connection", conn)
	}
	if n := egressA(); n != 5 {
		t.Fatalf("counted %d bytes, expecting 5", n)
	}
	// The counter of an organization accumulates the bytes of all its
	// connections.
	write(t, Dial("org-a"), addr, "world!")
	if n := egressA(); n != 11 {
		t.Fatalf("counted %d bytes, expecting 11", n)
	}
	// The bytes of an organization are not attributed to another one.
	write(t, Dial("org-b"), addr, "hi")
	if n := egressA(); n != 11 {
		t.Fatalf("counted %d bytes for org-a, expecting 11", n)
	}
	if n := egressB(); n != 2 {
		t.Fatalf("counted %d bytes for org-b, expecting 2", n)
	}
}

func TestDialWith(t *testing.T) {
	// The bytes are counted and the connection is established by the given dial
	// function, and not by a plain dialer.
	enable(t)
	addr := echoServer(t)
	egress := egress(t, "org-dial-with")
	var dialed bool
	dial := DialWith("org-dial-with")(func(ctx context.Context, network, address string) (net.Conn, error) {
		dialed = true
		var d net.Dialer
		return d.DialContext(ctx, network, address)
	})
	write(t, dial, addr, "hello")
	if !dialed {
		t.Fatal("the connection has not been established by the given dial function")
	}
	if n := egress(); n != 5 {
		t.Fatalf("counted %d bytes, expecting 5", n)
	}
}

func TestDialWithNilDialFunc(t *testing.T) {
	// A nil dial function is replaced by a plain dialer, as in Dial.
	enable(t)
	addr := echoServer(t)
	egress := egress(t, "org-nil-dial")
	write(t, DialWith("org-nil-dial")(nil), addr, "hello")
	if n := egress(); n != 5 {
		t.Fatalf("counted %d bytes, expecting 5", n)
	}
}

func TestDialWithContext(t *testing.T) {
	// A single dial function attributes the bytes to the organization carried
	// by the context of each dial.
	enable(t)
	addr := echoServer(t)
	egressA := egress(t, "org-ctx-a")
	egressB := egress(t, "org-ctx-b")
	var dialed bool
	dial := DialWithContext(func(ctx context.Context, network, address string) (net.Conn, error) {
		dialed = true
		var d net.Dialer
		return d.DialContext(ctx, network, address)
	})
	conn, err := dial(WithOrganization(t.Context(), "org-ctx-a"), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	if !dialed {
		t.Fatal("the connection has not been established by the given dial function")
	}
	if _, ok := conn.(*instrumentedConn); !ok {
		t.Fatalf("the connection is a %T, expecting an instrumented connection", conn)
	}
	if _, err = conn.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if n := egressA(); n != 5 {
		t.Fatalf("counted %d bytes for org-ctx-a, expecting 5", n)
	}
	// The same dial function attributes the bytes of another context to another
	// organization.
	conn, err = dial(WithOrganization(t.Context(), "org-ctx-b"), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conn.Write([]byte("hi")); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	if n := egressA(); n != 5 {
		t.Fatalf("counted %d bytes for org-ctx-a, expecting 5", n)
	}
	if n := egressB(); n != 2 {
		t.Fatalf("counted %d bytes for org-ctx-b, expecting 2", n)
	}
}

func TestDialWithContextWithoutOrganization(t *testing.T) {
	// The context carries no organization, so the bytes are not counted even if
	// the metrics are enabled.
	enable(t)
	addr := echoServer(t)
	egress := egress(t, "")
	conn := write(t, DialWithContext(nil), addr, "hello")
	if _, ok := conn.(*instrumentedConn); ok {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	if n := egress(); n != 0 {
		t.Fatalf("counted %d bytes, expecting 0", n)
	}
}

func TestDialWithContextDisabled(t *testing.T) {
	// The metrics are disabled, so the organization is not put into the context
	// and the dialer is transparent.
	addr := echoServer(t)
	egress := egress(t, "org-ctx-disabled")
	ctx := WithOrganization(t.Context(), "org-ctx-disabled")
	conn, err := DialWithContext(nil)(ctx, "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, ok := conn.(*instrumentedConn); ok {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	if _, err = conn.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if n := egress(); n != 0 {
		t.Fatalf("counted %d bytes, expecting 0", n)
	}
}

func TestTransport(t *testing.T) {
	base := http.DefaultTransport.(*http.Transport)

	// The metrics are disabled, so the base transport is returned unwrapped.
	if transport := Transport(base, "org-transport"); transport != http.RoundTripper(base) {
		t.Fatal("the base transport has been wrapped, expecting it unwrapped")
	}

	enable(t)

	// The organization is unknown, so the base transport is returned unwrapped.
	if transport := Transport(base, ""); transport != http.RoundTripper(base) {
		t.Fatal("the base transport has been wrapped, expecting it unwrapped")
	}

	// The bytes the requests send are counted, the bytes they receive are not.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(strings.Repeat("x", 1024)))
	}))
	defer server.Close()
	transport := Transport(base, "org-transport")
	if transport == http.RoundTripper(base) {
		t.Fatal("the base transport has not been wrapped")
	}
	egress := egress(t, "org-transport")
	body := strings.Repeat("a", 512)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	res, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	// The request sends its body plus its headers, and receives a longer
	// response, that must not be counted.
	n := egress()
	if n < uint64(len(body)) {
		t.Fatalf("counted %d bytes, expecting at least the %d bytes of the request body", n, len(body))
	}
	if n >= 1024 {
		t.Fatalf("counted %d bytes, expecting the bytes received not to be counted", n)
	}
}

func TestIsEnabled(t *testing.T) {
	if IsEnabled() {
		t.Fatal("enabled, expecting it to be disabled by default")
	}
	enable(t)
	if !IsEnabled() {
		t.Fatal("disabled, expecting it to be enabled")
	}
}
