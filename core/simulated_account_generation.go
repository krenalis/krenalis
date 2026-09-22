// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"slices"
	"sync"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
)

const (
	simulatedAccountBatchSize        = 256
	simulatedAccountReferenceDate    = "2026-01-01"
	simulatedAccountNamespaceVersion = 1
)

type simulatedAccountCheckpoint struct {
	Person            int    `json:"person"`
	Representation    int    `json:"representation"`
	CatalogSHA256     string `json:"catalogSHA256"`
	WorldModelVersion string `json:"worldModelVersion"`
	MarketDataVersion string `json:"marketDataVersion"`
	PersonSpecVersion string `json:"personSpecVersion"`
	SourceVersion     string `json:"sourceVersion"`
	NamespaceVersion  int    `json:"namespaceVersion"`
	ReferenceDate     string `json:"referenceDate"`
}

type simulatedAccountPlan struct {
	account   *state.SimulatedAccount
	catalog   *fakedata.FaceCatalog
	world     *fakedata.SourceWorld
	source    *fakedata.SourceInstance
	people    int
	duplicate int
	offset    uint64
	step      uint64
}

type simulatedAccountWorker struct {
	core   *Core
	wake   chan struct{}
	mu     sync.Mutex
	cancel context.CancelFunc // protected by mu
	wg     sync.WaitGroup
}

func newSimulatedAccountWorker(core *Core) *simulatedAccountWorker {
	return &simulatedAccountWorker{core: core, wake: make(chan struct{}, 1)}
}

func (worker *simulatedAccountWorker) Start() {
	worker.wg.Add(1)
	go worker.run()
}

func (worker *simulatedAccountWorker) Close() {
	worker.wg.Wait()
}

func (worker *simulatedAccountWorker) leaderChanged() {
	if !worker.core.state.IsLeader() {
		worker.mu.Lock()
		if worker.cancel != nil {
			worker.cancel()
		}
		worker.mu.Unlock()
	}
	worker.wakeUp()
}

func (worker *simulatedAccountWorker) wakeUp() {
	select {
	case worker.wake <- struct{}{}:
	default:
	}
}

func (worker *simulatedAccountWorker) run() {
	defer worker.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	worker.wakeUp()
	for {
		select {
		case <-worker.core.close.ctx.Done():
			return
		case <-worker.wake:
		case <-ticker.C:
		}
		if !worker.core.state.IsLeader() {
			continue
		}
		accounts := worker.preparingAccounts()
		for _, account := range accounts {
			if worker.core.close.ctx.Err() != nil || !worker.core.state.IsLeader() {
				break
			}
			ctx, cancel := context.WithCancel(worker.core.close.ctx)
			current, ok := account.Workspace().SimulatedAccount(account.ID)
			if !ok || current.Status != state.SimulatedAccountPreparing {
				cancel()
				continue
			}
			persisted, err := worker.readPersistentAccount(ctx, current)
			if err != nil {
				cancel()
				if err != sql.ErrNoRows {
					slog.Warn("cannot read simulated account progress", "account", account.ID, "error", err)
				}
				continue
			}
			if persisted.Status != state.SimulatedAccountPreparing {
				cancel()
				continue
			}
			worker.mu.Lock()
			worker.cancel = cancel
			worker.mu.Unlock()
			err = worker.prepare(ctx, persisted)
			worker.mu.Lock()
			worker.cancel = nil
			worker.mu.Unlock()
			interrupted := ctx.Err() != nil
			cancel()
			if err != nil && !interrupted {
				slog.Warn("cannot prepare simulated account; will retry", "account", account.ID)
			}
		}
	}
}

func (worker *simulatedAccountWorker) readPersistentAccount(ctx context.Context, account *state.SimulatedAccount) (*state.SimulatedAccount, error) {
	copy := *account
	var checkpoint []byte
	err := worker.core.db.QueryRow(ctx, "SELECT status, generated_record_count, generation_checkpoint,"+
		" generation_policy_version FROM simulated_accounts WHERE id = $1 AND workspace = $2",
		account.ID, account.Workspace().ID).Scan(&copy.Status, &copy.GeneratedRecordCount, &checkpoint,
		&copy.GenerationPolicyVersion)
	if err != nil {
		return nil, err
	}
	copy.GenerationCheckpoint = json.Value(checkpoint)
	return &copy, nil
}

func (worker *simulatedAccountWorker) preparingAccounts() []*state.SimulatedAccount {
	var accounts []*state.SimulatedAccount
	for _, workspace := range worker.core.state.Workspaces() {
		for _, account := range workspace.SimulatedAccounts() {
			if account.Status == state.SimulatedAccountPreparing {
				accounts = append(accounts, account)
			}
		}
	}
	slices.SortFunc(accounts, func(a, b *state.SimulatedAccount) int {
		if comparison := a.CreatedAt.Compare(b.CreatedAt); comparison != 0 {
			return comparison
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return accounts
}

func (worker *simulatedAccountWorker) prepare(ctx context.Context, account *state.SimulatedAccount) error {
	if worker.core.simulatedCatalog == nil {
		return worker.fail(ctx, account, "photo catalog is unavailable")
	}
	plan, checkpoint, err := newSimulatedAccountPlan(account.Workspace().ID, account, worker.core.simulatedCatalog)
	if err != nil {
		return worker.fail(ctx, account, "generation configuration is unavailable or changed")
	}
	store, ok := worker.core.datastore.Store(account.Workspace().ID)
	if !ok {
		return fmt.Errorf("workspace datastore is unavailable")
	}
	count := account.GeneratedRecordCount
	for count < account.UserCount {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !worker.core.state.IsLeader() {
			return context.Canceled
		}
		records := make([]datastore.SimulatedAccountRecord, 0, simulatedAccountBatchSize)
		next := checkpoint
		for len(records) < simulatedAccountBatchSize && count+len(records) < account.UserCount {
			record, err := plan.record(next.Person, next.Representation)
			if err != nil {
				return worker.fail(ctx, account, "record generation failed")
			}
			records = append(records, record)
			next = plan.advance(next)
		}
		err = store.MergeSimulatedAccountRecords(ctx, account.ID, records)
		if err != nil {
			return err
		}
		count += len(records)
		checkpoint = next
		err = worker.save(ctx, account, checkpoint, count, state.SimulatedAccountPreparing, "")
		if err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !worker.core.state.IsLeader() {
		return context.Canceled
	}
	actual, err := store.CountSimulatedAccountRecords(ctx, account.ID)
	if err != nil {
		return err
	}
	if actual != account.UserCount {
		return worker.fail(ctx, account, "warehouse record count differs from requested count")
	}
	return worker.save(ctx, account, checkpoint, count, state.SimulatedAccountReady, "")
}

func (worker *simulatedAccountWorker) fail(ctx context.Context, account *state.SimulatedAccount, message string) error {
	return worker.saveCheckpoint(ctx, account, account.GenerationCheckpoint, account.GeneratedRecordCount,
		state.SimulatedAccountFailed, message)
}

func (worker *simulatedAccountWorker) save(ctx context.Context, account *state.SimulatedAccount, checkpoint simulatedAccountCheckpoint, count int, status state.SimulatedAccountStatus, message string) error {
	value, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	return worker.saveCheckpoint(ctx, account, value, count, status, message)
}

func (worker *simulatedAccountWorker) saveCheckpoint(ctx context.Context, account *state.SimulatedAccount, value json.Value, count int, status state.SimulatedAccountStatus, message string) error {
	var checkpointValue, previousCheckpoint any
	if value != nil {
		checkpointValue = string(value)
	}
	if account.GenerationCheckpoint != nil {
		previousCheckpoint = string(account.GenerationCheckpoint)
	}
	n := state.UpdateSimulatedAccountGeneration{
		Workspace: account.Workspace().ID, ID: account.ID, Status: status,
		GeneratedRecordCount: count, GenerationCheckpoint: value, GenerationError: message, UpdatedAt: time.Now().UTC(),
	}
	err := worker.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE simulated_accounts SET status = $1, generated_record_count = $2,"+
			" generation_checkpoint = $3, generation_error = $4, updated_at = $5"+
			" WHERE id = $6 AND workspace = $7 AND status = 'Preparing'"+
			" AND generated_record_count = $8 AND generation_checkpoint IS NOT DISTINCT FROM $9::jsonb",
			n.Status, n.GeneratedRecordCount, checkpointValue, n.GenerationError, n.UpdatedAt,
			n.ID, n.Workspace, account.GeneratedRecordCount, previousCheckpoint)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, sql.ErrNoRows
		}
		return n, nil
	})
	if err == nil {
		account.Status = status
		account.GeneratedRecordCount = count
		account.GenerationCheckpoint = value
		account.GenerationError = message
	}
	return err
}

func (core *Core) wakeSimulatedAccountWorker() {
	if core.simulatedWorker != nil {
		core.simulatedWorker.wakeUp()
	}
}

func initialSimulatedAccountCheckpoint(catalog *fakedata.FaceCatalog) (json.Value, error) {
	checksum := catalog.Checksum()
	return json.Marshal(simulatedAccountCheckpoint{
		CatalogSHA256: hex.EncodeToString(checksum[:]), WorldModelVersion: fakedata.WorldModelVersion,
		MarketDataVersion: fakedata.MarketDataVersion, PersonSpecVersion: fakedata.PersonSpecVersion,
		SourceVersion: fakedata.SourceSimulationVersion, NamespaceVersion: simulatedAccountNamespaceVersion,
		ReferenceDate: simulatedAccountReferenceDate,
	})
}

func newSimulatedAccountPlan(workspaceID string, account *state.SimulatedAccount, catalog *fakedata.FaceCatalog) (*simulatedAccountPlan, simulatedAccountCheckpoint, error) {
	var checkpoint simulatedAccountCheckpoint
	err := json.Unmarshal(account.GenerationCheckpoint, &checkpoint)
	if err != nil {
		return nil, checkpoint, err
	}
	checksum := catalog.Checksum()
	if account.GenerationPolicyVersion != simulatedAccountPolicyVersion ||
		checkpoint.CatalogSHA256 != hex.EncodeToString(checksum[:]) ||
		checkpoint.WorldModelVersion != fakedata.WorldModelVersion ||
		checkpoint.MarketDataVersion != fakedata.MarketDataVersion ||
		checkpoint.PersonSpecVersion != fakedata.PersonSpecVersion ||
		checkpoint.SourceVersion != fakedata.SourceSimulationVersion ||
		checkpoint.NamespaceVersion != simulatedAccountNamespaceVersion ||
		checkpoint.ReferenceDate != simulatedAccountReferenceDate {
		return nil, checkpoint, fmt.Errorf("generation inputs changed")
	}
	if account.UserCount < 1 || account.UserCount > simulatedAccountMaxUserCount ||
		account.DuplicateRecordPercent.Sign() < 0 || account.DuplicateRecordPercent.Greater(decimal.MustInt(50)) {
		return nil, checkpoint, fmt.Errorf("invalid generation configuration")
	}
	percent, err := account.DuplicateRecordPercent.Binary(2)
	if err != nil {
		return nil, checkpoint, err
	}
	cents := new(big.Int).SetBytes(percent).Int64()
	duplicate := account.UserCount * int(cents) / 10_000
	people := account.UserCount - duplicate
	if duplicate < 0 || duplicate > people || checkpoint.Person < 0 || checkpoint.Person > people ||
		checkpoint.Representation < 0 || checkpoint.Representation > 1 ||
		(checkpoint.Representation == 1 && checkpoint.Person >= duplicate) {
		return nil, checkpoint, fmt.Errorf("invalid generation checkpoint")
	}
	expected := checkpoint.Person
	if checkpoint.Person < duplicate {
		expected = 2*checkpoint.Person + checkpoint.Representation
	} else {
		expected = 2*duplicate + checkpoint.Person - duplicate
	}
	if expected != account.GeneratedRecordCount {
		return nil, checkpoint, fmt.Errorf("generation checkpoint and counter differ")
	}
	worldKey := sha256.Sum256([]byte("krenalis/simulated-world/v1/" + workspaceID))
	namespace, err := fakedata.NewIdentityNamespace("w"+hex.EncodeToString(worldKey[:10]), simulatedAccountNamespaceVersion)
	if err != nil {
		return nil, checkpoint, err
	}
	world, err := fakedata.NewWorld(fakedata.WorldConfig{
		IdentityNamespace: namespace, WorldSeed: binary.BigEndian.Uint64(worldKey[10:18]),
		ReferenceDate: simulatedAccountReferenceDate, FaceCatalog: catalog,
	})
	if err != nil {
		return nil, checkpoint, err
	}
	sourceWorld, err := fakedata.NewSourceWorld(world, []fakedata.CountryShare{{
		Code: "IT", Version: fakedata.MarketDataVersion, Weight: 1,
	}})
	if err != nil {
		return nil, checkpoint, err
	}
	accountKey := sha256.Sum256([]byte("krenalis/simulated-account/v1/" + account.ID))
	source, err := fakedata.NewSourceInstance(sourceWorld, fakedata.SourceInstanceConfig{
		ID: "a" + hex.EncodeToString(accountKey[:10]), Version: simulatedAccountPolicyVersion,
		CoverageNumerator: 1, CoverageDenominator: 1, DuplicateNumerator: 1, DuplicateDenominator: 1,
	})
	if err != nil {
		return nil, checkpoint, err
	}
	modulus := uint64(fakedata.Layer1MaxPersonIndex)
	step := binary.BigEndian.Uint64(accountKey[8:16]) % modulus
	for gcdSimulatedAccount(step, modulus) != 1 {
		step++
	}
	return &simulatedAccountPlan{account: account, catalog: catalog, world: sourceWorld, source: source,
		people: people, duplicate: duplicate, offset: binary.BigEndian.Uint64(accountKey[:8]) % modulus,
		step: step}, checkpoint, nil
}

func gcdSimulatedAccount(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func (plan *simulatedAccountPlan) advance(checkpoint simulatedAccountCheckpoint) simulatedAccountCheckpoint {
	if checkpoint.Person < plan.duplicate && checkpoint.Representation == 0 {
		checkpoint.Representation = 1
	} else {
		checkpoint.Person++
		checkpoint.Representation = 0
	}
	return checkpoint
}

func (plan *simulatedAccountPlan) personIndex(person int) fakedata.PersonIndex {
	modulus := uint64(fakedata.Layer1MaxPersonIndex)
	return fakedata.PersonIndex((plan.offset+plan.step*uint64(person))%modulus + 1)
}

func (plan *simulatedAccountPlan) record(person, representation int) (datastore.SimulatedAccountRecord, error) {
	if person < 0 || person >= plan.people || representation < 0 || representation > 1 ||
		(representation == 1 && person >= plan.duplicate) {
		return datastore.SimulatedAccountRecord{}, fmt.Errorf("invalid record coordinate")
	}
	records, err := plan.source.Records(plan.personIndex(person))
	if err != nil {
		return datastore.SimulatedAccountRecord{}, err
	}
	record := records[representation]
	document := struct {
		SourceRecordID string  `json:"source_record_id"`
		FirstName      *string `json:"first_name,omitempty"`
		LastName       *string `json:"last_name,omitempty"`
		Email          *string `json:"email,omitempty"`
		Phone          *string `json:"phone,omitempty"`
		PhotoURL       *string `json:"photo_url,omitempty"`
		Country        *string `json:"country,omitempty"`
	}{
		SourceRecordID: record.ID, FirstName: record.FirstName, LastName: record.LastName,
		Email: record.Email, Phone: record.Phone, Country: record.Country,
	}
	if record.PhotoID != nil {
		asset, err := plan.catalog.Asset(*record.PhotoID, fakedata.PhotoSize256)
		if err != nil {
			return datastore.SimulatedAccountRecord{}, err
		}
		document.PhotoURL = &asset.Path
	}
	data, err := json.Marshal(document)
	if err != nil {
		return datastore.SimulatedAccountRecord{}, err
	}
	return datastore.SimulatedAccountRecord{ID: record.ID, Data: data}, nil
}

func (plan *simulatedAccountPlan) oracle(person, representation int) (fakedata.OracleRecord, error) {
	if person < 0 || person >= plan.people || representation < 0 || representation > 1 ||
		(representation == 1 && person >= plan.duplicate) {
		return fakedata.OracleRecord{}, fmt.Errorf("invalid oracle coordinate")
	}
	records, err := plan.world.Oracle(plan.source, plan.personIndex(person))
	if err != nil {
		return fakedata.OracleRecord{}, err
	}
	return records[representation], nil
}
