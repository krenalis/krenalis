// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package dialer

import (
	"context"
	"io"
	"net"
	"sync"
	"syscall"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	krprometheus "github.com/krenalis/krenalis/tools/prometheus"

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

// counterValue returns the current value of c, collecting it directly instead
// of through egressBytes. Unlike collected, it still works after c has been
// unregistered and dropped from the counters egressBytes collects.
func counterValue(t *testing.T, c *krprometheus.Counter) uint64 {
	t.Helper()
	ch := make(chan prometheus.Metric, 1)
	c.Collect(ch)
	close(ch)
	m := &client.Metric{}
	if err := (<-ch).Write(m); err != nil {
		t.Fatalf("cannot read the counter: %s", err)
	}
	return uint64(m.GetCounter().GetValue())
}

// forget removes the organizations with the given IDs, and unregisters their
// counters, when the test ends, so that they do not leak into the other tests.
func forget(t *testing.T, organizationIDs ...string) {
	t.Helper()
	t.Cleanup(func() {
		organizationsMu.Lock()
		for _, id := range organizationIDs {
			if c := organizations[id]; c != nil {
				c.Unregister()
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
			wg.Go(func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			})
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
		organizations[id] = egressBytes.Register(id)
	}
	countingEnabled = true
	organizationsMu.Unlock()
	t.Cleanup(func() { countingEnabled = false })
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

// instrumented reports whether conn is one of the wrappers this package uses to
// count the bytes written to a connection.
func instrumented(conn net.Conn) bool {
	switch conn.(type) {
	case *instrumentedConn, *instrumentedSyscallConn:
		return true
	default:
		return false
	}
}

func TestDialDisabled(t *testing.T) {
	// The metrics are disabled, so the dialer is transparent and the bytes are
	// not counted.
	addr := echoServer(t)
	egress := egress(t, "org-disabled")
	conn := write(t, Dial("org-disabled"), addr, "hello")
	if instrumented(conn) {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	if n := egress(); n != 0 {
		t.Fatalf("counted %d bytes, expecting 0", n)
	}
}

func TestDialEmptyOrganization(t *testing.T) {
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
	if instrumented(conn) {
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
	if instrumented(conn) {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
}

func TestPlainDialWithNilDialFunc(t *testing.T) {
	// A nil dial function is replaced by a plain dialer, as in PlainDial.
	addr := echoServer(t)
	conn := write(t, PlainDialWith()(nil), addr, "hello")
	if instrumented(conn) {
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
	if !instrumented(conn) {
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

func TestInstrumentedConnSyscallConn(t *testing.T) {
	// The wrapper preserves the syscall.Conn of the connection it wraps, when
	// there is one, and does not claim one when the wrapped connection has none.
	enable(t, "org-syscall")
	addr := echoServer(t)

	// A dialed TCP connection is a syscall.Conn, and so is its wrapper, whose
	// SyscallConn delegates to it.
	conn := write(t, Dial("org-syscall"), addr, "hello")
	sc, ok := conn.(syscall.Conn)
	if !ok {
		t.Fatalf("the connection is a %T, which is not a syscall.Conn", conn)
	}
	if _, err := sc.SyscallConn(); err != nil {
		t.Fatalf("SyscallConn returned an error: %s", err)
	}

	// A connection that is not a syscall.Conn, like one end of a net.Pipe, is
	// wrapped without gaining one.
	pipe, _ := net.Pipe()
	defer pipe.Close()
	organizationsMu.Lock()
	c := organizations["org-syscall"]
	organizationsMu.Unlock()
	if _, ok := newInstrumentedConn(pipe, c).(syscall.Conn); ok {
		t.Fatalf("the wrapper of a %T is a syscall.Conn, expecting none", pipe)
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
	if !instrumented(conn) {
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

func TestDialWithContextMissingOrganization(t *testing.T) {
	// Every dial made with DialWithContext is made on behalf of an
	// organization, so a context carrying none is a caller that has forgotten
	// to set it, and the dial fails instead of silently counting nothing. It
	// fails even when counting is disabled.
	addr := echoServer(t)
	for _, enabled := range []bool{false, true} {
		if enabled {
			enable(t)
		}
		conn, err := DialWithContext(nil)(t.Context(), "tcp", addr)
		if err == nil {
			conn.Close()
			t.Fatalf("dialing with no organization in the context succeeded, expecting it to fail (counting enabled: %t)", enabled)
		}
	}
}

func TestDialWithContextWithoutOrganization(t *testing.T) {
	// The context is marked as carrying no organization, so, unlike one that
	// carries none at all, the dial does not fail and the connection simply
	// counts nothing.
	enable(t)
	addr := echoServer(t)
	conn, err := DialWithContext(nil)(WithoutOrganization(t.Context()), "tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if instrumented(conn) {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	if _, err = conn.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if _, ok := collected(t, ""); ok {
		t.Fatal("a counter is collected for the empty organization, expecting none")
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
	if instrumented(conn) {
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
	// exist. The dial is not refused, the connection simply counts nothing.
	enable(t, "org-known")
	forget(t, "org-unknown")
	addr := echoServer(t)
	dial := Dial("org-unknown")
	conn := write(t, dial, addr, "hello")
	if instrumented(conn) {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}
	// No counter is registered for an organization that does not exist.
	if _, ok := collected(t, "org-unknown"); ok {
		t.Fatal("a counter is collected for an organization that does not exist")
	}

	// The counter of the organization is taken when the dial function is
	// created, so the dial function keeps counting nothing even if an
	// organization with the same ID is created later. A dial function created
	// after it, instead, counts.
	onCreateOrganization(state.CreateOrganization{ID: "org-unknown"})
	egress := egress(t, "org-unknown")
	write(t, dial, addr, "hello")
	if n := egress(); n != 0 {
		t.Fatalf("counted %d bytes, expecting 0", n)
	}
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
	// Its counter is registered with the organization, and it is collected at
	// zero until the organization dials, so that its series does not appear
	// only once it sends something.
	if n, ok := collected(t, "org-created"); !ok || n != 0 {
		t.Fatalf("the counter of the organization is collected at %d (registered: %t), expecting 0", n, ok)
	}
	egress := egress(t, "org-created")
	write(t, Dial("org-created"), addr, "hello")
	if n := egress(); n != 5 {
		t.Fatalf("counted %d bytes, expecting 5", n)
	}
}

func TestDeletedOrganization(t *testing.T) {
	// The counter of a deleted organization is discarded, so that the counters
	// do not accumulate for the whole life of the process. The organization can
	// still dial, it just counts nothing.
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

	organizationsMu.Lock()
	c := organizations["org-deleted"]
	organizationsMu.Unlock()

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
	before := counterValue(t, c)
	if _, err := conn.Write([]byte("world!")); err != nil {
		t.Fatalf("cannot write to a connection of a deleted organization: %s", err)
	}
	if _, ok := collected(t, "org-deleted"); ok {
		t.Fatal("a counter is collected again for the deleted organization")
	}
	if n := counterValue(t, c) - before; n != 6 {
		t.Fatalf("the counter increased by %d bytes, expecting 6", n)
	}

	// A dial function created before the deletion still dials, and the bytes
	// its connections send go to the counter it holds, which is no longer
	// collected.
	before = counterValue(t, c)
	write(t, dial, addr, "hello")
	if _, ok := collected(t, "org-deleted"); ok {
		t.Fatal("a counter is collected again for the deleted organization")
	}
	if n := counterValue(t, c) - before; n != 5 {
		t.Fatalf("the counter increased by %d bytes, expecting 5", n)
	}
}

func TestDialWithContextUnknownOrganization(t *testing.T) {
	// The organization carried by the context does not exist, because it has
	// been deleted or it has never been created. Unlike Dial and DialWith, the
	// dial succeeds and counts nothing: its callers delete the resources that
	// outlive their organization, and refusing to dial would leave them
	// undeleted forever.
	enable(t, "org-ctx-known")
	addr := echoServer(t)
	dial := DialWithContext(nil)
	conn := write(t, func(ctx context.Context, network, address string) (net.Conn, error) {
		return dial(WithOrganization(ctx, "org-ctx-unknown"), network, address)
	}, addr, "hello")
	if instrumented(conn) {
		t.Fatal("the connection is instrumented, expecting a plain connection")
	}

	// No counter is collected for it, so that the counters of the deleted
	// organizations are not resurrected by the traffic that follows them.
	if _, ok := collected(t, "org-ctx-unknown"); ok {
		t.Fatal("a counter is collected for an organization that does not exist")
	}
}
