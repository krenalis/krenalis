//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package testutils

import (
	"context"
	"net/http"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/connectors/httpclient"
)

// NewAppConnectorForTests returns an instance of the connector with the
// specified name for testing purposes.
// Settings are the connector settings encoded in JSON; they will be passed
// as-is to the connector instance.
// If a connector with the specified name hasn't been registered, this method
// panics.
func NewAppConnectorForTests(connectorName string, settings []byte) (any, error) {
	registeredApp := meergo.RegisteredApp(connectorName)
	httpClient := httpclient.New(nil, http.DefaultTransport).Client("", "", registeredApp.BackoffPolicy)
	app, err := registeredApp.New(&meergo.AppConfig{
		Settings:    settings,
		SetSettings: func(ctx context.Context, b []byte) error { return nil },
		HTTPClient:  httpClient,
	})
	if err != nil {
		return nil, err
	}
	return app, nil
}
