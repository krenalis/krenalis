// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	jsonv2 "encoding/json/v2"
	"fmt"
	"math/big"
	"net/url"
	"reflect"

	"github.com/krenalis/krenalis/core/internal/simulated"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/validation"
)

func (core *Core) usaData(ctx context.Context) (*fakedata.USAWorldData, error) {
	if core.usaWorldSource == nil {
		return nil, fmt.Errorf("USA World source is unavailable")
	}
	return core.usaWorldSource.Load(ctx)
}

func (core *Core) usaDocuments(ctx context.Context, workspaceID, accountID string) (*simulated.USADocuments, error) {

	workspace, ok := core.state.Workspace(workspaceID)
	if !ok || workspace.Environment != state.Development {
		return nil, fmt.Errorf("USA application workspace is unavailable")
	}
	account, ok := workspace.SimulatedSystem(accountID)
	if !ok {
		return nil, fmt.Errorf("USA application system is unavailable")
	}
	data, err := core.usaData(ctx)
	if err != nil {
		return nil, fmt.Errorf("USA application configuration is unavailable")
	}
	plan, _, err := newUSASimulatedSystemPlan(workspaceID, account, data, core.usaPhotoBaseURL)
	if err != nil {
		return nil, fmt.Errorf("USA application configuration is unavailable or changed")
	}

	return plan.usa, nil
}

type simulatedSystemUSABinding struct {
	Country        string `json:"country"`
	Currency       string `json:"currency"`
	AssetsSHA256   string `json:"assetsSHA256"`
	DocumentSHA256 string `json:"documentSHA256"`
	ConfigSHA256   string `json:"configSHA256"`
}

// newUSASimulatedSystemPlan binds the private persistence format to the
// existing USA identities; the historical identity domains remain unchanged.
func newUSASimulatedSystemPlan(workspaceID string, account *state.SimulatedSystem, data *fakedata.USAWorldData, photoBaseURL string) (*simulatedSystemPlan, simulatedSystemCheckpoint, error) {

	var checkpoint simulatedSystemCheckpoint
	if account == nil || account.GenerationPolicyVersion != simulated.USAGenerationPolicy || data == nil ||
		!IsValidID(workspaceID) || !IsValidID(account.ID) {
		return nil, checkpoint, fmt.Errorf("unsupported USA generation configuration")
	}

	// Validate country and sizing before constructing the World.
	weight, hasUS := account.Countries.Get([]string{"US"})
	if !hasUS || account.Countries.NumProperty() != 1 {
		return nil, checkpoint, fmt.Errorf("USA generation requires US country configuration")
	}
	value, err := weight.Int()
	if err != nil {
		return nil, checkpoint, fmt.Errorf("invalid USA country weight")
	}
	if value < 1 || value > 100 || account.UserCount < 0 || account.UserCount > simulatedSystemMaxUserCount ||
		account.DuplicateRecordPercent.Sign() < 0 || account.DuplicateRecordPercent.Greater(decimal.MustInt(50)) {
		return nil, checkpoint, fmt.Errorf("invalid USA generation configuration")
	}
	if account.UserCount == 0 && account.DuplicateRecordPercent.Sign() != 0 {
		return nil, checkpoint, fmt.Errorf("empty USA system cannot have duplicate records")
	}
	percent, err := account.DuplicateRecordPercent.Binary(2)
	if err != nil {
		return nil, checkpoint, err
	}
	duplicateBasisPoints := new(big.Int).SetBytes(percent).Int64()
	duplicate := int(int64(account.UserCount) * duplicateBasisPoints / 10000)
	people := account.UserCount - duplicate

	// Bind verified assets, photographs and the existing World/source identities.
	release, err := data.BaseRelease()
	if err != nil {
		return nil, checkpoint, err
	}
	digest, err := data.Digest()
	if err != nil {
		return nil, checkpoint, err
	}
	catalog, err := data.PhotoCatalog()
	if err != nil {
		return nil, checkpoint, err
	}
	base, err := validation.ParseURL(photoBaseURL, validation.NoQuery)
	if err != nil {
		return nil, checkpoint, err
	}
	photoURL, err := url.Parse(base)
	if err != nil {
		return nil, checkpoint, err
	}
	if photoURL.Host == "" || (photoURL.Scheme != "http" && photoURL.Scheme != "https") {
		return nil, checkpoint, fmt.Errorf("invalid USA photo URL")
	}
	photoURL = photoURL.JoinPath(workspaceID)
	photoURL.Path += "/"
	worldKey := sha256.Sum256([]byte("krenalis/simulated-world/us-demo-v1/" + workspaceID))
	namespace, err := fakedata.NewIdentityNamespace("w"+hex.EncodeToString(worldKey[:10]), simulatedSystemNamespaceVersion)
	if err != nil {
		return nil, checkpoint, err
	}
	accountKey := sha256.Sum256([]byte("krenalis/simulated-system/v1/" + account.ID))
	world, err := fakedata.NewUSAWorld(data, namespace, binary.BigEndian.Uint64(worldKey[10:18]),
		fakedata.SourceInstanceConfig{ID: "a" + hex.EncodeToString(accountKey[:10]), Version: fakedata.USASourceVersion,
			CoverageNumerator: 1, CoverageDenominator: 1, DuplicateNumerator: 1, DuplicateDenominator: 1})
	if err != nil {
		return nil, checkpoint, err
	}
	documentDigest, err := world.DocumentBinding()
	if err != nil {
		return nil, checkpoint, err
	}
	configuration, err := json.Marshal(struct {
		Workspace string
		System    string
		Count     int
		Duplicate int64
		Weight    int
	}{workspaceID, account.ID, account.UserCount, duplicateBasisPoints, value})
	if err != nil {
		return nil, checkpoint, err
	}
	configDigest := sha256.Sum256(configuration)
	checksum := catalog.Checksum()
	expected := simulatedSystemCheckpoint{
		CatalogSHA256: hex.EncodeToString(checksum[:]), CatalogRelease: catalog.Release(),
		WorldModelVersion: fakedata.WorldModelVersion, MarketDataVersion: release,
		PersonSpecVersion: "person-us-v1", SourceVersion: fakedata.USASourceVersion,
		NamespaceVersion: simulatedSystemNamespaceVersion, ReferenceDate: simulatedSystemReferenceDate,
		PhotoBaseURL: photoURL.String(), USA: &simulatedSystemUSABinding{
			Country: "US", Currency: "USD", AssetsSHA256: hex.EncodeToString(digest[:]),
			DocumentSHA256: hex.EncodeToString(documentDigest[:]), ConfigSHA256: hex.EncodeToString(configDigest[:]),
		},
	}
	checkpoint = expected
	if account.GenerationCheckpoint != nil {

		if len(account.GenerationCheckpoint) > 8192 {
			return nil, checkpoint, fmt.Errorf("USA generation checkpoint exceeds byte limit")
		}
		checkpoint = simulatedSystemCheckpoint{}
		err = jsonv2.Unmarshal(account.GenerationCheckpoint, &checkpoint, jsonv2.RejectUnknownMembers(true))
		if err != nil {
			return nil, checkpoint, fmt.Errorf("invalid USA generation checkpoint")
		}
		binding := checkpoint
		binding.Person, binding.Representation = 0, 0
		if !reflect.DeepEqual(binding, expected) {
			return nil, checkpoint, fmt.Errorf("USA generation inputs changed")
		}

	}

	// Validate the cursor against the durable count, including duplicate boundaries.
	if checkpoint.Person < 0 || checkpoint.Person > people ||
		checkpoint.Representation < 0 || checkpoint.Representation > 1 ||
		(checkpoint.Representation == 1 && checkpoint.Person >= duplicate) {
		return nil, checkpoint, fmt.Errorf("invalid USA generation checkpoint")
	}
	count := checkpoint.Person + duplicate
	if checkpoint.Person < duplicate {
		count = 2*checkpoint.Person + checkpoint.Representation
	}
	if count != account.GeneratedRecordCount {
		return nil, checkpoint, fmt.Errorf("USA generation checkpoint and counter differ")
	}
	documents, err := simulated.NewUSADocuments(world, photoURL)
	if err != nil {
		return nil, checkpoint, err
	}
	modulus := uint64(fakedata.Layer1MaxPersonIndex)
	step := binary.BigEndian.Uint64(accountKey[8:16]) % modulus
	for gcdSimulatedSystem(step, modulus) != 1 {
		step++
	}

	return &simulatedSystemPlan{account: account, usa: documents, catalog: catalog, photoBaseURL: photoURL,
		people: people, duplicate: duplicate, offset: binary.BigEndian.Uint64(accountKey[:8]) % modulus,
		step: step}, checkpoint, nil
}
