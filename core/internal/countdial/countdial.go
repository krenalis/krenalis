// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package countdial provides dial functions that count the bytes the
// connections they establish send, attributing them to an organization.
//
// The counted bytes are exposed as the
// krenalis_organization_network_egress_bytes_total Prometheus counter, labeled
// by organization. Only the bytes sent are counted, the bytes received are not.
//
// Counting is disabled by default and is enabled with [Enabled]. When it is
// disabled, the dial functions returned by [Dial], [DialWith] and
// [DialWithContext], and the transport returned by [Transport], establish the
// connections as they would without this package, with no overhead.
//
// This package keeps a counter per organization, so it must know which
// organizations exist in order not to keep the counters of the deleted ones
// forever. It knows them by listening to the state, see [Listen]: dialing on
// behalf of an organization that does not exist fails with [ErrNoOrganization].
package countdial

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/prometheus"
)

// DialFunc is the type of the dial functions returned by [Dial].
type DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// egressBytes is the Prometheus counter exposing the bytes written by each
// organization, that is its outbound (egress) traffic. Inbound traffic is not
// counted.
var egressBytes = prometheus.RegisterCounterVec(
	"krenalis_organization_network_egress_bytes_total",
	"Total bytes sent per organization",
	[]string{"organization"},
)

// enabled reports whether the bytes sent must be counted. It is false by
// default, so that the dial functions are plain and unwrapped unless counting
// is explicitly enabled.
var enabled atomic.Bool

// Enabled enables or disables counting. It should be called once, at startup,
// before any connection is dialed: the dial functions and the transports
// already returned keep the setting they were created with.
//
// When counting is disabled, no bytes are counted and no connection is wrapped.
func Enabled(v bool) {
	enabled.Store(v)
}

// IsEnabled reports whether counting is enabled (see [Enabled]).
func IsEnabled() bool {
	return enabled.Load()
}

// ErrNoOrganization is the error the dial functions fail with when the
// organization the bytes they send would be attributed to does not exist,
// because it has been deleted or it has never been created.
var ErrNoOrganization = errors.New("organization does not exist")

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
	// listening reports whether the organizations are known, that is whether
	// Listen has been called. Until it is, every organization is considered to
	// exist, because this package has no way to tell which ones do.
	listening bool
)

// Listen makes this package follow the organizations of st, so that the counter
// of an organization is discarded when the organization is deleted, and dialing
// on behalf of an organization that does not exist fails.
//
// It must be called once, at startup. Until it is called, every organization is
// considered to exist.
func Listen(st *state.State) {
	st.Freeze()
	st.AddListener(onCreateOrganization)
	st.AddListener(onDeleteOrganization)
	organizationsMu.Lock()
	for _, org := range st.Organizations() {
		if _, ok := organizations[org.ID]; !ok {
			organizations[org.ID] = &organization{}
		}
	}
	listening = true
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
// It fails with [ErrNoOrganization] if the organization does not exist, unless
// the organizations are not known yet, see [Listen].
//
// The dial functions resolve the organization once, when they are created, and
// keep the returned values, so that they do not have to take organizationsMu to
// establish a connection.
func resolve(organizationID string) (*organization, *prometheus.Counter, error) {
	organizationsMu.Lock()
	defer organizationsMu.Unlock()
	org, ok := organizations[organizationID]
	if !ok {
		if listening {
			return nil, nil, fmt.Errorf("countdial: %w: %s", ErrNoOrganization, organizationID)
		}
		org = &organization{}
		organizations[organizationID] = org
	}
	if org.egress == nil {
		org.egress = egressBytes.Register(organizationID)
	}
	return org, org.egress, nil
}

// Dial returns a dial function that dials with a plain net.Dialer, counting the
// bytes the connections it establishes send and attributing them to the
// organization with the given ID.
//
// If organizationID is empty, or counting is disabled (see [Enabled]), the
// returned function is a plain, unwrapped dialer and no bytes are counted.
// Otherwise, the returned function fails with [ErrNoOrganization] if the
// organization does not exist when it is called.
//
// Use [DialWith] instead to keep the dial options of an already configured
// dialer.
func Dial(organizationID string) DialFunc {
	return dialWith(organizationID, nil)
}

// DialWith returns a function that wraps a dial function, counting the bytes
// the connections it establishes send and attributing them to the organization
// with the given ID.
//
// Unlike [Dial], the connections are established by the wrapped dial function,
// which therefore keeps its own dial options, like its timeouts and its
// keep-alive. If the wrapped dial function is nil, a plain net.Dialer is used,
// as in [Dial].
//
// If organizationID is empty, or counting is disabled (see [Enabled]), the dial
// function is returned unwrapped and no bytes are counted. Otherwise, the
// returned function fails with [ErrNoOrganization] if the organization does not
// exist when it is called.
func DialWith(organizationID string) func(dial DialFunc) DialFunc {
	return func(dial DialFunc) DialFunc {
		return dialWith(organizationID, dial)
	}
}

type organizationKey struct{}

// WithOrganization returns a copy of ctx carrying the ID of the organization the
// bytes sent by the connections dialed with it are attributed to.
//
// Use it, together with [DialWithContext], when a client is shared by every
// organization and the organization is only known when the client is used, so
// that the dial function does not have to be fixed when the client is created.
//
// If organizationID is empty, or counting is disabled (see [Enabled]), ctx is
// returned unchanged.
func WithOrganization(ctx context.Context, organizationID string) context.Context {
	if !enabled.Load() || organizationID == "" {
		return ctx
	}
	return context.WithValue(ctx, organizationKey{}, organizationID)
}

// DialWithContext wraps a dial function, counting the bytes the connections it
// establishes send and attributing them to the organization carried by the
// context of each dial, set with [WithOrganization].
//
// Unlike [DialWith], the organization is not fixed when the dial function is
// created, so a single client can serve every organization.
//
// If the wrapped dial function is nil, a plain net.Dialer is used, as in
// [Dial]. If counting is disabled (see [Enabled]), the dial function is
// returned unwrapped and no bytes are counted. Otherwise, a dial whose context
// carries an organization that does not exist fails with [ErrNoOrganization].
func DialWithContext(dial DialFunc) DialFunc {
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}
	if !enabled.Load() {
		return dial
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		organizationID, _ := ctx.Value(organizationKey{}).(string)
		if organizationID == "" {
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

// Transport returns an HTTP transport that counts the bytes the requests made
// with it send, attributing them to the organization with the given ID.
//
// If organizationID is empty, or counting is disabled (see [Enabled]), base is
// returned unwrapped; otherwise the returned transport is a clone of base
// dialing with [Dial], so that base's timeouts and options are preserved, and
// its requests fail with [ErrNoOrganization] if the organization does not exist
// when they are made.
//
// A clone does not share the connection pool of base, so the caller should
// create one transport per organization and reuse it for all its requests,
// instead of creating one per request.
func Transport(base *http.Transport, organizationID string) http.RoundTripper {
	if !enabled.Load() || organizationID == "" {
		return base
	}
	t := base.Clone()
	t.DialContext = dialWith(organizationID, t.DialContext)
	return t
}

// dialWith is like [Dial], but the connections are established by dial instead
// of by a plain net.Dialer. If dial is nil, a plain net.Dialer is used.
//
// It allows counting the bytes sent by an already configured dialer, like the
// one of an http.Transport, preserving its timeouts and options.
func dialWith(organizationID string, dial DialFunc) DialFunc {
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}
	if !enabled.Load() || organizationID == "" {
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
			return nil, fmt.Errorf("countdial: %w: %s", ErrNoOrganization, organizationID)
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
