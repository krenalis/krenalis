// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	corePkg "github.com/krenalis/krenalis/core"
)

// healthChecker checks the dependencies of Krenalis.
type healthChecker interface {
	Health(ctx context.Context) []corePkg.HealthCheck
}

// allowGetOrHead reports whether the method of r is GET or HEAD. If it is not,
// it responds with 405 Method Not Allowed and returns false.
func allowGetOrHead(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == "GET" || r.Method == "HEAD" {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	return false
}

// newReadyzHandler returns the handler of the /readyz endpoint, which reports
// whether Krenalis can serve traffic. It responds 503 if a dependency is not
// available or if isShuttingDown is true.
func newReadyzHandler(checker healthChecker, isShuttingDown *atomic.Bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if !allowGetOrHead(w, r) {
			return
		}

		checks := checker.Health(r.Context())
		checks = append(checks, corePkg.HealthCheck{Name: "shutdown", OK: !isShuttingDown.Load()})
		isReady := true
		var b strings.Builder
		for _, check := range checks {
			if check.OK {
				b.WriteString("[+]" + check.Name + " ok\n")
			} else {
				b.WriteString("[-]" + check.Name + " failed\n")
				isReady = false
			}
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		switch {
		case !isReady:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(b.String() + "readyz check failed"))
		case r.URL.Query().Has("verbose"):
			_, _ = w.Write([]byte(b.String() + "readyz check passed"))
		default:
			_, _ = w.Write([]byte("ok"))
		}

	}
}

// serveLivez serves the /livez endpoint, which reports that the process is
// serving requests. It does not check any dependency.
func serveLivez(w http.ResponseWriter, r *http.Request) {
	if !allowGetOrHead(w, r) {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("ok"))
}
