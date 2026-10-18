// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// TestUSADemoSourceAndDestination checks exact USD and immutable demo facts.
func TestUSADemoSourceAndDestination(t *testing.T) {

	if _, exists := sourceSchema.Properties().ByName("annual_income_usd"); exists {
		t.Fatal("expected unchanged Italian schema, got USD field")
	}
	if _, exists := usaDemoSourceSchema.Properties().ByName("annual_income_eur"); exists {
		t.Fatal("expected USD-only demo schema, got EUR field")
	}
	for _, income := range []uint{0, 1, 1<<32 - 1} {
		t.Run(strconv.FormatUint(uint64(income), 10), func(t *testing.T) {

			document := completeDestinationAttributes()
			document["customer_id"] = "usa-record"
			document["country"], document["currency"] = "US", "USD"
			document["photo_id"] = "face-000001"
			document["annual_income_usd"] = income
			document["education_level"], document["primary_activity"] = "tertiary", "inactive"
			stored, err := types.MarshalValidate(document, usaDemoSourceSchema)
			if err != nil {
				t.Fatalf("expected typed USD Source document, got %v", err)
			}
			updated, err := updateDestinationRecordDocument("usa-record", stored,
				map[string]any{"first_name": "Bea"}, usaDemoSourceSchema)
			if err != nil {
				t.Fatalf("expected USA Destination partial update, got %v", err)
			}
			decoded, err := types.Decode[map[string]any](bytes.NewReader(updated), usaDemoSourceSchema)
			if err != nil {
				t.Fatalf("expected readable updated USA record, got %v", err)
			}
			if decoded["annual_income_usd"] != income || decoded["currency"] != "USD" ||
				decoded["photo_id"] != "face-000001" || decoded["first_name"] != "Bea" {
				t.Fatalf("expected unchanged USD facts and PhotoID with updated name, got %v", decoded)
			}
			for _, field := range []string{"annual_income_usd", "country", "currency", "photo_id"} {
				t.Run(field, func(t *testing.T) {
					_, err := updateDestinationRecordDocument("usa-record", stored,
						map[string]any{field: document[field]}, usaDemoSourceSchema)
					if err != nil {
						return
					}
					t.Fatalf("expected canonical %s update rejection, got success", field)
				})
			}

		})
	}

}
