//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"io/fs"
	"net/http"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/telemetry"
)

type UI struct {
	apis   *apis.APIs
	assets *assetsHandler
}

func newUI(apis *apis.APIs, assetsFs fs.FS) (*UI, error) {
	assets, err := newAssetsHandler(assetsFs)
	if err != nil {
		return nil, err
	}
	ui := &UI{
		apis:   apis,
		assets: assets,
	}
	return ui, nil
}

func (ui *UI) Close() {
	ui.assets.Close()
}

func (ui *UI) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, s := telemetry.TraceSpan(r.Context(), "ui.ServeHTTP", "urlPath", r.URL.Path)
	defer s.End()
	telemetry.IncrementCounter(ctx, "ui.ServeHTTP", 1)

	ui.assets.ServeHTTP(w, r)
}
