// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"io/fs"
	"net/http"
)

type admin struct {
	assets *assetsHandler
}

func newAdmin(assetsFs fs.FS) (*admin, error) {
	assets, err := newAssetsHandler(assetsFs)
	if err != nil {
		return nil, err
	}
	admin := &admin{
		assets: assets,
	}
	return admin, nil
}

func (a *admin) Close() {
	a.assets.Close()
}

func (a *admin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.assets.ServeHTTP(w, r)
}
