// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package dialer provides the dial functions Krenalis establishes its outbound
// connections with. Use it whenever a dial function is needed, so that every
// connection is dialed the same way.
//
// Every connection is established on behalf of an organization: [Dial] and
// [DialWith] return the dial functions of a single organization, fixed when
// they are created, while [DialWithContext] wraps the dial function of a client
// shared by every organization, taking it from the context of each dial. The
// organization is mandatory, and the functions taking one panic if it is empty.
//
// In the rare cases where there is no organization to dial on behalf of, as for
// a connector under test, use [PlainDial] and [PlainDialWith]. Dialing without
// an organization is therefore always a deliberate choice, and never the silent
// result of one a caller has forgotten to provide.
//
// Secondarily, the connections dialed on behalf of an organization count the
// bytes they send, exposing them as a Prometheus counter, see [EnableCounting].
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

// CountingEnabled reports whether the connections dialed on behalf of an
// organization count the bytes they send, that is whether [EnableCounting] has
// been called.
func CountingEnabled() bool {
	return enabled
}

// ErrNoOrganization is the error the dial functions fail with when the
// organization they dial on behalf of does not exist, because it has been
// deleted or it has never been created.
var ErrNoOrganization = errors.New("organization does not exist")

// ErrNoOrganizationInContext is the error [DialWithContext] fails a dial with
// when the context of the dial carries no organization, that is no ID set with
// [WithOrganization].
//
// It is a broken call site, and not a condition the callers are expected to
// recover from: it is an error, and not a panic, only because a dial is made
// while a request is being served.
var ErrNoOrganizationInContext = errors.New("dialer: no organization in the context of the dial")

// organization is an organization the connections are dialed on behalf of.
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

// EnableCounting makes the connections dialed on behalf of an organization
// count the bytes they send, exposing them as the
// krenalis_organization_network_egress_bytes_total Prometheus counter, labeled
// by organization. Only the bytes sent are counted, the bytes received are not.
//
// Counting is disabled by default, and it is left disabled by not calling this
// function at all, as when the Prometheus metrics are disabled: the other
// functions of this package can still be called, they just return plain,
// unwrapped dialers.
//
// It also makes this package follow the organizations of st, so that the
// counter of an organization is discarded when the organization is deleted,
// instead of being kept for the whole life of the process, and so that dialing
// on behalf of an organization that does not exist fails. The organizations are
// therefore only known once counting is enabled.
//
// It must be called at startup, before any other function of this package,
// because the dial functions already returned keep the setting they were
// created with, and it panics if it is called more than once.
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
// registering the counter the first time the organization is resolved. The
// boolean return value reports whether the organization exists.
//
// A missing organization is not an error here, because the dial functions do
// not agree on what it means: it is one for those whose organization is fixed
// when they are created, and it is not for [DialWithContext]. The counter is
// not registered for it in any case, so that the counters of the deleted
// organizations are not resurrected.
//
// It is only called when counting is enabled, so the organizations are known,
// see [EnableCounting].
//
// The dial functions resolve the organization once, when they are created, and
// keep the returned values, so that they do not have to take organizationsMu to
// establish a connection.
func resolve(organizationID string) (*organization, *prometheus.Counter, bool) {
	organizationsMu.Lock()
	defer organizationsMu.Unlock()
	org, ok := organizations[organizationID]
	if !ok {
		return nil, nil, false
	}
	if org.egress == nil {
		org.egress = egressBytes.Register(organizationID)
	}
	return org, org.egress, true
}

// Dial returns the dial function to establish the connections made on behalf of
// the organization with the given ID, dialing with a plain net.Dialer. Use
// [DialWith] instead to keep the dial options of an already configured dialer.
//
// It panics if organizationID is empty: use [PlainDial] when there is no
// organization to dial on behalf of. The returned function fails with
// [ErrNoOrganization] if the organization does not exist when it is called.
//
// The connections it establishes count the bytes they send, see
// [EnableCounting]. While counting is disabled the organizations are not known,
// so it returns a plain, unwrapped dialer that never fails.
func Dial(organizationID string) DialFunc {
	return dialWith(organizationID, nil)
}

// DialWith returns a function that wraps the dial function establishing the
// connections made on behalf of the organization with the given ID.
//
// Unlike [Dial], the connections are established by the wrapped dial function,
// which therefore keeps its own dial options, like its timeouts and its
// keep-alive. If the wrapped dial function is nil, a plain net.Dialer is used,
// as in [Dial].
//
// It panics if organizationID is empty: use [PlainDialWith] when there is no
// organization to dial on behalf of. The returned function fails with
// [ErrNoOrganization] if the organization does not exist when it is called.
//
// As in [Dial], the connections count the bytes they send, see
// [EnableCounting], and while counting is disabled the dial function is
// returned unwrapped.
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

// PlainDial returns a plain net.Dialer dial function.
//
// Use it, in place of [Dial], when the connections it establishes are not made
// on behalf of an organization, as for a connector under test. Dialing through
// this package, instead of with a net.Dialer of its own, keeps every connection
// Krenalis establishes dialed the same way. The bytes such connections send are
// counted for no one.
func PlainDial() DialFunc {
	var d net.Dialer
	return d.DialContext
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
			return PlainDial()
		}
		return dial
	}
}

// organizationKey is the key of the organization a dial is made on behalf of.
// Its value is a string, the ID of the organization, and it is never empty,
// because [WithOrganization] is the only way to set it and it panics on an
// empty ID. A context with no value at all, instead, is one whose organization
// has not been set, and [DialWithContext] refuses to dial with it.
type organizationKey struct{}

// WithOrganization returns a copy of ctx carrying the ID of the organization
// the connections dialed with it are established on behalf of.
//
// Use it, together with [DialWithContext], when a client is shared by every
// organization and the organization is only known when the client is used, so
// that the dial function does not have to be fixed when the client is created.
//
// It panics if organizationID is empty: every dial made with [DialWithContext]
// is made on behalf of an organization, and one that has been deleted is no
// exception, as its ID is passed all the same. The organization is carried by
// the context even when counting is disabled (see [EnableCounting]), so that
// [DialWithContext] can check that it has been provided in any case.
func WithOrganization(ctx context.Context, organizationID string) context.Context {
	if organizationID == "" {
		panic("dialer: empty organization ID")
	}
	return context.WithValue(ctx, organizationKey{}, organizationID)
}

// DialWithContext wraps the dial function of a client shared by every
// organization, establishing each connection on behalf of the organization
// carried by the context of the dial, set with [WithOrganization]. Unlike
// [DialWith], the organization is not fixed when the dial function is created,
// so a single client can serve every organization. If the wrapped dial function
// is nil, a plain net.Dialer is used, as in [Dial].
//
// A dial whose context carries no organization at all fails with
// [ErrNoOrganizationInContext]: every dial made through this function is made
// on behalf of an organization, so a context without one is a caller that has
// forgotten to set it.
//
// A dial whose context carries an organization that does not exist, instead,
// establishes the connection, unlike [Dial] and [DialWith], which fail with
// [ErrNoOrganization]. The organization is provided at every dial here, by
// callers that legitimately act on behalf of one that has been deleted: a
// resource outliving its organization, like a transformation function, is
// deleted after it, and refusing to dial would leave it undeleted forever. The
// bytes such a dial sends are counted for no one.
//
// The connections count the bytes they send, see [EnableCounting]. Unlike the
// other dial functions, this one wraps the given dial function even when
// counting is disabled, because it checks the context of every dial in any
// case: it then only reads the organization from the context.
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
		// The ID is never empty, see organizationKey.
		organizationID := v.(string)
		if !enabled {
			return dial(ctx, network, addr)
		}
		// Unlike the other dial functions, this one cannot resolve the
		// organization once, when it is created, because the organization is
		// only known at every dial, from its context.
		_, c, ok := resolve(organizationID)
		if !ok {
			// The organization does not exist, so there is nothing to attribute
			// the bytes sent to. It is not an error here, see the comment on
			// this function.
			return dial(ctx, network, addr)
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
	org, c, ok := resolve(organizationID)
	if !ok {
		err := fmt.Errorf("dialer: %w: %s", ErrNoOrganization, organizationID)
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
