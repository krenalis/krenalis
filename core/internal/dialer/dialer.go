// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package dialer provides the dial functions Krenalis establishes its outbound
// connections with.
//
// Every connection is established on behalf of an organization: [Dial] and
// [DialWith] return the dial functions of a single organization, fixed when
// they are created, while [DialWithContext] wraps the dial function of a client
// shared by every organization, taking it from the context of each dial.
//
// In cases where there is no organization to dial on behalf of, as for a
// connector under test, use [PlainDial] and [PlainDialWith], or
// [WithoutOrganization] with [DialWithContext].
//
// The connections dialed on behalf of an organization count the bytes they
// send, exposing them as a Prometheus counter, see [EnableCounting]. If an
// organization does not exist (eg. it has been deleted), its connections are
// simply established without counting anything.
package dialer

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/prometheus"
)

// DialFunc is the type of the dial functions this package returns. It matches
// the signature of [net.Dialer.DialContext] and of the DialContext field of
// [net/http.Transport].
type DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// egressBytes is the Prometheus counter exposing the bytes written by each
// organization, that is its outbound (egress) traffic. Inbound traffic is not
// counted.
var egressBytes = prometheus.RegisterCounterVec(
	"krenalis_organization_network_egress_bytes_total",
	"Total bytes sent per organization",
	[]string{"organization"},
)

// countingEnabled reports whether the bytes sent must be counted. It is written
// by [EnableCounting] and [DisableCounting] and read at every dial.
var countingEnabled atomic.Bool

var (
	organizationsMu sync.Mutex
	// organizations holds the counter of the bytes sent by each existing
	// organization, by ID.
	organizations = map[string]*prometheus.Counter{}
)

// Dial returns the dial function to establish the connections made on behalf of
// the given organization, dialing with a plain net.Dialer. Use [DialWith]
// instead to keep the dial options of an already configured dialer.
//
// Use [PlainDial] when there is no organization to dial on behalf of (for
// example in certain test scenarios).
//
// The connections it establishes count the bytes they send, see
// [EnableCounting]. It returns a plain, unwrapped dialer, counting nothing,
// while counting is disabled and when the organization does not exist.
func Dial(organization string) DialFunc {
	return dialWith(organization, nil)
}

// DialWith returns a function that wraps the dial function establishing the
// connections made on behalf of the given organization.
//
// Unlike [Dial], the connections are established by the wrapped dial function,
// which therefore keeps its own dial options, like its timeouts and its
// keep-alive. If the wrapped dial function is nil, a plain net.Dialer is used,
// as in [Dial].
//
// Use [PlainDialWith] when there is no organization to dial on behalf of (for
// example in certain test scenarios).
//
// As in [Dial], the connections count the bytes they send, see
// [EnableCounting], and the dial function is returned unwrapped while counting
// is disabled and when the organization does not exist.
func DialWith(organization string) func(dial DialFunc) DialFunc {
	// The organization is checked here, and not only in dialWith, so that an
	// empty one panics where DialWith is called and not where the function it
	// returns is, which may be far from it.
	if organization == "" {
		panic("dialer: empty organization ID")
	}
	return func(dial DialFunc) DialFunc {
		return dialWith(organization, dial)
	}
}

// DialWithContext wraps the dial function of a client shared by every
// organization, establishing each connection on behalf of the organization
// carried by the context of the dial, set with [WithOrganization]. Unlike
// [DialWith], the organization is not fixed when the dial function is created,
// so a single client can serve more organizations. If the dial function is nil,
// a plain net.Dialer is used, as in [Dial].
//
// A dial whose context carries no organization at all fails returning an error,
// unless the context is marked with [WithoutOrganization]; a context carrying an
// organization that does not exist, instead, is legitimate, and the dial is done
// without counting.
func DialWithContext(dial DialFunc) DialFunc {
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		v := ctx.Value(organizationKey{})
		if v == nil {
			return nil, errors.New("dialer: no organization in the context of the dial")
		}
		if !countingEnabled.Load() {
			return dial(ctx, network, addr)
		}
		organization := v.(string)
		// Unlike the other dial functions, this one cannot take the counter of
		// the organization once, when it is created, because the organization
		// is only known at every dial, from its context.
		organizationsMu.Lock()
		c := organizations[organization]
		organizationsMu.Unlock()
		if c == nil {
			// The organization does not exist: dial without counting.
			return dial(ctx, network, addr)
		}
		conn, err := dial(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return newInstrumentedConn(conn, c), nil
	}
}

// DisableCounting disables counting and unregisters the counter of every
// organization, so that [EnableCounting] can enable it again in the same
// process. It does nothing if counting is not enabled.
//
// It must be called when no other function of the package is being called.
// Because the listeners [EnableCounting] adds to the state cannot be removed,
// it must also be called only once that state has stopped dispatching
// notifications. The connections dialed while counting was enabled may still be
// written to, but the bytes they add to the counter they hold are no longer
// collected.
func DisableCounting() {
	organizationsMu.Lock()
	defer organizationsMu.Unlock()
	if !countingEnabled.Load() {
		return
	}
	countingEnabled.Store(false)
	for _, c := range organizations {
		c.Unregister()
	}
	clear(organizations)
}

// EnableCounting makes the connections dialed on behalf of an organization
// count the bytes they send, exposing them as the
// krenalis_organization_network_egress_bytes_total Prometheus counter, labeled
// by organization. Only the bytes sent are counted, the bytes received are not.
//
// st is the Krenalis state that this package listens on to track changes to
// organizations.
//
// Counting is disabled by default, and it is left disabled by not calling this
// function at all: the other functions of this package can still be called,
// they just return plain, unwrapped dialers.
//
// If this function is called, it must be called before any other function in
// the package, and so must every later call following a [DisableCounting]: the
// dial functions returned before it count nothing. Call [DisableCounting] to
// reverse it, for example before a new Core enables counting again in the same
// process.
func EnableCounting(st *state.State) {
	if countingEnabled.Load() {
		panic("dialer: counting is already enabled")
	}
	st.Freeze()
	st.AddListener(onCreateOrganization)
	st.AddListener(onDeleteOrganization)
	organizationsMu.Lock()
	for _, org := range st.Organizations() {
		organizations[org.ID] = egressBytes.Register(org.ID)
	}
	countingEnabled.Store(true)
	organizationsMu.Unlock()
	st.Unfreeze()
}

// PlainDial returns a plain net.Dialer dial function.
//
// Use it, in place of [Dial], when the connections it establishes are not made
// on behalf of an organization, as for a connector under test. The bytes such
// connections send are counted for no one.
func PlainDial() DialFunc {
	return plainDial
}

// PlainDialWith returns a function that returns the dial function it is given
// unwrapped, so that it establishes the connections with its own dial options.
// If the given dial function is nil, a plain net.Dialer is returned, as in
// [PlainDial].
//
// Use it, in place of [DialWith], when the connections are not established on
// behalf of an organization, as for [PlainDial].
func PlainDialWith() func(dial DialFunc) DialFunc {
	return func(dial DialFunc) DialFunc {
		if dial == nil {
			return plainDial
		}
		return dial
	}
}

// WithOrganization returns a copy of ctx carrying the ID of the organization
// the connections dialed with it are established on behalf of.
//
// Use it, together with [DialWithContext], when a client is shared by every
// organization and the organization is only known when the client is used, so
// that the dial function does not have to be fixed when the client is created.
func WithOrganization(ctx context.Context, organization string) context.Context {
	if organization == "" {
		panic("dialer: empty organization ID")
	}
	return context.WithValue(ctx, organizationKey{}, organization)
}

// WithoutOrganization returns a copy of ctx marking the connections dialed with
// it as established on behalf of no organization, so that the bytes they send
// are counted for nobody.
//
// Use it, in place of [WithOrganization], when there is no organization to dial
// on behalf of, so that the dial is not taken for one that forgot to set it.
func WithoutOrganization(ctx context.Context) context.Context {
	return context.WithValue(ctx, organizationKey{}, "")
}

// dialWith returns a dial function that establishes the connections made on
// behalf of the given organization using dial. If dial is nil, a plain
// net.Dialer is used.
//
// It allows counting the bytes sent by an already configured dialer, preserving
// its timeouts and options.
//
// The connections it establishes count the bytes they send, see
// [EnableCounting]. It returns dial unwrapped, counting nothing, while counting
// is disabled and when the organization does not exist.
func dialWith(organization string, dial DialFunc) DialFunc {
	if organization == "" {
		panic("dialer: empty organization ID")
	}
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}
	if !countingEnabled.Load() {
		return dial
	}
	organizationsMu.Lock()
	c := organizations[organization]
	organizationsMu.Unlock()
	if c == nil {
		// The organization does not exist, so the egress traffic is not
		// counted.
		return dial
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dial(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return newInstrumentedConn(conn, c), nil
	}
}

// instrumentedConn wraps a net.Conn, recording the bytes it writes into its
// organization's egress counter. It does not expose io.ReaderFrom or
// io.WriterTo, or io.Copy to the connection would bypass Write and the count.
type instrumentedConn struct {
	net.Conn
	egress *prometheus.Counter
}

func (c *instrumentedConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		c.egress.Add(n)
	}
	return n, err
}

// instrumentedSyscallConn is an instrumentedConn that also exposes the
// syscall.Conn of the connection it wraps.
type instrumentedSyscallConn struct {
	instrumentedConn
	syscallConn syscall.Conn
}

// SyscallConn returns the raw network connection of the wrapped connection.
func (c *instrumentedSyscallConn) SyscallConn() (syscall.RawConn, error) {
	return c.syscallConn.SyscallConn()
}

// newInstrumentedConn wraps conn so that the bytes it writes are counted into
// egress, preserving the syscall.Conn of conn when it has one so that the
// wrapper can stand in for conn wherever that is used.
func newInstrumentedConn(conn net.Conn, egress *prometheus.Counter) net.Conn {
	c := instrumentedConn{Conn: conn, egress: egress}
	if sc, ok := conn.(syscall.Conn); ok {
		return &instrumentedSyscallConn{instrumentedConn: c, syscallConn: sc}
	}
	return &c
}

// onCreateOrganization is called when an organization is created, registering
// its counter, which stays at zero until the organization dials.
func onCreateOrganization(n state.CreateOrganization) {
	organizationsMu.Lock()
	organizations[n.ID] = egressBytes.Register(n.ID)
	organizationsMu.Unlock()
}

// onDeleteOrganization is called when an organization is deleted. Its counter is
// unregistered, so that it is no longer collected and it is freed.
//
// The connections dialed by the organization before it was deleted may still be
// written to, and so may the connections its dial functions establish later,
// but the bytes they add to the counter they hold are no longer collected.
func onDeleteOrganization(n state.DeleteOrganization) {
	organizationsMu.Lock()
	c := organizations[n.ID]
	delete(organizations, n.ID)
	organizationsMu.Unlock()
	// The organization has no counter when the notification is dispatched after
	// a call to DisableCounting dropped it.
	if c != nil {
		c.Unregister()
	}
}

// organizationKey is the key of the organization a dial is made on behalf of.
// Its value is a string, the ID of the organization, empty when the dial is
// made on behalf of no organization.
type organizationKey struct{}

// plainDial is the dial function of a plain net.Dialer.
var plainDial DialFunc = new(net.Dialer).DialContext
