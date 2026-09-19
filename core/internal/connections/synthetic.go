// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/dialer"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/synthetic"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

type syntheticS3 struct {
	s3       any
	scenario *synthetic.Scenario
}

func (s *syntheticS3) AbsolutePath(ctx context.Context, name string) (string, error) {
	_, err := synthetic.Path(name)
	if err != nil {
		return "", err
	}
	return s.s3.(fileStorageAbsolutePathConnection).AbsolutePath(ctx, name)
}

func (s *syntheticS3) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	return s.scenario.Reader(ctx, name)
}

func (s *syntheticS3) ServeUI(ctx context.Context, event string, settings json.Value, role connectors.Role) (*connectors.UI, error) {
	return s.s3.(uiHandlerConnection).ServeUI(ctx, event, settings, role)
}

// SupportsSynthetic reports the connector roles implemented in Synthetic mode.
func SupportsSynthetic(connector *state.Connector, role state.Role) bool {
	return supportsSynthetic(connector, role)
}

func supportsSynthetic(connector *state.Connector, role state.Role) bool {
	if connector == nil {
		return false
	}
	return role == state.Source &&
		(connector.Type == state.FileStorage && connector.Code == "s3" ||
			connector.Type == state.File && connector.Code == "csv")
}

// CheckConnector rejects unsupported operations before constructing a connector.
func (c *Connections) CheckConnector(workspace *state.Workspace, connector *state.Connector, role state.Role) error {
	if workspace == nil || connector == nil {
		return errors.New("workspace and connector are required")
	}
	if !workspace.Synthetic {
		return nil
	}
	if !supportsSynthetic(connector, role) {
		return errors.New("Synthetic workspace supports only S3 and CSV sources")
	}
	if c.synthetic == nil {
		return errors.New("Synthetic scenario is not available")
	}
	return nil
}

// SyntheticUnavailable reports whether this instance has no prepared scenario.
func (c *Connections) SyntheticUnavailable() bool {
	return c.synthetic == nil
}

func (c *Connections) newFileStorage(code string, role state.Role, settings connectors.SettingsStore, organization string, workspace *state.Workspace) (any, error) {
	err := c.CheckConnector(workspace, &state.Connector{Code: code, Type: state.FileStorage}, role)
	if err != nil {
		return nil, err
	}
	if workspace == nil || !workspace.Synthetic {
		return connectors.RegisteredFileStorage(code).New(&connectors.FileStorageEnv{
			Settings: settings, Dial: dialer.Dial(organization), DialWith: dialer.DialWith(organization),
		})
	}
	denied := func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("external S3 connection disabled in Synthetic mode")
	}
	s3, err := connectors.RegisteredFileStorage("s3").New(&connectors.FileStorageEnv{
		Settings: settings,
		Dial:     denied,
		DialWith: func(dialer.DialFunc) connectors.DialFunc { return denied },
	})
	if err != nil {
		return nil, err
	}
	return &syntheticS3{s3: s3, scenario: c.synthetic}, nil
}
