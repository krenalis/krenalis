// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
)

type connectionSettingStore struct {
	state      *state.State
	connection *state.Connection
}

func newConnectionSettingStore(st *state.State, c *state.Connection) *connectionSettingStore {
	return &connectionSettingStore{state: st, connection: c}
}

func (store *connectionSettingStore) Load(ctx context.Context, dst any) error {
	settings, err := store.connection.Settings(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(settings, dst)
}

func (store *connectionSettingStore) Store(ctx context.Context, src any) error {
	return store.storeWithSimulatedAccount(ctx, src, nil)
}

func (store *connectionSettingStore) storeWithSimulatedAccount(ctx context.Context, src any, simulatedAccount *string) error {
	s, err := json.Marshal(src)
	if err != nil {
		return err
	}
	if len(s) > maxSettingsLen && utf8.RuneCount(s) > maxSettingsLen {
		return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
	}
	n := state.SetConnectionSettings{
		Connection:       store.connection.ID,
		SimulatedAccount: simulatedAccount,
	}
	n.Settings, err = store.connection.EncryptSettings(ctx, s)
	if err != nil {
		return err
	}
	err = store.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		query := "UPDATE connections SET settings = $1 WHERE id = $2 AND settings <> $1"
		args := []any{n.Settings, n.Connection}
		if n.SimulatedAccount != nil {
			var value any
			if *n.SimulatedAccount != "" {
				value = *n.SimulatedAccount
			}
			query = "UPDATE connections SET settings = $1, simulated_account = $2 " +
				"WHERE id = $3 AND (settings <> $1 OR simulated_account IS DISTINCT FROM $2)"
			args = []any{n.Settings, value, n.Connection}
		}
		result, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		return n, err
	})
	return err
}

type pipelineSettingStore struct {
	state    *state.State
	pipeline *state.Pipeline
}

func newPipelineSettingStore(st *state.State, p *state.Pipeline) *pipelineSettingStore {
	return &pipelineSettingStore{state: st, pipeline: p}
}

func (store *pipelineSettingStore) Load(ctx context.Context, dst any) error {
	return json.Unmarshal(store.pipeline.FormatSettings, dst)
}

func (store *pipelineSettingStore) Store(ctx context.Context, src any) error {
	s, err := json.Marshal(src)
	if err != nil {
		return err
	}
	if len(s) > maxSettingsLen && utf8.RuneCount(s) > maxSettingsLen {
		return fmt.Errorf("format settings is longer than %d runes", maxSettingsLen)
	}
	n := state.SetPipelineFormatSettings{
		Pipeline: store.pipeline.ID,
		Settings: s,
	}
	err = store.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE pipelines SET format_settings = $1 WHERE id = $2", n.Settings, n.Pipeline)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		return n, nil
	})
	return err
}

type uiSettingStore struct {
	settings json.Value
}

func newUISettingStore(settings json.Value) *uiSettingStore {
	return &uiSettingStore{settings: settings}
}

func (store *uiSettingStore) Load(_ context.Context, out any) error {
	if store.settings == nil {
		return nil
	}
	return store.settings.Unmarshal(out)
}

func (store *uiSettingStore) Store(ctx context.Context, src any) error {
	s, err := json.Marshal(src)
	if err != nil {
		return err
	}
	if len(s) > maxSettingsLen && utf8.RuneCount(s) > maxSettingsLen {
		return fmt.Errorf("format settings is longer than %d runes", maxSettingsLen)
	}
	store.settings = s
	return nil
}
