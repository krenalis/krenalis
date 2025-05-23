//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// This code implements a tunnel for Sentry errors, which are sent first to
// Meergo and then forwarded to Sentry.
//
// See the documentation here:
// https://docs.sentry.io/platforms/javascript/troubleshooting/#using-the-tunnel-option.

// Configuration for forwarding events to Sentry.
const (
	sentryAdminHost      = "o4509282180136960.ingest.de.sentry.io"
	sentryAdminProjectID = "4509292547211344"
)

var sentryUpstreamAdminURL = fmt.Sprintf("https://%s/api/%s/envelope/", sentryAdminHost, sentryAdminProjectID)

// During a time slot, each client can send a limited number of requests to be
// forwarded to Sentry. After that limit, the requests are ignored.
var (
	timeSlotDuration    = time.Minute
	timeSlotMaxRequests = 300 // an avg. of 5 requests per second seems more than enough.
)

// debugTunnel, if enabled, prints debug tunnel information to stderr.
const debugTunnel = false

type telemetryErrorTunnel struct {
	done   chan bool
	ticker *time.Ticker

	mu            sync.Mutex
	requestsPerIP map[string]int
}

// newTelemetryErrorTunnel instantiates a new telemetryErrorsTunnel, which can
// be used to forward error reporting from a client to Sentry.
func newTelemetryErrorTunnel() *telemetryErrorTunnel {
	t := &telemetryErrorTunnel{
		done:          make(chan bool),
		ticker:        time.NewTicker(timeSlotDuration),
		requestsPerIP: map[string]int{},
	}
	go func() {
		for {
			select {
			case <-t.done:
				return
			case <-t.ticker.C:
				t.clearRequestsPerIP()
			}
		}
	}()
	return t
}

// ServeHTTP receives an HTTP request, representing an error report to send to
// Sentry, and forwards it to Sentry.
//
// There is a maximum number of requests that a client can request to forward to
// Sentry in a given time slot; if this limit is exceeded, forward requests are
// silently ignored.
func (t *telemetryErrorTunnel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP, ok := normalizedIP(r)
	if !ok {
		return
	}
	err := t.increaseRequestsPerIP(clientIP)
	if err != nil {
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		debugTunnelInfo("an error occurred while increasing counter for IP %q: %s", clientIP, err)
		return
	}
	contentType := r.Header.Get("Content-type")
	_, err = http.Post(sentryUpstreamAdminURL, contentType, r.Body)
	if err != nil {
		debugTunnelInfo("error while issuing HTTP POST request to Sentry: %s", err)
		return
	}
	debugTunnelInfo("forwarded POST request to %s from client %q", sentryUpstreamAdminURL, clientIP)
}

func (t *telemetryErrorTunnel) Close() {
	t.done <- true
}

// increaseRequestsPerIP increments the counter of requests made from the given
// IP.
//
// If the number of requests exceeds the maximum allowed, the counter is not
// incremented and an error is returned.
func (t *telemetryErrorTunnel) increaseRequestsPerIP(ip string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	count := t.requestsPerIP[ip]
	if count == timeSlotMaxRequests {
		return fmt.Errorf("too many requests for IP %q", ip)
	}
	t.requestsPerIP[ip] = count + 1
	debugTunnelInfo("requestsPerIP[%s] = %d", ip, count+1)
	return nil
}

func (t *telemetryErrorTunnel) clearRequestsPerIP() {
	debugTunnelInfo("clearing 'requestsPerIP'")
	t.mu.Lock()
	clear(t.requestsPerIP)
	t.mu.Unlock()
}

func debugTunnelInfo(format string, a ...any) {
	if !debugTunnel {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05.000000000")
	_, _ = fmt.Fprintf(os.Stderr, "[tunnel debug info] ["+timestamp+"] "+format+"\n", a...)
}

func normalizedIP(r *http.Request) (string, bool) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		debugTunnelInfo("cannot split IP address host and port: %s", err)
		return "", false
	}
	userIP := net.ParseIP(ip)
	if userIP == nil {
		debugTunnelInfo("cannot parse IP: %s", err)
		return "", false
	}
	return userIP.String(), true
}
