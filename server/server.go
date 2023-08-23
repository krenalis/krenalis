//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package server

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"chichi/admin"
	"chichi/apis"
	"chichi/telemetry"
)

type Settings struct {
	Main struct {
		Host                         string
		PrintESBuildWarningsOnStderr bool
	}
	PostgreSQL apis.PostgreSQLConfig
	Telemetry  struct {
		Enable bool
	}
}

func Run(ctx context.Context, settings *Settings) error {

	if settings.Telemetry.Enable {
		err := telemetry.Init(ctx)
		if err != nil {
			return err
		}
	}

	apis, err := apis.New(&apis.Config{
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
	defer apis.Close()

	admin := admin.New(apis)

	apisServer := &apisServer{apis}

	addr := settings.Main.Host
	if addr == "" {
		addr = "127.0.0.1:9090"
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/admin/"):
			admin.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/api/v1/"):
			apis.ServeEvents(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			apisServer.ServeHTTP(w, r)
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
	certPem, err := filepath.Abs("cert.pem")
	if err != nil {
		return err
	}
	keyPem, err := filepath.Abs("key.pem")
	if err != nil {
		return err
	}

	exited := make(chan error)
	go func() {
		exited <- httpServer.ListenAndServeTLS(certPem, keyPem)
	}()

	select {
	case <-ctx.Done():
		err = httpServer.Shutdown(context.Background())
		if err != nil {
			return err
		}
		err = <-exited
	case err = <-exited:
	}
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
