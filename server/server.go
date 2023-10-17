//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package server

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"chichi/admin"
	"chichi/apis"
	"chichi/telemetry"
)

type Settings struct {
	Main struct {
		Host string
	}
	ESBuild struct {
		PrintWarningsOnStderr bool `yaml:"printWarningsOnStderr"`
	}
	PostgreSQL  apis.PostgreSQLConfig `yaml:"postgreSQL"`
	Transformer struct {
		Lambda LambdaConfig
		Local  LocalConfig
	}
	Telemetry struct {
		Enable bool
	}
}

type LambdaConfig struct {
	AccessKeyID     string `yaml:"accessKeyID"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	Region          string
	Role            string
	Node            struct {
		Runtime string
		Layer   string
	}
	Python struct {
		Runtime string
		Layer   string
	}
}

type LocalConfig struct {
	NodeExecutable   string `yaml:"nodeExecutable"`
	PythonExecutable string `yaml:"pythonExecutable"`
	FunctionsDir     string `yaml:"functionsDir"`
}

// Run runs the server.
// Cancel ctx to terminate the execution. If ctx is cancelled, Run does not
// return any error.
func Run(ctx context.Context, settings *Settings) error {

	if settings.Telemetry.Enable {
		err := telemetry.Init(ctx)
		if err != nil {
			return err
		}
	}

	config := apis.Config{
		PostgreSQL: settings.PostgreSQL,
	}

	// Choose the transformer setting.
	if settings.Transformer.Lambda.Node.Runtime != "" || settings.Transformer.Lambda.Python.Runtime != "" {
		config.Transformer = apis.LambdaConfig(settings.Transformer.Lambda)
	}
	if settings.Transformer.Local.NodeExecutable != "" || settings.Transformer.Local.PythonExecutable != "" {
		if config.Transformer != nil {
			return errors.New("cannot specify both the Lambda and the local transformer in the configuration (hint: check your 'config.yaml' file)")
		}
		config.Transformer = apis.LocalConfig(settings.Transformer.Local)
	}

	apis, err := apis.New(&config)
	if err != nil {
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
