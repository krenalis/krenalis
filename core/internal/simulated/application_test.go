// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"bytes"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	_ "github.com/krenalis/krenalis/connectors/dummy"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestRegistration verifies the Simulated Application registration contract.
func TestRegistration(t *testing.T) {
	app := connectors.RegisteredApplication(ApplicationCode)
	if app.Code != ApplicationCode || app.Label != "Simulated Application" {
		t.Fatalf("expected Simulated Application registration, got code=%q label=%q", app.Code, app.Label)
	}
	if app.AsSource == nil || app.AsSource.Targets != connectors.TargetUser || !app.AsSource.HasSettings {
		t.Fatalf("expected user source with settings, got %+v", app.AsSource)
	}
	if app.AsDestination == nil || app.AsDestination.Targets != connectors.TargetUser ||
		app.AsDestination.SendingMode != connectors.Server || !app.AsDestination.HasSettings {
		t.Fatalf("expected user server destination with settings, got %+v", app.AsDestination)
	}
	if !reflect.DeepEqual(app.OAuth, connectors.OAuth{}) || app.Terms != (connectors.ApplicationTerms{}) || app.EndpointGroups != nil {
		t.Fatalf("expected no OAuth, terms, or endpoint groups, got OAuth=%+v terms=%+v endpointGroups=%v",
			app.OAuth, app.Terms, app.EndpointGroups)
	}
}

// TestExistingApplicationAcceptsNilExtra verifies that existing connectors
// continue to construct with the default environment.
func TestExistingApplicationAcceptsNilExtra(t *testing.T) {
	_, err := connectors.RegisteredApplication("dummy").New(&connectors.ApplicationEnv{})
	if err != nil {
		t.Fatalf("expected existing connector with nil Extra, got %v", err)
	}
}

// TestNewApplicationRequiresStore verifies that the connector fails closed without its
// internal capability.
func TestNewApplicationRequiresStore(t *testing.T) {
	_, err := NewApplication(&connectors.ApplicationEnv{})
	if err == nil {
		t.Fatal("expected missing Simulated Application store to be rejected, got nil")
	}
}

func TestSourceRecordSchema(t *testing.T) {
	application := &Application{}
	schema, err := application.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected source record schema, got %v", err)
	}
	properties := schema.Properties().Slice()
	wantNames := []string{"customer_id", "first_name", "last_name", "email", "gender", "birth_date",
		"current_age", "phone", "site_url", "address", "photo"}
	gotNames := make([]string, len(properties))
	for i, property := range properties {
		gotNames[i] = property.Name
	}
	if !slices.Equal(gotNames, wantNames) {
		t.Fatalf("expected source schema properties %v, got %v", wantNames, gotNames)
	}
	wantKinds := map[string]types.Kind{
		"customer_id": types.StringKind,
		"first_name":  types.StringKind,
		"last_name":   types.StringKind,
		"email":       types.StringKind,
		"gender":      types.StringKind,
		"birth_date":  types.DateKind,
		"current_age": types.IntKind,
		"phone":       types.StringKind,
		"site_url":    types.StringKind,
		"address":     types.ObjectKind,
		"photo":       types.StringKind,
	}
	for _, property := range properties {
		if property.Type.Kind() != wantKinds[property.Name] {
			t.Fatalf("expected %s to have type %s, got %s", property.Name, wantKinds[property.Name], property.Type.Kind())
		}
		if property.Nullable {
			t.Fatalf("expected %s to reject JSON null, got nullable", property.Name)
		}
		if property.ReadOptional != (property.Name == "email") {
			t.Fatalf("expected %s ReadOptional=%t, got %t", property.Name, property.Name == "email", property.ReadOptional)
		}
	}
	property, ok := schema.Properties().ByName("current_age")
	if !ok || property.Type.BitSize() != 8 {
		t.Fatalf("expected current_age to use int(8), got %+v", property)
	}
	property, ok = schema.Properties().ByName("gender")
	if !ok || !slices.Equal(property.Type.Values(), []string{"female", "male"}) {
		t.Fatalf("expected gender values [female male], got %v", property.Type.Values())
	}
	for name, semantic := range map[string]types.Semantic{
		"email": types.EmailSemantic, "phone": types.PhoneSemantic, "site_url": types.URLSemantic,
		"photo": types.URLSemantic,
	} {
		property, ok := schema.Properties().ByName(name)
		if !ok || property.Type.Semantic() != semantic {
			t.Fatalf("expected %s semantic %s, got %v", name, semantic, property.Type.Semantic())
		}
	}
	property, ok = schema.Properties().ByName("address")
	if !ok {
		t.Fatal("expected address property, got none")
	}
	addressProperties := property.Type.Properties().Slice()
	wantAddressNames := []string{"street", "municipality", "state_prov", "postal_code", "country"}
	gotAddressNames := make([]string, len(addressProperties))
	for i, addressProperty := range addressProperties {
		gotAddressNames[i] = addressProperty.Name
		if addressProperty.ReadOptional || addressProperty.Nullable {
			t.Fatalf("expected address.%s to be required and non-nullable, got optional=%t nullable=%t",
				addressProperty.Name, addressProperty.ReadOptional, addressProperty.Nullable)
		}
	}
	if !slices.Equal(gotAddressNames, wantAddressNames) {
		t.Fatalf("expected address properties %v, got %v", wantAddressNames, gotAddressNames)
	}
	country, ok := property.Type.Properties().ByName("country")
	if !ok || country.Type.Semantic() != types.CountrySemantic || country.Type.CountryFormat() != types.ISO3166Alpha2 {
		t.Fatalf("expected address.country to use ISO alpha-2, got %+v", country)
	}
	destination, err := application.RecordSchema(t.Context(), connectors.TargetUser, connectors.Destination)
	if err != nil {
		t.Fatalf("expected destination record schema, got %v", err)
	}
	destinationProperties := destination.Properties().Slice()
	if len(destinationProperties) != len(wantNames)-1 {
		t.Fatalf("expected %d destination properties, got %d", len(wantNames)-1, len(destinationProperties))
	}
	for _, property := range destinationProperties {
		if property.Name == "customer_id" {
			t.Fatal("expected customer_id to be excluded from destination schema")
		}
		if property.Nullable || property.UpdateRequired {
			t.Fatalf("expected %s to be nullable=false and updateRequired=false, got nullable=%t updateRequired=%t",
				property.Name, property.Nullable, property.UpdateRequired)
		}
	}
}

func TestDecodeSourceRecord(t *testing.T) {
	data := json.Value(`{"customer_id":"sr_v2_a_1","first_name":"Ada","last_name":"Rossi",` +
		`"email":"ada@example.com","gender":"female","birth_date":"1990-01-02","current_age":36,` +
		`"phone":"+393331234567","site_url":"https://profilo.test/ada",` +
		`"address":{"street":"Via Roma 1","municipality":"Roma","state_prov":"RM",` +
		`"postal_code":"00100","country":"IT"},"photo":"https://krenalis.test/photos/catalog/256/face-000001.webp"}`)
	attributes, err := types.Decode[map[string]any](bytes.NewReader(data), sourceSchema)
	if err != nil {
		t.Fatalf("expected final source document to decode, got %v", err)
	}
	if attributes["customer_id"] != "sr_v2_a_1" || attributes["gender"] != "female" ||
		attributes["current_age"] != 36 || attributes["phone"] != "+393331234567" ||
		attributes["site_url"] != "https://profilo.test/ada" ||
		attributes["photo"] != "https://krenalis.test/photos/catalog/256/face-000001.webp" {
		t.Fatalf("expected decoded scalar values, got %+v", attributes)
	}
	birthDate, ok := attributes["birth_date"].(time.Time)
	if !ok || !birthDate.Equal(time.Date(1990, time.January, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected UTC birth date, got %v", attributes["birth_date"])
	}
	address, ok := attributes["address"].(map[string]any)
	if !ok || address["street"] != "Via Roma 1" || address["municipality"] != "Roma" ||
		address["state_prov"] != "RM" || address["postal_code"] != "00100" || address["country"] != "IT" {
		t.Fatalf("expected decoded address, got %+v", attributes["address"])
	}

	missingEmail := json.Value(`{"customer_id":"sr_v2_a_1","first_name":"Ada","last_name":"Rossi",` +
		`"gender":"female","birth_date":"1990-01-02","current_age":36,"phone":"+393331234567",` +
		`"site_url":"https://profilo.test/ada","address":{"street":"Via Roma 1",` +
		`"municipality":"Roma","state_prov":"RM","postal_code":"00100","country":"IT"},` +
		`"photo":"https://krenalis.test/photos/catalog/256/face-000001.webp"}`)
	attributes, err = types.Decode[map[string]any](bytes.NewReader(missingEmail), sourceSchema)
	if err != nil {
		t.Fatalf("expected missing observed email to decode, got %v", err)
	}
	if _, ok := attributes["email"]; ok {
		t.Fatalf("expected omitted email to remain absent, got %v", attributes["email"])
	}
}

func TestRecordsValidation(t *testing.T) {
	application := &Application{store: &Store{}}
	_, _, err := application.Records(t.Context(), connectors.TargetEvent, time.Time{}, "", sourceSchema)
	if err == nil {
		t.Fatal("expected unsupported target to be rejected, got nil")
	}
	_, _, err = application.Records(t.Context(), connectors.TargetUser, time.Time{}, "malformed", sourceSchema)
	if err == nil {
		t.Fatal("expected malformed cursor to be rejected, got nil")
	}
	_, _, err = application.Records(t.Context(), connectors.TargetUser, time.Time{}, "", sourceSchema)
	if err == nil {
		t.Fatal("expected missing account configuration to be rejected, got nil")
	}
}
