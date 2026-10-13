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
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/validation"
)

const (
	simulatedSystemBatchSize        = 256
	simulatedSystemReferenceDate    = "2026-01-01"
	simulatedSystemNamespaceVersion = 1
	simulatedSystemSourceVersion    = "generation-v1"
)

type simulatedSystemCheckpoint struct {
	Person            int    `json:"person"`
	Representation    int    `json:"representation"`
	CatalogSHA256     string `json:"catalogSHA256"`
	CatalogRelease    string `json:"catalogRelease,omitempty"`
	WorldModelVersion string `json:"worldModelVersion"`
	MarketDataVersion string `json:"marketDataVersion"`
	PersonSpecVersion string `json:"personSpecVersion"`
	SourceVersion     string `json:"sourceVersion"`
	NamespaceVersion  int    `json:"namespaceVersion"`
	ReferenceDate     string `json:"referenceDate"`
	PhotoBaseURL      string `json:"photoBaseURL"`
}

type simulatedSystemPlan struct {
	account       *state.SimulatedSystem
	world         *fakedata.SourceWorld
	socioeconomic *fakedata.SocioeconomicWorld
	source        *fakedata.SourceInstance
	catalog       *fakedata.FaceCatalog
	photoBaseURL  *url.URL
	people        int
	duplicate     int
	offset        uint64
	step          uint64
}

type simulatedSystemWorker struct {
	core   *Core
	wake   chan struct{}
	mu     sync.Mutex
	cancel context.CancelFunc // protected by mu
	wg     sync.WaitGroup
}

func newSimulatedSystemWorker(core *Core) *simulatedSystemWorker {
	return &simulatedSystemWorker{core: core, wake: make(chan struct{}, 1)}
}

func (worker *simulatedSystemWorker) Close() {
	worker.wg.Wait()
}

func (worker *simulatedSystemWorker) Start() {
	worker.wg.Add(1)
	go worker.run()
}

func (worker *simulatedSystemWorker) accountsToProcess() []*state.SimulatedSystem {
	var accounts []*state.SimulatedSystem
	for _, workspace := range worker.core.state.Workspaces() {
		for _, account := range workspace.SimulatedSystems() {
			if account.Status == state.SimulatedSystemDeleting || account.Status == state.SimulatedSystemPreparing {
				accounts = append(accounts, account)
			}
		}
	}
	slices.SortFunc(accounts, func(a, b *state.SimulatedSystem) int {
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

func (worker *simulatedSystemWorker) delete(ctx context.Context, account *state.SimulatedSystem) error {
	workspace := account.Workspace()
	store, ok := worker.core.datastore.Store(workspace.ID)
	if !ok {
		return fmt.Errorf("workspace datastore is unavailable")
	}
	var hasConnections bool
	err := worker.core.db.QueryRow(ctx, "SELECT EXISTS (SELECT FROM connections WHERE workspace = $1 AND simulated_system = $2)",
		workspace.ID, account.ID).Scan(&hasConnections)
	if err != nil {
		return err
	}
	if hasConnections {
		return fmt.Errorf("simulated system is still referenced by a connection")
	}
	var claimed bool
	var generation int64
	err = worker.core.db.QueryRow(ctx, "SELECT claimed_run IS NOT NULL, writer_generation FROM simulated_systems "+
		"WHERE id = $1 AND status = 'Deleting'", account.ID).Scan(&claimed, &generation)
	if err != nil {
		return err
	}
	if claimed {
		return fmt.Errorf("simulated system still has a live destination run")
	}
	err = store.FenceSimulatedSystem(ctx, account.ID, generation+1)
	if err != nil {
		return err
	}
	ctx, err = datastore.WithSimulatedSystemWriter(ctx, account.ID, generation+1)
	if err != nil {
		return err
	}
	err = store.DeleteSimulatedSystemRecords(ctx, account.ID)
	if err != nil {
		return err
	}
	n := state.DeleteSimulatedSystem{Workspace: workspace.ID, ID: account.ID}
	return worker.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx,
			"DELETE FROM simulated_systems WHERE id = $1 AND workspace = $2 AND status = $3 AND claimed_run IS NULL",
			account.ID, workspace.ID, state.SimulatedSystemDeleting)
		if err != nil {
			return nil, err
		}
		if result.RowsAffected() == 0 {
			return nil, nil
		}
		return n, nil
	})
}

func (worker *simulatedSystemWorker) fail(ctx context.Context, account *state.SimulatedSystem, message string) error {
	return worker.saveCheckpoint(ctx, account, account.GenerationCheckpoint, account.GeneratedRecordCount,
		state.SimulatedSystemFailed, message)
}

func (worker *simulatedSystemWorker) leaderChanged() {
	if !worker.core.state.IsLeader() {
		worker.mu.Lock()
		if worker.cancel != nil {
			worker.cancel()
		}
		worker.mu.Unlock()
	}
	worker.wakeUp()
}

func (worker *simulatedSystemWorker) prepare(ctx context.Context, account *state.SimulatedSystem) error {

	if account.GenerationError != "" {
		return worker.fail(ctx, account, account.GenerationError)
	}
	switch account.GenerationPolicyVersion {
	case "generation-v3", "generation-v4":
		return worker.fail(ctx, account, "legacy generation policy "+account.GenerationPolicyVersion+" is no longer supported")
	case simulatedSystemPolicyVersion:
	default:
		return worker.fail(ctx, account, "unsupported generation policy")
	}

	data, err := worker.core.simulatedWorldSource.Load(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return worker.fail(ctx, account, "Italian World data is unavailable")
	}

	// Keep the persisted checkpoint intact for saveCheckpoint's comparison.
	candidate := *account
	if candidate.GenerationCheckpoint == nil {
		if account.GeneratedRecordCount != 0 {
			return worker.fail(ctx, account, "generation checkpoint and counter differ")
		}
		checkpoint, err := initialSimulatedSystemCheckpoint(data, worker.core.simulatedPhotoBaseURL)
		if err != nil {
			return worker.fail(ctx, account, "generation configuration is unavailable or changed")
		}
		candidate.GenerationCheckpoint = checkpoint
	}

	plan, checkpoint, err := newSimulatedSystemPlan(account.Workspace().ID, &candidate, data)
	if err != nil {
		return worker.fail(ctx, account, "generation configuration is unavailable or changed")
	}
	plan.account = account

	if account.GenerationCheckpoint == nil {
		err = worker.saveCheckpoint(ctx, account, candidate.GenerationCheckpoint, 0, state.SimulatedSystemPreparing, "")
		if err != nil {
			return err
		}
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
		records := make([]datastore.SimulatedSystemRecord, 0, simulatedSystemBatchSize)
		next := checkpoint
		for len(records) < simulatedSystemBatchSize && count+len(records) < account.UserCount {
			record, err := plan.record(next.Person, next.Representation)
			if err != nil {
				return worker.fail(ctx, account, "record generation failed")
			}
			records = append(records, record)
			next = plan.advance(next)
		}
		err = store.MergeSimulatedSystemRecords(ctx, account.ID, records)
		if err != nil {
			return err
		}
		count += len(records)
		checkpoint = next
		err = worker.save(ctx, account, checkpoint, count, state.SimulatedSystemPreparing, "")
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
	actual, err := store.CountSimulatedSystemRecords(ctx, account.ID)
	if err != nil {
		return err
	}
	if actual != account.UserCount {
		return worker.fail(ctx, account, "warehouse record count differs from requested count")
	}

	return worker.save(ctx, account, checkpoint, count, state.SimulatedSystemAvailable, "")
}

func (worker *simulatedSystemWorker) readPersistentAccount(ctx context.Context, account *state.SimulatedSystem) (*state.SimulatedSystem, error) {
	copy := *account
	var checkpoint []byte
	err := worker.core.db.QueryRow(ctx, "SELECT status, generated_record_count, generation_checkpoint,"+
		" generation_policy_version, generation_error FROM simulated_systems WHERE id = $1 AND workspace = $2",
		account.ID, account.Workspace().ID).Scan(&copy.Status, &copy.GeneratedRecordCount, &checkpoint,
		&copy.GenerationPolicyVersion, &copy.GenerationError)
	if err != nil {
		return nil, err
	}
	copy.GenerationCheckpoint = json.Value(checkpoint)
	return &copy, nil
}

func (worker *simulatedSystemWorker) run() {
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
		accounts := worker.accountsToProcess()
		for _, account := range accounts {
			if worker.core.close.ctx.Err() != nil || !worker.core.state.IsLeader() {
				break
			}
			ctx, cancel := context.WithCancel(worker.core.close.ctx)
			current, ok := account.Workspace().SimulatedSystem(account.ID)
			if !ok || current.Status != state.SimulatedSystemPreparing && current.Status != state.SimulatedSystemDeleting {
				cancel()
				continue
			}
			persisted, err := worker.readPersistentAccount(ctx, current)
			if err != nil {
				cancel()
				if err != sql.ErrNoRows {
					slog.Warn("cannot read simulated system progress", "account", account.ID, "error", err)
				}
				continue
			}
			if persisted.Status != state.SimulatedSystemPreparing && persisted.Status != state.SimulatedSystemDeleting {
				cancel()
				continue
			}
			worker.mu.Lock()
			worker.cancel = cancel
			worker.mu.Unlock()
			if persisted.Status == state.SimulatedSystemDeleting {
				err = worker.delete(ctx, persisted)
			} else {
				err = worker.prepare(ctx, persisted)
			}
			worker.mu.Lock()
			worker.cancel = nil
			worker.mu.Unlock()
			interrupted := ctx.Err() != nil
			cancel()
			if err != nil && !interrupted {
				slog.Warn("cannot process simulated system; will retry", "account", account.ID)
			}
		}
	}
}

func (worker *simulatedSystemWorker) save(ctx context.Context, account *state.SimulatedSystem, checkpoint simulatedSystemCheckpoint, count int, status state.SimulatedSystemStatus, message string) error {
	value, err := json.Marshal(checkpoint)
	if err != nil {
		return err
	}
	return worker.saveCheckpoint(ctx, account, value, count, status, message)
}

// saveCheckpoint persists progress and seals generation before terminal states.
func (worker *simulatedSystemWorker) saveCheckpoint(ctx context.Context, account *state.SimulatedSystem, value json.Value, count int, status state.SimulatedSystemStatus, message string) error {

	var checkpointValue, previousCheckpoint any
	if value != nil {
		checkpointValue = string(value)
	}
	if account.GenerationCheckpoint != nil {
		previousCheckpoint = string(account.GenerationCheckpoint)
	}

	// Persist failure intent before sealing, so restart can finish a failed
	// status transition without attempting to reopen generation zero.
	if status == state.SimulatedSystemFailed && account.GenerationError == "" {
		intent := state.UpdateSimulatedSystemGeneration{
			Workspace: account.Workspace().ID, ID: account.ID, Status: state.SimulatedSystemPreparing,
			GeneratedRecordCount: account.GeneratedRecordCount, GenerationCheckpoint: account.GenerationCheckpoint,
			GenerationError: message, UpdatedAt: time.Now().UTC(),
		}
		err := worker.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
			result, err := tx.Exec(ctx, "UPDATE simulated_systems SET generation_error = $1, updated_at = $2 "+
				"WHERE id = $3 AND workspace = $4 AND status = 'Preparing' AND generation_error = '' "+
				"AND generated_record_count = $5 AND generation_checkpoint IS NOT DISTINCT FROM $6::jsonb",
				intent.GenerationError, intent.UpdatedAt, intent.ID, intent.Workspace,
				account.GeneratedRecordCount, previousCheckpoint)
			if err != nil {
				return nil, err
			}
			if result.RowsAffected() == 0 {
				return nil, sql.ErrNoRows
			}
			return intent, nil
		})
		if err != nil {
			return err
		}
		account.GenerationError = message
	}

	if status != state.SimulatedSystemPreparing {
		store, ok := worker.core.datastore.Store(account.Workspace().ID)
		if !ok {
			return fmt.Errorf("workspace datastore is unavailable")
		}
		err := store.FenceSimulatedSystem(ctx, account.ID, 1)
		if err != nil {
			return err
		}
	}
	n := state.UpdateSimulatedSystemGeneration{
		Workspace: account.Workspace().ID, ID: account.ID, Status: status,
		GeneratedRecordCount: count, GenerationCheckpoint: value, GenerationError: message, UpdatedAt: time.Now().UTC(),
	}
	err := worker.core.state.Transaction(ctx, func(tx *db.Tx) (any, error) {
		result, err := tx.Exec(ctx, "UPDATE simulated_systems SET status = $1, generated_record_count = $2,"+
			" generation_checkpoint = $3, generation_error = $4, updated_at = $5"+
			" WHERE id = $6 AND workspace = $7 AND status = 'Preparing'"+
			" AND generated_record_count = $8 AND generation_checkpoint IS NOT DISTINCT FROM $9::jsonb"+
			" AND generation_error = $10",
			n.Status, n.GeneratedRecordCount, checkpointValue, n.GenerationError, n.UpdatedAt,
			n.ID, n.Workspace, account.GeneratedRecordCount, previousCheckpoint, account.GenerationError)
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

func (worker *simulatedSystemWorker) wakeUp() {
	select {
	case worker.wake <- struct{}{}:
	default:
	}
}

func (core *Core) wakeSimulatedSystemWorker() {
	if core.simulatedWorker != nil {
		core.simulatedWorker.wakeUp()
	}
}

func initialSimulatedSystemCheckpoint(data *fakedata.ItalianWorldData, photoBaseURL string) (json.Value, error) {

	catalog, err := data.PhotoCatalog()
	if err != nil {
		return nil, err
	}
	release, err := data.BaseRelease()
	if err != nil {
		return nil, err
	}
	photoBaseURL, err = validation.ParseURL(photoBaseURL, validation.NoQuery)
	if err != nil {
		return nil, err
	}
	if photoBaseURL == "" {
		return nil, fmt.Errorf("invalid simulated photo URL")
	}
	checksum := catalog.Checksum()

	return json.Marshal(simulatedSystemCheckpoint{
		CatalogSHA256: hex.EncodeToString(checksum[:]), CatalogRelease: catalog.Release(),
		WorldModelVersion: fakedata.WorldModelVersion,
		MarketDataVersion: release, PersonSpecVersion: fakedata.ItalianJSONPersonSpecVersion,
		SourceVersion: fakedata.SourceSimulationVersion, NamespaceVersion: simulatedSystemNamespaceVersion,
		ReferenceDate: simulatedSystemReferenceDate, PhotoBaseURL: photoBaseURL,
	})
}

func newSimulatedSystemPlan(workspaceID string, account *state.SimulatedSystem, data *fakedata.ItalianWorldData) (*simulatedSystemPlan, simulatedSystemCheckpoint, error) {

	var checkpoint simulatedSystemCheckpoint
	if account.GenerationPolicyVersion != simulatedSystemPolicyVersion {
		return nil, checkpoint, fmt.Errorf("unsupported generation policy")
	}
	err := json.Unmarshal(account.GenerationCheckpoint, &checkpoint)
	if err != nil {
		return nil, checkpoint, err
	}
	catalog, err := data.PhotoCatalog()
	if err != nil {
		return nil, checkpoint, err
	}
	release, err := data.BaseRelease()
	if err != nil {
		return nil, checkpoint, err
	}
	model, err := data.Model()
	if err != nil {
		return nil, checkpoint, err
	}

	photoBaseURL, err := validation.ParseURL(checkpoint.PhotoBaseURL, validation.NoQuery)
	if err != nil {
		return nil, checkpoint, err
	}
	if photoBaseURL == "" || photoBaseURL != checkpoint.PhotoBaseURL {
		return nil, checkpoint, fmt.Errorf("invalid simulated photo URL")
	}
	photoURL, err := url.Parse(photoBaseURL)
	if err != nil {
		return nil, checkpoint, err
	}
	checksum := catalog.Checksum()
	if checkpoint.CatalogRelease != catalog.Release() ||
		checkpoint.CatalogSHA256 != hex.EncodeToString(checksum[:]) ||
		checkpoint.WorldModelVersion != fakedata.WorldModelVersion ||
		checkpoint.MarketDataVersion != release ||
		checkpoint.PersonSpecVersion != fakedata.ItalianJSONPersonSpecVersion ||
		checkpoint.SourceVersion != fakedata.SourceSimulationVersion ||
		checkpoint.NamespaceVersion != simulatedSystemNamespaceVersion ||
		checkpoint.ReferenceDate != simulatedSystemReferenceDate {
		return nil, checkpoint, fmt.Errorf("generation inputs changed")
	}
	if account.UserCount < 1 || account.UserCount > simulatedSystemMaxUserCount ||
		account.DuplicateRecordPercent.Sign() < 0 || account.DuplicateRecordPercent.Greater(decimal.MustInt(50)) {
		return nil, checkpoint, fmt.Errorf("invalid generation configuration")
	}
	percent, err := account.DuplicateRecordPercent.Binary(2)
	if err != nil {
		return nil, checkpoint, err
	}
	cents := new(big.Int).SetBytes(percent).Int64()
	duplicate := int(int64(account.UserCount) * cents / 10_000)
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
	namespace, err := fakedata.NewIdentityNamespace("w"+hex.EncodeToString(worldKey[:10]), simulatedSystemNamespaceVersion)
	if err != nil {
		return nil, checkpoint, err
	}
	world, err := fakedata.NewItalianWorldV1(fakedata.ItalianWorldConfig{
		IdentityNamespace: namespace, WorldSeed: binary.BigEndian.Uint64(worldKey[10:18]),
		ReferenceDate: simulatedSystemReferenceDate,
	}, data)
	if err != nil {
		return nil, checkpoint, err
	}
	socioeconomic, err := fakedata.NewSocioeconomicWorld(world, model)
	if err != nil {
		return nil, checkpoint, err
	}
	sourceWorld, err := fakedata.NewSourceWorld(world, []fakedata.CountryShare{{
		Code: "IT", Version: fakedata.MarketDataVersion, Weight: 1,
	}})
	if err != nil {
		return nil, checkpoint, err
	}
	accountKey := sha256.Sum256([]byte("krenalis/simulated-system/v1/" + account.ID))
	source, err := fakedata.NewSourceInstance(sourceWorld, fakedata.SourceInstanceConfig{
		ID: "a" + hex.EncodeToString(accountKey[:10]), Version: simulatedSystemSourceVersion,
		CoverageNumerator: 1, CoverageDenominator: 1, DuplicateNumerator: 1, DuplicateDenominator: 1,
	})
	if err != nil {
		return nil, checkpoint, err
	}
	modulus := uint64(fakedata.Layer1MaxPersonIndex)
	step := binary.BigEndian.Uint64(accountKey[8:16]) % modulus
	for gcdSimulatedSystem(step, modulus) != 1 {
		step++
	}

	return &simulatedSystemPlan{account: account, world: sourceWorld, socioeconomic: socioeconomic,
		source: source, catalog: catalog, photoBaseURL: photoURL,
		people: people, duplicate: duplicate, offset: binary.BigEndian.Uint64(accountKey[:8]) % modulus,
		step: step}, checkpoint, nil
}

func gcdSimulatedSystem(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func (plan *simulatedSystemPlan) advance(checkpoint simulatedSystemCheckpoint) simulatedSystemCheckpoint {
	if checkpoint.Person < plan.duplicate && checkpoint.Representation == 0 {
		checkpoint.Representation = 1
	} else {
		checkpoint.Person++
		checkpoint.Representation = 0
	}
	return checkpoint
}

func (plan *simulatedSystemPlan) personIndex(person int) fakedata.PersonIndex {
	modulus := uint64(fakedata.Layer1MaxPersonIndex)
	return fakedata.PersonIndex((plan.offset+plan.step*uint64(person))%modulus + 1)
}

func (plan *simulatedSystemPlan) record(person, representation int) (datastore.SimulatedSystemRecord, error) {

	if person < 0 || person >= plan.people || representation < 0 || representation > 1 ||
		(representation == 1 && person >= plan.duplicate) {
		return datastore.SimulatedSystemRecord{}, fmt.Errorf("invalid record coordinate")
	}
	index := plan.personIndex(person)
	records, err := plan.source.Records(index)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}
	record := records[representation]
	canonical, err := plan.world.Person(index)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}
	var photo *string
	if record.PhotoID != nil {
		asset, err := plan.catalog.Asset(*record.PhotoID, fakedata.PhotoSize256)
		if err != nil {
			return datastore.SimulatedSystemRecord{}, err
		}
		value := plan.photoBaseURL.JoinPath(asset.Path).String()
		photo = &value
	}
	document := newSimulatedSystemDocument(canonical, record, photo)

	facts, err := plan.socioeconomic.Facts(index)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}
	err = setSimulatedSystemSocioeconomicFacts(&document, plan.socioeconomic, facts)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}

	data, err := json.Marshal(document)
	if err != nil {
		return datastore.SimulatedSystemRecord{}, err
	}

	return datastore.SimulatedSystemRecord{ID: record.ID, Data: data}, nil
}

func (plan *simulatedSystemPlan) oracle(person, representation int) (fakedata.OracleRecord, error) {
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
