// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"net/http"

	"github.com/krenalis/krenalis/core"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// ExplainProfileFilter returns the filter's measured outcomes for one profile.
func (workspace workspace) ExplainProfileFilter(_ http.ResponseWriter, r *http.Request) (any, error) {
	ws, err := workspace.admitWorkspaceRequest(r, x1)
	if err != nil {
		return nil, err
	}
	kpid := r.PathValue("kpid")
	if _, ok := types.NormalizeUUID(kpid); !ok {
		return nil, errors.BadRequest("invalid KPID")
	}
	q := r.URL.Query()
	text := q.Get("filter")
	if len(text) == 0 || len(text) > maxRequestSize {
		return nil, errors.BadRequest("filter is required and must be at most 500 KiB")
	}
	var filter *core.Filter
	err = json.Unmarshal([]byte(text), &filter)
	if err != nil {
		return nil, errors.BadRequest("invalid filter: %s", err)
	}
	if q.Has("at") && q.Get("at") == "" {
		return nil, errors.BadRequest("invalid at")
	}
	return ws.ExplainProfileFilter(r.Context(), kpid, filter, q.Get("at"))
}
