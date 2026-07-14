// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package netdial provides the dial function Krenalis passes to connectors and
// warehouses to establish outbound network connections.
//
// When Prometheus metrics are enabled (see [Enabled]), the dial function
// returned by [Dial] attributes the bytes written by the connections it
// establishes to the given organization, exposing them as the
// krenalis_organization_network_egress_bytes_total Prometheus counter.
// Otherwise, it behaves like a plain net.Dialer.
package netdial

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/krenalis/krenalis/tools/prometheus"
)

// DialFunc is the type of the dial functions returned by [Dial]. It matches,
// among others, pgconn.Config.DialFunc.
type DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// egressBytes is the Prometheus counter exposing the bytes written by each
// organization, that is its outbound (egress) traffic. Inbound traffic is not
// counted.
var egressBytes = prometheus.RegisterCounterVec(
	"krenalis_organization_network_egress_bytes_total",
	"Total bytes sent per organization",
	[]string{"organization"},
)

// enabled reports whether connectors' network traffic must be attributed to
// organizations and recorded as Prometheus metrics. It is false by default,
// so that [Dial] returns a plain, unwrapped dialer unless explicitly enabled.
var enabled atomic.Bool

// Enabled enables or disables the attribution of connectors' network traffic to
// organizations. It should be called once, at startup, before any connector
// dials a connection.
//
// When disabled, [Dial] returns a plain, unwrapped dialer.
func Enabled(v bool) {
	enabled.Store(v)
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

// Dial returns the dial function Krenalis passes to a connector to establish
// its outbound network connections, attributing them to the organization with
// the given ID.
//
// If organizationID is empty, or Prometheus metrics are disabled (see
// [Enabled]), the returned function is a plain, unwrapped dialer; otherwise
// every connection it establishes has the bytes it writes recorded as the
// krenalis_organization_network_egress_bytes_total Prometheus counter, labeled
// by organization. The bytes it reads are not counted.
func Dial(organizationID string) DialFunc {
	return dialWith(organizationID, nil)
}

// DialWith returns the function Krenalis passes to a connector that has its own
// dialer, to count the bytes the dialer sends, attributing them to the
// organization with the given ID.
//
// Unlike [Dial], which replaces the connector's dialer with a plain one, the
// returned function wraps the dial function it is given, so that the connector
// keeps its own dial options, like its timeouts and its keep-alive. If it is
// given a nil dial function, a plain net.Dialer is used, as in [Dial].
func DialWith(organizationID string) func(dial DialFunc) DialFunc {
	return func(dial DialFunc) DialFunc {
		return dialWith(organizationID, dial)
	}
}

// Transport returns the transport Krenalis uses for the HTTP requests of the
// organization with the given ID, attributing to it the bytes they send.
//
// If organizationID is empty, or Prometheus metrics are disabled (see
// [Enabled]), base is returned unwrapped; otherwise the returned transport is a
// clone of base dialing with [Dial], so that base's timeouts and options are
// preserved.
//
// The returned transport has its own connection pool, so callers should reuse
// it for all the requests of the organization.
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
