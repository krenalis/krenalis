// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package dialer provides the dial functions Krenalis establishes its outbound
// connections with.
//
// Use it whenever a dial function is needed, and not only when the bytes sent
// must be counted: [Dial] and [DialWith] for the connections made on behalf of
// an organization, [DialWithContext] for a client shared by every organization,
// and [PlainDial] and [PlainDialWith] when there is no organization to dial on
// behalf of. Going through this package keeps every connection dialed the same
// way, and keeps counting a detail the callers do not have to care about.
//
// The organization is mandatory: the functions taking one panic if it is empty,
// and the connections that are not established on behalf of an organization
// must be dialed with [PlainDial] or [PlainDialWith], or, for
// [DialWithContext], with a context marked with [WithoutOrganization]. Not
// attributing the bytes sent is therefore always a deliberate choice, and never
// the silent result of an organization a caller has forgotten to provide.
//
// Counting is the secondary aspect: the connections established on behalf of an
// organization count the bytes they send, exposing them as the
// krenalis_organization_network_egress_bytes_total Prometheus counter, labeled
// by organization. Only the bytes sent are counted, the bytes received are not.
//
// Counting is disabled by default and is enabled with [EnableCounting], which
// is not called at all when the Prometheus metrics are disabled. While it is
// disabled, the dial functions returned by [Dial] and [DialWith] establish the
// connections as they would without this package, with no overhead, and the one
// returned by [DialWithContext] only reads the organization from the context of
// each dial, to check that it carries one.
//
// When counting is enabled, this package keeps a counter per organization, so
// it must know which organizations exist in order not to keep the counters of
// the deleted ones forever. It knows them by listening to the state, see
// [EnableCounting]: dialing on behalf of an organization that does not exist
// fails with [ErrNoOrganization].
package dialer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/prometheus"
)

// DialFunc is the type of the dial functions this package returns.
type DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// egressBytes is the Prometheus counter exposing the bytes written by each
// organization, that is its outbound (egress) traffic. Inbound traffic is not
// counted.
var egressBytes = prometheus.RegisterCounterVec(
	"krenalis_organization_network_egress_bytes_total",
	"Total bytes sent per organization",
	[]string{"organization"},
)

// enabled reports whether the bytes sent must be counted. It also reports
// whether the organizations are known, because it is only set once they are,
// see [EnableCounting].
var enabled bool

// CountingEnabled reports whether counting is enabled, that is whether
// [EnableCounting] has been called.
func CountingEnabled() bool {
	return enabled
}

// ErrNoOrganization is the error the dial functions fail with when the
// organization the bytes they send would be attributed to does not exist,
// because it has been deleted or it has never been created.
var ErrNoOrganization = errors.New("organization does not exist")

// ErrNoOrganizationInContext is the error [DialWithContext] fails a dial with
// when the context of the dial carries no organization, neither an ID set with
// [WithOrganization] nor the mark set with [WithoutOrganization].
//
// It is a broken call site, and not a condition the callers are expected to
// recover from: it is an error, and not a panic, only because a dial is made
// while a request is being served.
var ErrNoOrganizationInContext = errors.New("dialer: no organization in the context of the dial")

// organization is an organization the bytes sent can be attributed to.
//
// The dial functions resolve it once, when they are created, and then only read
// its deleted field, so that establishing a connection does not have to look it
// up and does not have to take a lock.
type organization struct {
	// egress is the counter of the bytes sent by the organization. It is
	// registered the first time the organization is resolved, and it is only
	// written with organizationsMu held.
	egress *prometheus.Counter
	// deleted reports whether the organization has been deleted. It is the only
	// field the dial functions read after they have been created.
	deleted atomic.Bool
}

var (
	organizationsMu sync.Mutex
	// organizations holds the existing organizations, by ID. An organization is
	// removed when it is deleted, so that the counters do not accumulate for
	// the whole life of the process.
	organizations = map[string]*organization{}
)

// EnableCounting enables counting and makes this package follow the
// organizations of st, so that the counter of an organization is discarded when
// the organization is deleted, and dialing on behalf of an organization that
// does not exist fails.
//
// It is not called at all when the Prometheus metrics are disabled, leaving
// counting disabled: the other functions of this package can still be called,
// they just return plain, unwrapped dialers and count nothing.
//
// When it is called, instead, it must be called at startup, before any other
// function of this package, because the dial functions already returned keep
// the setting they were created with and because the organizations are only
// known once it has returned, and it panics if it is called more than once.
func EnableCounting(st *state.State) {
	if enabled {
		panic("dialer: EnableCounting called more than once")
	}
	st.Freeze()
	st.AddListener(onCreateOrganization)
	st.AddListener(onDeleteOrganization)
	organizationsMu.Lock()
	for _, org := range st.Organizations() {
		organizations[org.ID] = &organization{}
	}
	// Counting is enabled only now that the organizations are known, so that
	// the dial functions never resolve one while they are being populated.
	enabled = true
	organizationsMu.Unlock()
	st.Unfreeze()
}

// onCreateOrganization is called when an organization is created. Its counter
// is not registered until the organization is resolved.
func onCreateOrganization(n state.CreateOrganization) {
	organizationsMu.Lock()
	if _, ok := organizations[n.ID]; !ok {
		organizations[n.ID] = &organization{}
	}
	organizationsMu.Unlock()
}

// onDeleteOrganization is called when an organization is deleted. It is marked
// as deleted, so that the dial functions that resolved it stop dialing, and its
// counter is unregistered, so that it is no longer collected and it is freed.
//
// The connections dialed by the organization before it was deleted may still be
// written to, and they keep a reference to their counter, but the bytes they
// add to it are no longer collected and the counter is freed together with the
// last connection referencing it.
func onDeleteOrganization(n state.DeleteOrganization) {
	organizationsMu.Lock()
	org, ok := organizations[n.ID]
	delete(organizations, n.ID)
	organizationsMu.Unlock()
	if !ok {
		return
	}
	org.deleted.Store(true)
	if org.egress != nil {
		org.egress.Unregister()
	}
}

// resolve returns the organization with the given ID and its egress counter,
// registering the counter the first time the organization is resolved.
//
// It fails with [ErrNoOrganization] if the organization does not exist. It is
// only called when counting is enabled, so the organizations are known, see
// [EnableCounting].
//
// The dial functions resolve the organization once, when they are created, and
// keep the returned values, so that they do not have to take organizationsMu to
// establish a connection.
func resolve(organizationID string) (*organization, *prometheus.Counter, error) {
	organizationsMu.Lock()
	defer organizationsMu.Unlock()
	org, ok := organizations[organizationID]
	if !ok {
		return nil, nil, fmt.Errorf("dialer: %w: %s", ErrNoOrganization, organizationID)
	}
	if org.egress == nil {
		org.egress = egressBytes.Register(organizationID)
	}
	return org, org.egress, nil
}

// Dial returns the dial function to establish the connections made on behalf
// of the organization with the given ID, dialing with a plain net.Dialer. The
// bytes the connections send are counted and attributed to the organization.
//
// It panics if organizationID is empty: use [PlainDial] when there is no
// organization to dial on behalf of. If counting is disabled (see
// [EnableCounting]), the returned function is a plain, unwrapped dialer and no
// bytes are counted. Otherwise, the returned function fails with
// [ErrNoOrganization] if the organization does not exist when it is called.
//
// Use [DialWith] instead to keep the dial options of an already configured
// dialer.
func Dial(organizationID string) DialFunc {
	return dialWith(organizationID, nil)
}

// DialWith returns a function that wraps the dial function establishing the
// connections made on behalf of the organization with the given ID. The bytes
// the connections send are counted and attributed to the organization.
//
// Unlike [Dial], the connections are established by the wrapped dial function,
// which therefore keeps its own dial options, like its timeouts and its
// keep-alive. If the wrapped dial function is nil, a plain net.Dialer is used,
// as in [Dial].
//
// It panics if organizationID is empty: use [PlainDialWith] when there is no
// organization to dial on behalf of. If counting is disabled (see
// [EnableCounting]), the dial function is returned unwrapped and no bytes are
// counted. Otherwise, the returned function fails with [ErrNoOrganization] if
// the organization does not exist when it is called.
func DialWith(organizationID string) func(dial DialFunc) DialFunc {
	// The organization is checked here, and not only in dialWith, so that an
	// empty one panics where DialWith is called and not where the function it
	// returns is, which may be far from it.
	if organizationID == "" {
		panic("dialer: empty organization ID")
	}
	return func(dial DialFunc) DialFunc {
		return dialWith(organizationID, dial)
	}
}

// PlainDial returns a plain net.Dialer dial function, that counts no bytes.
//
// Use it, in place of [Dial], when the connections it establishes are not made
// on behalf of an organization, and there is therefore no organization to
// attribute the bytes they send to, as for a connector under test. Dialing
// through this package, instead of with a net.Dialer of its own, keeps every
// connection Krenalis establishes dialed the same way.
func PlainDial() DialFunc {
	var d net.Dialer
	return d.DialContext
}

// PlainDialWith returns a function that returns the dial function it is given
// unwrapped, so that it establishes the connections with its own dial options
// and no bytes are counted. If the given dial function is nil, a plain
// net.Dialer is returned, as in [PlainDial].
//
// Use it, in place of [DialWith], when the connections are not established on
// behalf of an organization, as for [PlainDial].
func PlainDialWith() func(dial DialFunc) DialFunc {
	return func(dial DialFunc) DialFunc {
		if dial == nil {
			return PlainDial()
		}
		return dial
	}
}

// organizationKey is the key of the organization a dial is made on behalf of.
// Its value is a string: the ID of the organization, or the empty string when
// the dial is explicitly made on behalf of no organization, see
// [WithoutOrganization]. A context with no value at all, instead, is one whose
// organization has not been set, and [DialWithContext] refuses to dial with it.
type organizationKey struct{}

// WithOrganization returns a copy of ctx carrying the ID of the organization the
// bytes sent by the connections dialed with it are attributed to.
//
// Use it, together with [DialWithContext], when a client is shared by every
// organization and the organization is only known when the client is used, so
// that the dial function does not have to be fixed when the client is created.
//
// It panics if organizationID is empty: use [WithoutOrganization] when the
// connections dialed with the context are not established on behalf of an
// organization. The organization is carried by the context even when counting
// is disabled (see [EnableCounting]), so that [DialWithContext] can check that
// it has been provided regardless of whether the metrics are enabled.
func WithOrganization(ctx context.Context, organizationID string) context.Context {
	if organizationID == "" {
		panic("dialer: empty organization ID")
	}
	return context.WithValue(ctx, organizationKey{}, organizationID)
}

// WithoutOrganization returns a copy of ctx marked as not dialing on behalf of
// any organization, so that the bytes sent by the connections dialed with it
// are not counted.
//
// Use it, in place of [WithOrganization], when the requests made with a client
// shared by every organization are not made on behalf of one, as when a
// resource is deleted after its organization is no longer known. It is the
// [PlainDial] of [DialWithContext]: without it, [DialWithContext] refuses to
// dial, so that an organization is never left out by mistake.
func WithoutOrganization(ctx context.Context) context.Context {
	return context.WithValue(ctx, organizationKey{}, "")
}

// DialWithContext wraps the dial function of a client shared by every
// organization. The bytes the connections it establishes send are counted and
// attributed to the organization carried by the context of each dial, set with
// [WithOrganization].
//
// Unlike [DialWith], the organization is not fixed when the dial function is
// created, so a single client can serve every organization.
//
// A dial whose context carries no organization at all fails with
// [ErrNoOrganizationInContext], because an organization that has not been set
// cannot be told from one that has been forgotten: mark the context with
// [WithoutOrganization] when the connections are not established on behalf of
// an organization. A dial whose context carries an organization that does not
// exist fails with [ErrNoOrganization].
//
// If the wrapped dial function is nil, a plain net.Dialer is used, as in
// [Dial]. Unlike the other dial functions, this one is wrapped even when
// counting is disabled (see [EnableCounting]), because it checks the context of
// every dial in any case: it then only reads the organization from the context,
// counting no bytes.
func DialWithContext(dial DialFunc) DialFunc {
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		v := ctx.Value(organizationKey{})
		if v == nil {
			return nil, ErrNoOrganizationInContext
		}
		organizationID := v.(string)
		// The organization is empty when the context has been marked as dialing
		// on behalf of none, see WithoutOrganization, and there is then nothing
		// to attribute the bytes sent to.
		if organizationID == "" || !enabled {
			return dial(ctx, network, addr)
		}
		// Unlike the other dial functions, this one cannot resolve the
		// organization once, when it is created, because the organization is
		// only known at every dial, from its context.
		_, c, err := resolve(organizationID)
		if err != nil {
			return nil, err
		}
		conn, err := dial(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &instrumentedConn{Conn: conn, egress: c}, nil
	}
}

// dialWith is like [Dial], but the connections are established by dial instead
// of by a plain net.Dialer. If dial is nil, a plain net.Dialer is used.
//
// It allows counting the bytes sent by an already configured dialer, like the
// one of an http.Transport, preserving its timeouts and options.
//
// It panics if organizationID is empty. The check is made even when counting is
// disabled, so that a caller that does not provide the organization is caught
// regardless of whether the Prometheus metrics are enabled.
func dialWith(organizationID string, dial DialFunc) DialFunc {
	if organizationID == "" {
		panic("dialer: empty organization ID")
	}
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}
	if !enabled {
		return dial
	}
	// The organization is resolved once, here, and not at every dial, so that
	// establishing a connection does not have to look it up and take a lock.
	org, c, err := resolve(organizationID)
	if err != nil {
		return func(context.Context, string, string) (net.Conn, error) {
			return nil, err
		}
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// The organization may have been deleted after it was resolved, while
		// this function was still referenced by a long-lived client.
		if org.deleted.Load() {
			return nil, fmt.Errorf("dialer: %w: %s", ErrNoOrganization, organizationID)
		}
		conn, err := dial(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &instrumentedConn{Conn: conn, egress: c}, nil
	}
}

// instrumentedConn wraps a net.Conn, recording the bytes it writes into its
// organization's egress counter.
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
