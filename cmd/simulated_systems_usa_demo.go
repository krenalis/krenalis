// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/errors"
)

// SimulatedSystemUSAPhoto serves a local rendition to an Admin session.
func (workspace workspace) SimulatedSystemUSAPhoto(w http.ResponseWriter, r *http.Request) (any, error) {

	if _, exists := r.Header["Authorization"]; exists {
		return nil, errors.Unauthorized("USA demo photos require Admin session authentication")
	}
	org, _, _, err := workspace.authenticateAdminRequest(r)
	if err != nil {
		return nil, err
	}
	ws, err := org.Workspace(r.PathValue("workspace"))
	if err != nil {
		return nil, err
	}
	data, err := ws.SimulatedSystemUSAPhoto(r.Context(), strings.TrimSuffix(r.PathValue("id"), ".webp"))
	if err != nil {
		return nil, err
	}
	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodHead {
		_, err = w.Write(data)
	}

	return nil, err
}

// SimulatedSystemUSAPreview returns a bounded preview of stored demo records.
func (workspace workspace) SimulatedSystemUSAPreview(_ http.ResponseWriter, r *http.Request) (any, error) {

	ws, err := workspace.authenticateSimulatedSystemRequest(r)
	if err != nil {
		return nil, err
	}
	records, err := ws.SimulatedSystemUSAPreview(r.Context(), r.PathValue("id"))
	if err != nil {
		return nil, err
	}

	return map[string]any{"records": records}, nil
}
