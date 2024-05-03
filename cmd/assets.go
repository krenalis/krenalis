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

	"github.com/open2b/chichi/telemetry"
)

type assets struct {
	handler *assetsHandler
}

func newAssets(assetsFs fs.FS) (*assets, error) {
	handler, err := newAssetsHandler(assetsFs)
	if err != nil {
		return nil, err
	}
	a := &assets{
		handler: handler,
	}
	return a, nil
}

func (a *assets) Close() {
	a.handler.Close()
}

func (a *assets) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, s := telemetry.TraceSpan(r.Context(), "ui.ServeHTTP", "urlPath", r.URL.Path)
	defer s.End()
	telemetry.IncrementCounter(ctx, "ui.ServeHTTP", 1)

	a.handler.ServeHTTP(w, r)
}
