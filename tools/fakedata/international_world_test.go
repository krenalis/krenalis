// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// TestItalianJSONAdmission rejects incomplete or unrelated construction inputs.
func TestItalianJSONAdmission(t *testing.T) {

	data := loadInternationalTestAssets(t, internationalAssetFixture(t))
	config := ItalianWorldConfig{
		IdentityNamespace: IdentityNamespace{Name: "krenalis-demo", Generation: 1},
		WorldSeed:         layer1TestSeed, ReferenceDate: layer1TestReferenceDate,
	}
	for _, incomplete := range []*ItalianWorldData{nil, {}} {
		world, err := NewItalianWorldV1(config, incomplete)
		if err != nil {
			if world != nil {
				t.Fatal("expected no incomplete World, got non-nil result")
			}
			continue
		}
		t.Fatal("expected incomplete data rejected, got success")
	}
	for _, invalid := range []ItalianWorldConfig{
		{}, {IdentityNamespace: config.IdentityNamespace, ReferenceDate: "2025-12-31"},
		{IdentityNamespace: IdentityNamespace{Name: "bad", Generation: 0}, ReferenceDate: config.ReferenceDate},
	} {
		_, err := NewItalianWorldV1(invalid, data)
		if err != nil {
			continue
		}
		t.Fatal("expected invalid coordinates rejected, got success")
	}
	world := mustItalianJSONWorld(t, data)
	for _, version := range []string{"", "arbitrary-person-v2"} {
		copyWorld := *world
		copyWorld.personVersion = version
		_, err := NewSocioeconomicWorld(&copyWorld, data.socioeconomic.model)
		if err != nil {
			if !errors.Is(err, ErrInvalidSocioeconomicWorld) {
				t.Fatalf("expected unsupported provenance classification, got %v", err)
			}
			continue
		}
		t.Fatal("expected unrelated World provenance rejected, got success")
	}
	copyModel := *data.socioeconomic.model
	_, err := NewSocioeconomicWorld(world, &copyModel)
	if err != nil {
		if !errors.Is(err, ErrInvalidSocioeconomicWorld) {
			t.Fatalf("expected unbound model rejection, got %v", err)
		}
		return
	}

	t.Fatal("expected the prepared model bound to the new base, got arbitrary model admission")

}

// TestItalianJSONPhotoIdentity verifies the full identity separation through
// actual Person selection and socioeconomic construction.
func TestItalianJSONPhotoIdentity(t *testing.T) {

	assets := internationalAssetFixture(t)
	base := loadInternationalTestAssets(t, assets)
	changedDocument := internationalPhotoFixture()
	for i := range changedDocument.Faces {
		for j := range changedDocument.Faces[i].Renditions {
			changedDocument.Faces[i].Renditions[j].Path = "new/" + changedDocument.Faces[i].Renditions[j].Path
		}
	}
	changedPhotos := mustInternationalPhotos(t, changedDocument)
	changed, err := PrepareItalianWorldData(base.base, base.socioeconomic, changedPhotos)
	if err != nil {
		t.Fatalf("expected path-only replacement prepared, got %v", err)
	}
	first, second := mustItalianJSONWorld(t, base), mustItalianJSONWorld(t, changed)
	if base.photos.fingerprint != changed.photos.fingerprint || base.photos.digest == changed.photos.digest ||
		first.WorldID() != second.WorldID() || first.SnapshotID() == second.SnapshotID() {
		t.Fatal("expected independent selection, catalog, World and Snapshot identities, got conflation")
	}
	layer2 := mustSocioeconomicWorld(t, first, base.socioeconomic.model)
	other := mustSocioeconomicWorld(t, second, base.socioeconomic.model)
	if layer2.ID() != other.ID() || layer2.model.fingerprints != other.model.fingerprints {
		t.Fatal("expected identical L2 identity and fifteen fingerprints, got photographic dependency")
	}
	for index := PersonIndex(1); index <= 64; index++ {
		person, err := first.Person(index)
		if err != nil {
			t.Fatalf("expected original person, got %v", err)
		}
		got, err := second.Person(index)
		if err != nil {
			t.Fatalf("expected person after path change, got %v", err)
		}
		facts, otherFacts := socioeconomicFactsAt(t, layer2, index), socioeconomicFactsAt(t, other, index)
		if person != got || !reflect.DeepEqual(facts, otherFacts) {
			t.Fatal("expected unchanged PhotoID, Person and L2 facts, got path-dependent generation")
		}
	}

}

// TestItalianJSONPhotoSelection covers nearest ages and deterministic ID ties.
func TestItalianJSONPhotoSelection(t *testing.T) {

	photos := mustInternationalPhotos(t, internationalPhotoFixture())
	catalog, err := prepareItalianPhotoCatalog(photos, photos.document.GenderTaxonomy)
	if err != nil {
		t.Fatalf("expected prepared photo selection, got %v", err)
	}
	factory := goldenStreamFactory(t)
	for _, gender := range []Gender{GenderFemale, GenderMale} {
		for index := PersonIndex(1); index <= 32; index++ {
			for _, age := range []int{21, 30, 75} {
				want := string(gender) + "-a"
				if age == 75 {
					want = string(gender) + "-b"
				}
				if age == 30 {
					rng, err := factory.entityStream("person", uint64(index), "photo/select",
						mustStringComponent(t, "face-catalog-version", "v1"),
						mustBytesComponent(t, "face-catalog-sha256", photos.fingerprint[:]))
					if err != nil {
						t.Fatalf("expected photo stream, got %v", err)
					}
					choice, err := rng.Uint64n(2)
					if err != nil {
						t.Fatalf("expected bounded tie choice, got %v", err)
					}
					want = []string{string(gender) + "-a", string(gender) + "-b"}[choice]
				}
				got, err := catalog.selectPhoto(factory, index, gender, age)
				if err != nil {
					t.Fatalf("expected nearest photo, got %v", err)
				}
				if got != want {
					t.Fatalf("expected nearest/tied photo %s, got %s", want, got)
				}
			}
		}
	}

}

// TestItalianJSONStagedAssets qualifies real local documents when present;
// portable tests use the explicitly artificial fixtures above.
func TestItalianJSONStagedAssets(t *testing.T) {

	const stageI = "../../synthetic-world-staging-i-20261013/"
	const stageII = "../../synthetic-world-staging-ii-20261013/"
	paths := map[string]string{
		ItalianReleasePath: stageII + "release.json",
		"synthetic-world/countries/IT/it-2026a-json1/base.json":          stageI + "base.json",
		"synthetic-world/countries/IT/it-2026a-json1/socioeconomic.json": stageI + "socioeconomic.json",
		"synthetic-world/photos/v1-json1/catalog.json":                   stageII + "photos.json",
	}
	assets := map[string][]byte{}
	for path, file := range paths {
		data, err := os.ReadFile(file)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Skip("real local staging assets are not part of the portable test fixtures")
			}
			t.Fatalf("expected readable staging document, got %v", err)
		}
		assets[path] = data
	}
	data := loadInternationalTestAssets(t, assets)
	world := mustItalianJSONWorld(t, data)
	layer2 := mustSocioeconomicWorld(t, world, data.socioeconomic.model)
	if len(data.photos.document.Faces) != 1250 || fmt.Sprintf("%x", layer2.model.digest) != italianSourceModelSHA256 {
		t.Fatal("expected complete real photos and frozen numerical model, got different data")
	}
	source, err := NewSourceWorld(world, []CountryShare{{Code: "IT", Version: "it-2026a", Weight: 1}})
	if err != nil {
		t.Fatalf("expected SourceWorld on real prepared data, got %v", err)
	}
	for _, index := range []PersonIndex{1, 42, 65536, Layer1MaxPersonIndex} {
		person, err := source.Person(index)
		if err != nil {
			t.Fatalf("expected real-data Person, got %v", err)
		}
		facts := socioeconomicFactsAt(t, layer2, index)
		t.Logf("index=%d photo=%s activity=%s", index, person.Person.PhotoID, facts.PrimaryActivity)
	}

}

// TestItalianJSONTransportOrder keeps all meaningful array order while
// changing object properties and the two unordered photographic collections.
func TestItalianJSONTransportOrder(t *testing.T) {

	assets := internationalAssetFixture(t)
	first := loadInternationalTestAssets(t, assets)
	for path, body := range assets {
		assets[path] = reverseInternationalObjects(t, body)
	}
	photo := internationalPhotoFixture()
	assets["photos.json"] = internationalTestJSON(t, photo)
	second := loadInternationalTestAssets(t, assets)
	a, b := mustItalianJSONWorld(t, first), mustItalianJSONWorld(t, second)
	if a.WorldID() != b.WorldID() || a.SnapshotID() != b.SnapshotID() || first.packageDigest != second.packageDigest {
		t.Fatal("expected identical identities after physical reorder, got divergence")
	}
	for _, index := range []PersonIndex{1, 42, Layer1MaxPersonIndex} {
		left, err := a.Person(index)
		if err != nil {
			t.Fatalf("expected original Person, got %v", err)
		}
		right, err := b.Person(index)
		if err != nil {
			t.Fatalf("expected reordered Person, got %v", err)
		}
		if left != right {
			t.Fatal("expected equivalent Person after physical reorder, got different output")
		}
	}

}

// TestItalianJSONWorld exercises the same algorithms with external data and
// verifies the prepared data and identities used by construction and generation.
func TestItalianJSONWorld(t *testing.T) {

	assets := internationalAssetFixture(t)
	legacy := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	data := loadInternationalTestAssets(t, assets)
	world := mustItalianJSONWorld(t, data)
	projected := *data.data
	projected.ageDependency, projected.firstNameDependency = legacy.data.ageDependency, legacy.data.firstNameDependency
	projected.lastNameDependency, projected.placesDependency = legacy.data.lastNameDependency, legacy.data.placesDependency
	projected.postalDependency = legacy.data.postalDependency
	projected.streetNameDependency = legacy.data.streetNameDependency
	if !reflect.DeepEqual(projected, *legacy.data) {
		t.Fatal("expected complete transfer of sampling tables and buckets, got different runtime data")
	}
	if world.data != data.data || world.italian != data || world.faceCatalog != data.catalog ||
		world.PersonSpecVersion() != "person-v2" || world.MarketDataVersion() != "it-2026a-json1" ||
		world.Layer1Protocol() != "layer1-json-v1" {
		t.Fatal("expected prepared data and explicit JSON versions, got different World state")
	}
	if world.streams != legacy.streams || WorldModelVersion != "world-v1" || PersonSpecVersion != "person-v1" ||
		MarketDataVersion != "it-2026a" {
		t.Fatal("expected unchanged root and historical constants, got altered baseline")
	}
	wantID, err := ComposeWorldID(world.namespace,
		mustStringComponent(t, "market", "italy"), mustStringComponent(t, "market-data-version", "it-2026a-json1"),
		mustStringComponent(t, "person-spec-version", "person-v2"), mustStringComponent(t, "world-model-version", "world-v1"),
		mustUint64Component(t, "world-seed", layer1TestSeed))
	if err != nil {
		t.Fatalf("expected new World identity composition, got %v", err)
	}
	if world.WorldID() != wantID || world.WorldID() == legacy.WorldID() {
		t.Fatalf("expected distinct correctly composed WorldID %s, got %s", wantID, world.WorldID())
	}
	wantSnapshot, err := ComposeSnapshotID(wantID,
		mustStringComponent(t, "reference-date", layer1TestReferenceDate),
		mustStringComponent(t, "face-catalog-version", "v1"),
		mustBytesComponent(t, "face-catalog-sha256", data.photos.digest[:]))
	if err != nil {
		t.Fatalf("expected new Snapshot identity composition, got %v", err)
	}
	if world.SnapshotID() != wantSnapshot {
		t.Fatal("expected D_photos in SnapshotID, got different identity")
	}

	// Both municipality branches retain the same places fingerprint.
	fingerprints, err := data.base.Fingerprints()
	if err != nil {
		t.Fatalf("expected six Layer 1 fingerprints, got %v", err)
	}
	for i, dependency := range []Component{
		world.data.ageDependency, world.data.firstNameDependency, world.data.lastNameDependency,
		world.data.placesDependency, world.data.postalDependency, world.data.streetNameDependency,
	} {
		names := []string{"age-data-sha256", "first-name-data-sha256", "last-name-data-sha256",
			"places-data-sha256", "postal-data-sha256", "street-name-data-sha256"}
		if dependency.name != names[i] || !bytes.Equal(dependency.value, fingerprints[i][:]) {
			t.Fatalf("expected fingerprint in component %s, got %s/%x", names[i], dependency.name, dependency.value)
		}
	}
	if !bytes.Equal(data.catalog.dependencies[1].value, data.photos.fingerprint[:]) ||
		string(data.catalog.dependencies[0].value) != "v1" || data.catalog.checksum != data.photos.digest {
		t.Fatal("expected F_photo for selection and D_photos for catalog identity, got conflated inputs")
	}

	indices := []PersonIndex{1, 42, 255, 256, 1024, 999_999, Layer1MaxPersonIndex}
	persons := map[PersonIndex]Person{}
	for _, index := range indices {
		person, err := world.Person(index)
		if err != nil {
			t.Fatalf("expected direct Person %d, got %v", index, err)
		}
		persons[index] = person
	}
	fresh := mustItalianJSONWorld(t, data)
	for i := len(indices) - 1; i >= 0; i-- {
		for _, w := range []*World{fresh, world, fresh} {
			got, err := w.Person(indices[i])
			if err != nil {
				t.Fatalf("expected interleaved Person, got %v", err)
			}
			if got != persons[indices[i]] {
				t.Fatal("expected identical reverse/interleaved access, got divergent Person")
			}
		}
	}
	var group sync.WaitGroup
	for _, index := range indices {
		group.Go(func() {
			got, err := world.Person(index)
			if err != nil {
				t.Errorf("expected concurrent Person, got %v", err)
				return
			}
			if got != persons[index] {
				t.Error("expected deterministic concurrent Person, got different value")
			}
		})
	}
	group.Wait()

	// Compilers, Company draws and all forty Employer indexes are unchanged.
	model, err := data.Model()
	if err != nil {
		t.Fatalf("expected prepared numerical model, got %v", err)
	}
	layer2 := mustSocioeconomicWorld(t, world, model)
	other := mustSocioeconomicWorld(t, fresh, model)
	oldLayer2 := mustSocioeconomicWorld(t, legacy, model)
	if layer2.base != world || layer2.ID() != other.ID() ||
		!reflect.DeepEqual(layer2.companyCounts, oldLayer2.companyCounts) ||
		!reflect.DeepEqual(layer2.companyIndustries, oldLayer2.companyIndustries) ||
		!reflect.DeepEqual(layer2.employers, oldLayer2.employers) {
		t.Fatal("expected authoritative base, stable L2 identity and unchanged Company/Employer algorithms, got divergence")
	}
	for _, index := range indices {
		facts := socioeconomicFactsAt(t, layer2, index)
		if !reflect.DeepEqual(facts, socioeconomicFactsAt(t, other, index)) {
			t.Fatal("expected repeated L2 facts, got divergence")
		}
	}
	for _, index := range []CompanyIndex{1, 42, 65536} {
		if socioeconomicCompanyAt(t, layer2, index) != socioeconomicCompanyAt(t, oldLayer2, index) {
			t.Fatal("expected unchanged Company facts at identical root/model, got divergence")
		}
	}

	source, err := NewSourceWorld(world, []CountryShare{{Code: "IT", Version: "it-2026a", Weight: 1}})
	if err != nil {
		t.Fatalf("expected SourceWorld with historical binding, got %v", err)
	}
	if source.base != world || source.base != layer2.base {
		t.Fatal("expected Source and L2 to share the exact base, got reconstructed World")
	}
	instance := mustSource(t, source, "json-source", 100, 50)
	for _, index := range indices {
		person, err := source.Person(index)
		if err != nil {
			t.Fatalf("expected Source Person, got %v", err)
		}
		if person.Person != persons[index] {
			t.Fatal("expected direct Source delegation, got different ground truth")
		}
		records, err := instance.Records(index)
		if err != nil {
			t.Fatalf("expected Source observations, got %v", err)
		}
		oracle, err := source.Oracle(instance, index)
		if err != nil {
			t.Fatalf("expected Source Oracle, got %v", err)
		}
		if len(records) != len(oracle) || len(records) == 0 || oracle[0].PersonID != persons[index].ID {
			t.Fatal("expected observations mapped to the authoritative person, got invalid Oracle")
		}
	}
	_, err = NewSourceWorld(world, []CountryShare{{Code: "IT", Version: data.base.document.Release, Weight: 1}})
	if err != nil {
		return
	}

	t.Fatal("expected base.release rejected as historical CountryShare binding, got success")

}

func loadInternationalTestAssets(t *testing.T, assets map[string][]byte) *ItalianWorldData {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, exists := assets[strings.TrimPrefix(r.URL.Path, "/")]
		if !exists {
			t.Error("expected only four JSON requests, got undeclared asset or image request")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
	defer server.Close()
	source := mustItalianWorldSource(t, server.URL)
	data, err := source.Load(t.Context())
	if err != nil {
		t.Fatalf("expected complete prepared documents, got %v", err)
	}
	return data
}

func mustItalianJSONWorld(t *testing.T, data *ItalianWorldData) *World {
	t.Helper()
	world, err := NewItalianWorldV1(ItalianWorldConfig{
		IdentityNamespace: IdentityNamespace{Name: "krenalis-demo", Generation: 1},
		WorldSeed:         layer1TestSeed, ReferenceDate: layer1TestReferenceDate,
	}, data)
	if err != nil {
		t.Fatalf("expected Italian JSON World, got %v", err)
	}
	return world
}
