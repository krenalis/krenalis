// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"

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

// counted returns the value of the egress counter collected for the
// organization with the given ID. It returns 0 if no counter is collected for
// it.
func counted(t *testing.T, organizationID string) uint64 {
	t.Helper()
	n, _ := collected(t, organizationID)
	return n
}

// collected returns the value of the egress counter collected for the
// organization with the given ID, and reports whether a counter is collected
// for it at all.
func collected(t *testing.T, organizationID string) (uint64, bool) {
	t.Helper()
	ch := make(chan prometheus.Metric)
	go func() {
		egressBytes.Collect(ch)
		close(ch)
	}()
	var value uint64
	var found bool
	for metric := range ch {
		m := &client.Metric{}
		if err := metric.Write(m); err != nil {
			t.Errorf("cannot read the egress counter: %s", err)
			continue
		}
		for _, label := range m.GetLabel() {
			if label.GetName() == "organization" && label.GetValue() == organizationID {
				value, found = uint64(m.GetCounter().GetValue()), true
			}
		}
	}
	return value, found
}

// forget removes the organizations with the given IDs, and unregisters their
// counters, when the test ends, so that they do not leak into the other tests.
func forget(t *testing.T, organizationIDs ...string) {
	t.Helper()
	t.Cleanup(func() {
		organizationsMu.Lock()
		for _, id := range organizationIDs {
			if org, ok := organizations[id]; ok && org.egress != nil {
				org.egress.Unregister()
			}
			delete(organizations, id)
		}
		organizationsMu.Unlock()
	})
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

// enable enables counting for the duration of the test, making the
// organizations with the given IDs the existing ones, as EnableCounting does
// with the ones of a state. Every other organization does not exist.
func enable(t *testing.T, organizationIDs ...string) {
	t.Helper()
	forget(t, organizationIDs...)
	organizationsMu.Lock()
	for _, id := range organizationIDs {
		organizations[id] = &organization{}
	}
	enabled = true
	organizationsMu.Unlock()
	t.Cleanup(func() { enabled = false })
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
	// The organization is mandatory: an empty one is a broken call site, and it
	// panics instead of silently counting nothing. It panics even when counting
	// is disabled, so that it is caught regardless of the metrics.
	for _, enabled := range []bool{false, true} {
		if enabled {
			enable(t)
		}
		for name, f := range map[string]func(){
			"Dial":             func() { Dial("") },
			"DialWith":         func() { DialWith("") },
			"WithOrganization": func() { WithOrganization(t.Context(), "") },
		} {
			func() {
				defer func() {
					if recover() == nil {
						t.Errorf("%s with an empty organization did not panic (counting enabled: %t)", name, enabled)
					}
				}()
				f()
			}()
		}
	}
}

func TestPlainDial(t *testing.T) {
	// The connections dialed with no organization are not instrumented and no
	// bytes are counted, even when counting is enabled.
	enable(t)
	addr := echoServer(t)
	conn := write(t, PlainDial(), addr, "hello")
	if _, ok := conn.(*instrumentedConn); ok {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	var dialed bool
	dial := PlainDialWith()(func(ctx context.Context, network, address string) (net.Conn, error) {
		dialed = true
		var d net.Dialer
		return d.DialContext(ctx, network, address)
	})
	conn = write(t, dial, addr, "hello")
	if !dialed {
		t.Fatal("the connection has not been established by the given dial function")
	}
	if _, ok := conn.(*instrumentedConn); ok {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
}

func TestDial(t *testing.T) {
	// Only the bytes sent are counted, and they are attributed to the
	// organization the dialer was created for.
	enable(t, "org-a", "org-b")
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
	enable(t, "org-dial-with")
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
	enable(t, "org-nil-dial")
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
	enable(t, "org-ctx-a", "org-ctx-b")
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
	// The context carries no organization at all, which cannot be told from one
	// that has been forgotten, so the dial fails instead of silently counting
	// nothing. It fails even when counting is disabled.
	addr := echoServer(t)
	for _, enabled := range []bool{false, true} {
		if enabled {
			enable(t)
		}
		_, err := DialWithContext(nil)(t.Context(), "tcp", addr)
		if !errors.Is(err, ErrNoOrganizationInContext) {
			t.Fatalf("dialing returned the error %v, expecting ErrNoOrganizationInContext (counting enabled: %t)", err, enabled)
		}
	}

	// A context marked as dialing on behalf of no organization, instead, dials
	// and counts nothing, even though counting is enabled.
	egress := egress(t, "")
	dial := DialWithContext(nil)
	conn := write(t, func(ctx context.Context, network, address string) (net.Conn, error) {
		return dial(WithoutOrganization(ctx), network, address)
	}, addr, "hello")
	if _, ok := conn.(*instrumentedConn); ok {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	if n := egress(); n != 0 {
		t.Fatalf("counted %d bytes, expecting 0", n)
	}
}

func TestDialWithContextDisabled(t *testing.T) {
	// The metrics are disabled, so no bytes are counted, but the organization is
	// carried by the context anyway, so that it is checked in any case.
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

func TestDialUnknownOrganization(t *testing.T) {
	// The organization dialing is not among the existing ones, so it does not
	// exist and the dial fails.
	enable(t, "org-known")
	forget(t, "org-unknown")
	addr := echoServer(t)
	dial := Dial("org-unknown")
	_, err := dial(t.Context(), "tcp", addr)
	if !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("dialing returned the error %v, expecting ErrNoOrganization", err)
	}
	// No counter is registered for an organization that does not exist.
	if _, ok := collected(t, "org-unknown"); ok {
		t.Fatal("a counter is collected for an organization that does not exist")
	}

	// The organization is resolved when the dial function is created, so the
	// dial function keeps failing even if an organization with the same ID is
	// created later. A dial function created after it, instead, dials.
	onCreateOrganization(state.CreateOrganization{ID: "org-unknown"})
	if _, err := dial(t.Context(), "tcp", addr); !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("dialing returned the error %v, expecting ErrNoOrganization", err)
	}
	egress := egress(t, "org-unknown")
	write(t, Dial("org-unknown"), addr, "hello")
	if n := egress(); n != 5 {
		t.Fatalf("counted %d bytes, expecting 5", n)
	}
}

func TestDialCreatedOrganization(t *testing.T) {
	// An organization created after counting is enabled exists, so it can dial
	// and its bytes are counted.
	enable(t)
	forget(t, "org-created")
	addr := echoServer(t)
	onCreateOrganization(state.CreateOrganization{ID: "org-created"})
	egress := egress(t, "org-created")
	write(t, Dial("org-created"), addr, "hello")
	if n := egress(); n != 5 {
		t.Fatalf("counted %d bytes, expecting 5", n)
	}
}

func TestDeletedOrganization(t *testing.T) {
	// The counter of a deleted organization is discarded, so that the counters
	// do not accumulate for the whole life of the process, and the organization
	// can no longer dial.
	enable(t, "org-deleted")
	addr := echoServer(t)
	dial := Dial("org-deleted")
	conn, err := dial(t.Context(), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if _, ok := collected(t, "org-deleted"); !ok {
		t.Fatal("no counter is collected for the organization, expecting one")
	}

	onDeleteOrganization(state.DeleteOrganization{ID: "org-deleted"})

	// The organization is gone, and so is its counter.
	organizationsMu.Lock()
	_, kept := organizations["org-deleted"]
	organizationsMu.Unlock()
	if kept {
		t.Fatal("the deleted organization is still kept")
	}
	if _, ok := collected(t, "org-deleted"); ok {
		t.Fatal("a counter is still collected for the deleted organization")
	}

	// A connection dialed before the deletion may still be written to. Its
	// bytes are added to the counter it holds, which is no longer collected.
	if _, err := conn.Write([]byte("world!")); err != nil {
		t.Fatalf("cannot write to a connection of a deleted organization: %s", err)
	}
	if _, ok := collected(t, "org-deleted"); ok {
		t.Fatal("a counter is collected again for the deleted organization")
	}

	// A dial function created before the deletion no longer dials, because the
	// organization is looked up at every dial.
	if _, err := dial(t.Context(), "tcp", addr); !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("dialing returned the error %v, expecting ErrNoOrganization", err)
	}
}

func TestDialWithContextUnknownOrganization(t *testing.T) {
	// The organization carried by the context does not exist, so the dial
	// fails.
	enable(t, "org-ctx-known")
	addr := echoServer(t)
	ctx := WithOrganization(t.Context(), "org-ctx-unknown")
	_, err := DialWithContext(nil)(ctx, "tcp", addr)
	if !errors.Is(err, ErrNoOrganization) {
		t.Fatalf("dialing returned the error %v, expecting ErrNoOrganization", err)
	}
}

func TestCountingEnabled(t *testing.T) {
	if CountingEnabled() {
		t.Fatal("enabled, expecting it to be disabled by default")
	}
	enable(t)
	if !CountingEnabled() {
		t.Fatal("disabled, expecting it to be enabled")
	}
}
