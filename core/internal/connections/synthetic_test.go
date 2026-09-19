// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"strings"
	"sync"
	"testing"

	"github.com/krenalis/krenalis/connectors/s3"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/synthetic"
	"github.com/krenalis/krenalis/tools/json"
)

func TestSyntheticFileStorageSelection(t *testing.T) {
	settings := newUISettingStore(nil)
	normal := &Connections{}
	original, err := normal.newFileStorage("s3", state.Source, settings, "test", &state.Workspace{})
	if err != nil {
		t.Fatalf("expected normal S3 storage, got %v", err)
	}
	if _, ok := original.(*s3.S3); !ok {
		t.Fatalf("expected real S3 storage, got %T", original)
	}

	active := &Connections{synthetic: &synthetic.Scenario{}}
	workspace := &state.Workspace{Synthetic: true}
	ordinary, err := active.newFileStorage("s3", state.Source, settings, "test", &state.Workspace{})
	if err != nil {
		t.Fatalf("expected real S3 storage with scenario present, got %v", err)
	}
	if _, ok := ordinary.(*s3.S3); !ok {
		t.Fatalf("expected real S3 storage, got %T", ordinary)
	}
	if !SupportsSynthetic(&state.Connector{Code: "csv", Type: state.File}, state.Source) {
		t.Fatalf("expected CSV source format to be supported, got rejected")
	}
	if SupportsSynthetic(&state.Connector{Code: "csv", Type: state.File}, state.Destination) ||
		SupportsSynthetic(&state.Connector{Code: "s3", Type: state.Database}, state.Source) {
		t.Fatalf("expected unsupported connector roles to be rejected, got supported")
	}
	if err := active.CheckConnector(workspace, &state.Connector{Code: "csv", Type: state.File}, state.Source); err != nil {
		t.Fatalf("expected CSV source format, got %v", err)
	}
	if err := normal.CheckConnector(workspace, &state.Connector{Code: "s3", Type: state.FileStorage}, state.Source); err == nil {
		t.Fatalf("expected unavailable scenario error, got nil")
	}
	_, err = normal.newFileStorage("s3", state.Source, settings, "test", workspace)
	if err == nil {
		t.Fatalf("expected unavailable scenario without real S3 fallback, got nil")
	}
	for _, test := range []struct {
		code string
		role state.Role
	}{
		{"sftp", state.Source}, {"filesystem", state.Source}, {"s3", state.Destination},
	} {
		err := active.CheckConnector(workspace, &state.Connector{Code: test.code, Type: state.FileStorage}, test.role)
		if err == nil {
			t.Fatalf("expected rejected %s role %d, got nil", test.code, test.role)
		}
		_, err = active.newFileStorage(test.code, test.role, settings, "test", workspace)
		if err == nil {
			t.Fatalf("expected rejected construction of %s role %d, got nil", test.code, test.role)
		}
		connector := &state.Connector{Code: test.code, Type: state.FileStorage}
		conf := &ConnectorConfig{Role: test.role, Organization: "test", Workspace: workspace}
		_, err = active.ServeConnectorUI(t.Context(), connector, conf, "load", nil)
		if err == nil {
			t.Fatalf("expected rejected UI for %s role %d, got nil", test.code, test.role)
		}
		_, err = active.UpdatedSettings(t.Context(), connector, conf, json.Value(`{}`))
		if err == nil {
			t.Fatalf("expected rejected settings for %s role %d, got nil", test.code, test.role)
		}
	}
	inner, err := active.newFileStorage("s3", state.Source, settings, "test", workspace)
	if err != nil {
		t.Fatalf("expected adapted S3 storage, got %v", err)
	}
	adapter := inner.(*syntheticS3)
	value := json.Value(`{"accessKeyID":"AAAAAAAAAAAAAAAAAAAA","secretAccessKey":"BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB","region":"us-east-1","bucket":"demo"}`)
	_, err = adapter.ServeUI(t.Context(), "save", value, 1)
	if err != nil {
		t.Fatalf("expected S3 settings save, got %v", err)
	}
	path, err := adapter.AbsolutePath(t.Context(), "/customers-a.csv")
	if err != nil || path != "s3://demo/customers-a.csv" {
		t.Fatalf("expected S3 absolute path, got %q, %v", path, err)
	}
	_, err = adapter.AbsolutePath(t.Context(), "missing.csv")
	if err == nil {
		t.Fatal("expected unknown path rejection, got nil")
	}
	_, _, err = adapter.Reader(t.Context(), "missing.csv")
	if err == nil {
		t.Fatal("expected unknown reader path rejection, got nil")
	}
	invalid := json.Value(strings.Replace(string(value), "AAAAAAAAAAAAAAAAAAAA", "short", 1))
	_, err = adapter.ServeUI(t.Context(), "save", invalid, 1)
	if err == nil {
		t.Fatal("expected S3 credential validation, got nil")
	}
}

func TestSyntheticSelectionConcurrent(t *testing.T) {
	active := &Connections{synthetic: &synthetic.Scenario{}}
	workspaces := []*state.Workspace{{}, {Synthetic: true}}
	var group sync.WaitGroup
	for _, workspace := range workspaces {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 20 {
				inner, err := active.newFileStorage("s3", state.Source, newUISettingStore(nil), "test", workspace)
				if err != nil {
					t.Errorf("expected selected S3 storage, got %v", err)
					return
				}
				if workspace.Synthetic {
					if _, ok := inner.(*syntheticS3); !ok {
						t.Errorf("expected Synthetic S3, got %T", inner)
					}
				} else if _, ok := inner.(*s3.S3); !ok {
					t.Errorf("expected real S3, got %T", inner)
				}
			}
		}()
	}
	group.Wait()
}
