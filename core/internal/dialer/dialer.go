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
// connector under test, use [PlainDial] and [PlainDialWith].
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

// countingEnabled reports whether the bytes sent must be counted.
var countingEnabled bool

var (
	organizationsMu sync.Mutex
	// organizations holds the counter of the bytes sent by each existing
	// organization, by ID. The counter is registered when the organization is
	// created and it is unregistered when the organization is deleted, so that
	// the counters do not accumulate for the whole life of the process.
	//
	// An organization that does not exist has no entry here, so looking it up
	// yields a nil counter: dialing on its behalf is not an error, there is
	// simply nothing to count the bytes it sends for. The dial functions look
	// it up only when counting is enabled, so the organizations are known, see
	// [EnableCounting].
	organizations = map[string]*prometheus.Counter{}
)

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
// the package.
func EnableCounting(st *state.State) {
	if countingEnabled {
		panic("dialer: EnableCounting called more than once")
	}
	st.Freeze()
	st.AddListener(onCreateOrganization)
	st.AddListener(onDeleteOrganization)
	organizationsMu.Lock()
	for _, org := range st.Organizations() {
		organizations[org.ID] = egressBytes.Register(org.ID)
	}
	// Counting is enabled only now that the organizations are known, so that
	// the dial functions never look one up while they are being populated.
	countingEnabled = true
	organizationsMu.Unlock()
	st.Unfreeze()
}

// onCreateOrganization is called when an organization is created, registering
// its counter, which stays at zero until the organization dials.
func onCreateOrganization(n state.CreateOrganization) {
	organizationsMu.Lock()
	if _, ok := organizations[n.ID]; !ok {
		organizations[n.ID] = egressBytes.Register(n.ID)
	}
	organizationsMu.Unlock()
}

// onDeleteOrganization is called when an organization is deleted. Its counter is
// unregistered, so that it is no longer collected and it is freed.
//
// The connections dialed by the organization before it was deleted may still be
// written to, and so may the connections its dial functions establish later,
// but the bytes they add to the counter they hold are no longer collected, and
// the counter is freed together with the last of them.
func onDeleteOrganization(n state.DeleteOrganization) {
	organizationsMu.Lock()
	c := organizations[n.ID]
	delete(organizations, n.ID)
	organizationsMu.Unlock()
	// The counter is nil when the organization does not exist, and there is
	// then nothing to unregister.
	if c != nil {
		c.Unregister()
	}
}

// Dial returns the dial function to establish the connections made on behalf of
// the given organization, dialing with a plain net.Dialer. Use [DialWith]
// instead to keep the dial options of an already configured dialer.
//
// It panics if organization is empty: use [PlainDial] when there is no
// organization to dial on behalf of.
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
// It panics if organization is empty: use [PlainDialWith] when there is no
// organization to dial on behalf of.
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
// It panics if organization is empty: every dial made with [DialWithContext]
// is made on behalf of an organization, and one that has been deleted is no
// exception, as its ID is passed all the same. The organization is carried by
// the context even when counting is disabled (see [EnableCounting]), so that
// [DialWithContext] can check that it has been provided in any case.
func WithOrganization(ctx context.Context, organization string) context.Context {
	if organization == "" {
		panic("dialer: empty organization ID")
	}
	return context.WithValue(ctx, organizationKey{}, organization)
}

// DialWithContext wraps the dial function of a client shared by every
// organization, establishing each connection on behalf of the organization
// carried by the context of the dial, set with [WithOrganization]. Unlike
// [DialWith], the organization is not fixed when the dial function is created,
// so a single client can serve every organization. If the wrapped dial function
// is nil, a plain net.Dialer is used, as in [Dial].
//
// A dial whose context carries no organization at all fails: every dial made
// through this function is made on behalf of an organization, so a context
// without one is a caller that has forgotten to set it. A context carrying an
// organization that does not exist, instead, is legitimate, as its callers act
// on behalf of the organizations that have been deleted: a resource outliving
// its organization, like a transformation function, is deleted after it.
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
			return nil, errors.New("dialer: no organization in the context of the dial")
		}
		// The ID is never empty, see organizationKey.
		organization := v.(string)
		if !countingEnabled {
			return dial(ctx, network, addr)
		}
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
		return &instrumentedConn{Conn: conn, egress: c}, nil
	}
}

// dialWith is like [Dial], but the connections are established by dial instead
// of by a plain net.Dialer. If dial is nil, a plain net.Dialer is used.
//
// It allows counting the bytes sent by an already configured dialer, like the
// one of an http.Transport, preserving its timeouts and options.
//
// It panics if organization is empty. The check is made even when counting is
// disabled, so that a caller that does not provide the organization is caught
// regardless of whether the Prometheus metrics are enabled.
func dialWith(organization string, dial DialFunc) DialFunc {
	if organization == "" {
		panic("dialer: empty organization ID")
	}
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}
	if !countingEnabled {
		return dial
	}
	// The counter of the organization is taken once, here, and not at every
	// dial, so that establishing a connection does not have to look it up and
	// take the lock.
	organizationsMu.Lock()
	c := organizations[organization]
	organizationsMu.Unlock()
	if c == nil {
		// The organization does not exist, so there is nothing to count the
		// bytes sent for and the connections are established as they would be
		// without this package.
		return dial
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
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
