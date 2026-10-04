// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package simulated provides connectors for simulated systems.
package simulated

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// ApplicationCode is the Simulated Application connector code.
const ApplicationCode = "simulated-application"

const recordsPageSize = 1000

func init() {
	connectors.RegisterApplication(connectors.ApplicationSpec{
		Code:       ApplicationCode,
		Label:      "Simulated Application",
		Categories: connectors.CategoryTesting,
		AsSource: &connectors.AsApplicationSource{
			Targets:     connectors.TargetUser,
			HasSettings: true,
		},
		AsDestination: &connectors.AsApplicationDestination{
			Targets:     connectors.TargetUser,
			SendingMode: connectors.Server,
			HasSettings: true,
		},
	}, NewApplication)
}

// Account is a selectable simulated system.
type Account struct {
	ID   string
	Name string
}

// Store provides the simulated-system context for one application instance.
type Store struct {
	workspace        string
	account          string
	state            *state.State
	store            *datastore.Store
	writerGeneration *int64
}

// NewStore returns a store for a workspace and optional bound account.
func NewStore(workspace, account string, state *state.State, store *datastore.Store) (*Store, error) {
	if workspace == "" {
		return nil, fmt.Errorf("simulated application workspace is missing")
	}
	if state == nil {
		return nil, fmt.Errorf("simulated application state is missing")
	}
	if store == nil {
		return nil, fmt.Errorf("simulated application datastore is missing")
	}
	if _, ok := state.Workspace(workspace); !ok {
		return nil, fmt.Errorf("simulated application workspace does not exist")
	}
	result := &Store{workspace: workspace, account: account, state: state, store: store}
	if account != "" {
		if err := result.validateAccount(account); err != nil {
			return nil, connectors.NewInvalidSettingsError(err.Error())
		}
	}
	return result, nil
}

// Accounts returns copies of the available simulated systems in the workspace.
func (store *Store) Accounts() ([]Account, error) {
	workspace, ok := store.state.Workspace(store.workspace)
	if !ok {
		return nil, fmt.Errorf("simulated application workspace does not exist")
	}
	accounts := workspace.SimulatedSystems()
	result := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Status == state.SimulatedSystemAvailable {
			result = append(result, Account{ID: account.ID, Name: account.Name})
		}
	}
	return result, nil
}

// AuthorizeWriter binds a store to its admitted Destination generation.
func (store *Store) AuthorizeWriter(generation int64) error {
	if generation <= 0 || generation%2 != 0 {
		return fmt.Errorf("invalid simulated application writer generation")
	}
	store.writerGeneration = &generation
	return nil
}

// Records returns one page of records for the bound simulated system.
func (store *Store) Records(ctx context.Context, query datastore.SimulatedRecordQuery) (datastore.SimulatedRecordPage, error) {
	if store.account == "" {
		return datastore.SimulatedRecordPage{}, connectors.NewInvalidSettingsError("simulated application account is not configured")
	}
	return store.store.SimulatedSystemRecords(ctx, store.account, query)
}

func (store *Store) recordsByID(ctx context.Context, ids []string) (map[string]json.Value, error) {
	if store.account == "" {
		return nil, connectors.NewInvalidSettingsError("simulated application account is not configured")
	}
	return store.store.SimulatedSystemRecordsByID(ctx, store.account, ids)
}

func (store *Store) writeRecords(ctx context.Context, creates, updates []datastore.SimulatedSystemRecordWrite) (datastore.SimulatedSystemRecordWriteResult, error) {
	if store.account == "" {
		return datastore.SimulatedSystemRecordWriteResult{}, connectors.NewInvalidSettingsError("simulated application account is not configured")
	}
	if store.writerGeneration == nil {
		return datastore.SimulatedSystemRecordWriteResult{}, fmt.Errorf("simulated application writer has not been admitted")
	}
	ctx, err := datastore.WithSimulatedSystemWriter(ctx, store.account, *store.writerGeneration)
	if err != nil {
		return datastore.SimulatedSystemRecordWriteResult{}, err
	}
	return store.store.WriteSimulatedSystemRecords(ctx, store.account, creates, updates)
}

type settings struct {
	Account string `json:"account"`
}

// Application is the Simulated Application connector.
type Application struct {
	env             *connectors.ApplicationEnv
	store           *Store
	selectedAccount string
}

// NewApplication returns a Simulated Application connector.
func NewApplication(env *connectors.ApplicationEnv) (*Application, error) {
	store, ok := env.Extra.(*Store)
	if !ok || store == nil {
		return nil, fmt.Errorf("simulated application store is missing")
	}
	return &Application{env: env, store: store}, nil
}

// RecordSchema returns the fixed source schema for the specified target.
func (*Application) RecordSchema(_ context.Context, target connectors.Targets, role connectors.Role) (types.Type, error) {
	if target != connectors.TargetUser {
		return types.Type{}, fmt.Errorf("simulated application target is not supported")
	}
	if role == connectors.Destination {
		return destinationSchema, nil
	}
	if role != connectors.Source {
		return types.Type{}, fmt.Errorf("simulated application role is not supported")
	}
	return sourceSchema, nil
}

// Records returns one page of source records for the specified target.
func (app *Application) Records(ctx context.Context, target connectors.Targets, updatedAt time.Time, cursor string, schema types.Type) ([]connectors.Record, string, error) {
	if target != connectors.TargetUser {
		return nil, "", fmt.Errorf("simulated application target is not supported")
	}
	after, err := datastore.ParseSimulatedRecordsCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	page, err := app.store.Records(ctx, datastore.SimulatedRecordQuery{Since: updatedAt, After: after, Limit: recordsPageSize})
	if err != nil {
		return nil, "", err
	}
	// appRecords projects the requested schema after receiving the complete document.
	records := make([]connectors.Record, len(page.Records))
	for i, simulatedRecord := range page.Records {
		attributes, err := types.Decode[map[string]any](bytes.NewReader(simulatedRecord.Attributes), sourceSchema)
		if err != nil {
			return nil, "", fmt.Errorf("simulated application record %q is invalid: %w", simulatedRecord.ID, err)
		}
		customerID, ok := attributes["customer_id"].(string)
		if !ok || customerID != simulatedRecord.ID {
			return nil, "", fmt.Errorf("simulated application record %q has a customer_id that does not match its external ID", simulatedRecord.ID)
		}
		records[i] = connectors.Record{
			ID:         simulatedRecord.ID,
			Attributes: attributes,
			UpdatedAt:  simulatedRecord.UpdatedAt,
		}
	}
	if page.HasMore {
		return records, page.Next.String(), nil
	}
	return records, "", io.EOF
}

// ServeUI serves the Simulated Application account selector.
func (app *Application) ServeUI(ctx context.Context, event string, input json.Value, role connectors.Role) (*connectors.UI, error) {

	if app.store.account != "" {
		if event == "save" {
			return nil, connectors.NewInvalidSettingsError("simulated data store used by this connection cannot be changed")
		}
		if event != "load" {
			return nil, connectors.ErrUIEventNotExist
		}
		accounts, err := app.store.Accounts()
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			if account.ID == app.store.account {
				return &connectors.UI{Fields: []connectors.Component{
					&connectors.Text{Label: "Simulated data store", Text: account.Name, Role: connectors.Both},
				}, Settings: json.Value("{}")}, nil
			}
		}
		return nil, connectors.NewInvalidSettingsError("simulated data store is not available in this workspace")
	}

	workspace, ok := app.store.state.Workspace(app.store.workspace)
	if !ok {
		return nil, fmt.Errorf("simulated application workspace does not exist")
	}
	roleName := strings.ToLower(role.String())
	usedAccounts := map[string]bool{}
	for _, connection := range workspace.Connections() {
		if connection.Role == state.Role(role) {
			if account, ok := connection.SimulatedSystem(); ok {
				usedAccounts[account.ID] = true
			}
		}
	}

	var s settings
	switch event {
	case "load":
		err := app.env.Settings.Load(ctx, &s)
		if err != nil {
			return nil, err
		}
		if s.Account != "" {
			if err := app.store.validateAccount(s.Account); err != nil {
				return nil, connectors.NewInvalidSettingsError(err.Error())
			}
		}
	case "refresh-accounts":
		err := input.Unmarshal(&s)
		if err != nil {
			return nil, connectors.NewInvalidSettingsError("simulated data store settings are malformed")
		}
	case "save":
		err := input.Unmarshal(&s)
		if err != nil {
			return nil, connectors.NewInvalidSettingsError("simulated data store settings are malformed")
		}
		if err := app.store.validateAccount(s.Account); err != nil {
			return nil, connectors.NewInvalidSettingsError(err.Error())
		}
		if usedAccounts[s.Account] {
			return nil, connectors.NewInvalidSettingsError("simulated data store already has a " + roleName + " connection")
		}
		err = app.env.Settings.Store(ctx, struct{}{})
		if err != nil {
			return nil, err
		}
		app.selectedAccount = s.Account
		return nil, nil
	default:
		return nil, connectors.ErrUIEventNotExist
	}

	accounts, err := app.store.Accounts()
	if err != nil {
		return nil, err
	}
	if event == "refresh-accounts" && s.Account != "" {
		isAvailable := false
		for _, account := range accounts {
			if account.ID == s.Account && !usedAccounts[account.ID] {
				isAvailable = true
				break
			}
		}
		if !isAvailable {
			s.Account = ""
		}
	}
	input, err = json.Marshal(s)
	if err != nil {
		return nil, err
	}
	refreshButton := connectors.Button{
		Event: "refresh-accounts", Text: "Refresh simulated data stores", Variant: "neutral", Role: connectors.Both,
	}
	if len(accounts) == 0 {
		return &connectors.UI{
			Fields: []connectors.Component{
				&connectors.Text{
					Label: "Simulated data store",
					Text: "There are no available simulated data stores. Create one in the Simulated world section " +
						"of the Development workspace, then return here.",
					Role: connectors.Both,
				},
			},
			Settings: input,
			Buttons:  []connectors.Button{refreshButton},
		}, nil
	}
	options := make([]connectors.Option, len(accounts))
	for i, account := range accounts {
		options[i] = connectors.Option{Text: account.Name, Value: account.ID, Disabled: usedAccounts[account.ID]}
		if options[i].Disabled {
			options[i].Text += " (" + roleName + " connection already exists)"
		}
	}

	return &connectors.UI{
		Fields: []connectors.Component{
			&connectors.Select{
				Name:        "account",
				Label:       "Simulated data store",
				Placeholder: "Select a simulated data store",
				HelpText:    "Only available simulated data stores without a " + roleName + " connection can be selected.",
				Options:     options,
				Role:        connectors.Both,
			},
		},
		Settings: input,
		Buttons: []connectors.Button{
			refreshButton,
			connectors.SaveButton,
		},
	}, nil
}

// SimulatedSystem returns the account selected by the most recent save.
func (app *Application) SimulatedSystem() string {
	return app.selectedAccount
}

// Upsert creates or partially updates simulated-system records.
func (app *Application) Upsert(ctx context.Context, target connectors.Targets, records connectors.Records, _ types.Type) error {
	if target != connectors.TargetUser {
		return fmt.Errorf("simulated application target is not supported")
	}
	if err := app.store.validateAccount(app.store.account); err != nil {
		return connectors.NewInvalidSettingsError(err.Error())
	}

	input := make([]connectors.Record, 0, recordsPageSize)
	for record := range records.All() {
		input = append(input, record)
	}
	recordsError := make(connectors.RecordsError)
	creates := make([]datastore.SimulatedSystemRecordWrite, 0, len(input))
	createIndexes := make(map[string]int, len(input))
	pendingUpdates := make([]pendingDestinationUpdate, 0, len(input))
	updateIndexes := make(map[string][]int, len(input))
	for i, record := range input {
		if record.Err != nil {
			recordsError[i] = record.Err
			continue
		}
		create := record.IsCreate()
		if err := validateDestinationAttributes(record.Attributes, create); err != nil {
			recordsError[i] = err
			continue
		}
		if create {
			id := generatedDestinationRecordID()
			data, err := newDestinationRecordDocument(id, record.Attributes)
			if err != nil {
				recordsError[i] = err
				continue
			}
			creates = append(creates, datastore.SimulatedSystemRecordWrite{ID: id, Data: data})
			createIndexes[id] = i
			continue
		}
		pendingUpdates = append(pendingUpdates, pendingDestinationUpdate{index: i, id: record.ID, patch: record.Attributes})
		updateIndexes[record.ID] = append(updateIndexes[record.ID], i)
	}

	ids := make([]string, 0, len(pendingUpdates))
	seen := map[string]struct{}{}
	for _, update := range pendingUpdates {
		if _, ok := seen[update.id]; ok {
			continue
		}
		seen[update.id] = struct{}{}
		ids = append(ids, update.id)
	}
	stored, err := app.store.recordsByID(ctx, ids)
	if err != nil {
		return err
	}
	working := make(map[string]json.Value, len(stored))
	for id, data := range stored {
		working[id] = data
	}
	updates := make([]datastore.SimulatedSystemRecordWrite, 0, len(pendingUpdates))
	updatePositions := make(map[string]int, len(pendingUpdates))
	for _, pending := range pendingUpdates {
		data, ok := working[pending.id]
		if !ok {
			recordsError[pending.index] = fmt.Errorf("simulated system record %q does not exist", pending.id)
			continue
		}
		updated, err := updateDestinationRecordDocument(pending.id, data, pending.patch)
		if err != nil {
			recordsError[pending.index] = err
			continue
		}
		working[pending.id] = updated
		write := datastore.SimulatedSystemRecordWrite{ID: pending.id, Data: updated}
		if position, ok := updatePositions[pending.id]; ok {
			updates[position] = write
		} else {
			updatePositions[pending.id] = len(updates)
			updates = append(updates, write)
		}
	}

	if len(creates) == 0 && len(updates) == 0 {
		if len(recordsError) > 0 {
			return recordsError
		}
		return nil
	}
	writeResult, err := app.store.writeRecords(ctx, creates, updates)
	if err != nil {
		return err
	}
	for _, id := range writeResult.CollidingCreates {
		if index, ok := createIndexes[id]; ok {
			recordsError[index] = fmt.Errorf("simulated system record %q already exists", id)
		}
	}
	for _, id := range writeResult.MissingUpdates {
		for _, index := range updateIndexes[id] {
			recordsError[index] = fmt.Errorf("simulated system record %q does not exist", id)
		}
	}
	if len(recordsError) > 0 {
		return recordsError
	}
	return nil
}

type pendingDestinationUpdate struct {
	index int
	id    string
	patch map[string]any
}

func (store *Store) validateAccount(id string) error {
	if id == "" {
		return fmt.Errorf("simulated data store must be selected")
	}
	accounts, err := store.Accounts()
	if err != nil {
		return err
	}
	for _, account := range accounts {
		if account.ID == id {
			return nil
		}
	}
	return fmt.Errorf("simulated data store %s is not available in workspace %s", id, store.workspace)
}
