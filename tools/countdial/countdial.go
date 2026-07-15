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
// disabled, the dial functions returned by [Dial] and [DialWith], and the
// transport returned by [Transport], establish the connections as they would
// without this package, with no overhead.
package countdial

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

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
//
// Since the dial functions are per organization, a caller that would otherwise
// keep one client per organization can use it to keep a single shared client
// when counting is disabled.
func IsEnabled() bool {
	return enabled.Load()
}

var (
	countersMu sync.Mutex
	counters   = map[string]*prometheus.Counter{} // organization ID -> egress counter
)

// counterFor returns the egress counter for the given organization, registering
// it the first time the organization is seen.
func counterFor(organizationID string) *prometheus.Counter {
	countersMu.Lock()
	defer countersMu.Unlock()
	c, ok := counters[organizationID]
	if !ok {
		c = egressBytes.Register(organizationID)
		counters[organizationID] = c
	}
	return c
}

// Dial returns a dial function that dials with a plain net.Dialer, counting the
// bytes the connections it establishes send and attributing them to the
// organization with the given ID.
//
// If organizationID is empty, or counting is disabled (see [Enabled]), the
// returned function is a plain, unwrapped dialer and no bytes are counted.
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
// function is returned unwrapped and no bytes are counted.
func DialWith(organizationID string) func(dial DialFunc) DialFunc {
	return func(dial DialFunc) DialFunc {
		return dialWith(organizationID, dial)
	}
}

// Transport returns an HTTP transport that counts the bytes the requests made
// with it send, attributing them to the organization with the given ID.
//
// If organizationID is empty, or counting is disabled (see [Enabled]), base is
// returned unwrapped; otherwise the returned transport is a clone of base
// dialing with [Dial], so that base's timeouts and options are preserved.
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
	c := counterFor(organizationID)
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
