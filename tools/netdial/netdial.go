// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package netdial provides the dial function Krenalis passes to connectors and
// warehouses to establish outbound network connections.
//
// When Prometheus metrics are enabled (see [SetEnabled]), the dial function
// returned by [Dial] attributes the bytes read and written by the connections
// it establishes to the given organization, exposing them as the
// krenalis_organization_network_bytes_total Prometheus counter. Otherwise, it
// behaves like a plain net.Dialer.
package netdial

import (
	"context"
	"net"
	"sync"
	"sync/atomic"

	"github.com/krenalis/krenalis/tools/prometheus"
)

// DialFunc is the type of the dial functions returned by [Dial]. It matches,
// among others, pgconn.Config.DialFunc.
type DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error)

// networkBytes is the Prometheus counter exposing the bytes transferred by
// each organization, partitioned by direction ("ingress" or "egress").
var networkBytes = prometheus.RegisterCounterVec(
	"krenalis_organization_network_bytes_total",
	"Total bytes transferred per organization",
	[]string{"organization", "direction"},
)

// enabled reports whether connectors' network traffic must be attributed to
// organizations and recorded as Prometheus metrics. It is false by default,
// so that [Dial] returns a plain, unwrapped dialer unless explicitly enabled.
var enabled atomic.Bool

// SetEnabled enables or disables the attribution of connectors' network traffic
// to organizations. It should be called once, at startup, before any connector
// dials a connection.
//
// When disabled, [Dial] returns a plain, unwrapped dialer.
func SetEnabled(v bool) {
	enabled.Store(v)
}

// counterPair holds the ingress and egress counters for an organization.
type counterPair struct {
	ingress *prometheus.Counter
	egress  *prometheus.Counter
}

var (
	countersMu sync.Mutex
	counters   = map[string]*counterPair{} // organization ID -> counters
)

// countersFor returns the ingress/egress counters for the given organization,
// registering them the first time the organization is seen.
func countersFor(organizationID string) *counterPair {
	countersMu.Lock()
	defer countersMu.Unlock()
	c, ok := counters[organizationID]
	if !ok {
		c = &counterPair{
			ingress: networkBytes.Register(organizationID, "ingress"),
			egress:  networkBytes.Register(organizationID, "egress"),
		}
		counters[organizationID] = c
	}
	return c
}

// Dial returns the dial function Krenalis passes to a connector to establish
// its outbound network connections, attributing them to the organization with
// the given ID.
//
// If organizationID is empty, or Prometheus metrics are disabled (see
// [SetEnabled]), the returned function is a plain, unwrapped dialer; otherwise
// every connection it establishes has the bytes it reads and writes recorded
// as the krenalis_organization_network_bytes_total Prometheus counter,
// labeled by organization and direction ("ingress" for bytes read, "egress"
// for bytes written).
func Dial(organizationID string) DialFunc {
	var d net.Dialer
	if organizationID == "" || !enabled.Load() {
		return d.DialContext
	}
	c := countersFor(organizationID)
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := d.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return &instrumentedConn{Conn: conn, counters: c}, nil
	}
}

// instrumentedConn wraps a net.Conn, recording the bytes it reads and writes
// into its organization's counters.
type instrumentedConn struct {
	net.Conn
	counters *counterPair
}

func (c *instrumentedConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		c.counters.ingress.Add(n)
	}
	return n, err
}

func (c *instrumentedConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		c.counters.egress.Add(n)
	}
	return n, err
}
