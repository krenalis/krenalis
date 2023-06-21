//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"chichi/admin"
	"chichi/apis"
)

type Settings struct {
	Main struct {
		Host                         string
		PrintESBuildWarningsOnStderr bool
	}
	PostgreSQL apis.PostgreSQLConfig
}

func Run(ctx context.Context, settings *Settings) error {

	apis, err := apis.New(ctx, &apis.Config{
		PostgreSQL: settings.PostgreSQL,
	})
	if err != nil {
		if err == context.Canceled {
			select {
			case <-ctx.Done():
				os.Exit(0)
			}
		}
		return err
	}
	admin := admin.New(apis)

	addr := settings.Main.Host
	if addr == "" {
		addr = "127.0.0.1:9090"
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/admin/"):
			admin.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/api/"):
			apis.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/webhook/"):
			apis.ServeWebhook(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/trace-events-script/"):
			http.FileServer(http.Dir(".")).ServeHTTP(w, r)
			return
		default:
			http.NotFound(w, r)
			return
		}
	})
	httpServer := http.Server{
		Addr:    addr,
		Handler: handler,
	}
	go func() {
		<-ctx.Done()
		err := httpServer.Shutdown(context.Background())
		if err != nil {
			log.Printf("[error] shutting down HTTP server: %s", err)
		}
	}()
	certPem, err := filepath.Abs("cert.pem")
	if err != nil {
		return err
	}
	keyPem, err := filepath.Abs("key.pem")
	if err != nil {
		return err
	}
	err = httpServer.ListenAndServeTLS(certPem, keyPem)
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
