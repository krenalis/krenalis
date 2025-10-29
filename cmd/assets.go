// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"io/fs"
	"net/http"
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
	a.handler.ServeHTTP(w, r)
}
