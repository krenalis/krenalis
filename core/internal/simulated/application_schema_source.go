// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import "github.com/krenalis/krenalis/tools/types"

var sourceSchema = types.Object([]types.Property{
	{Name: "customer_id", Type: types.String()},
	{Name: "first_name", Type: types.String()},
	{Name: "last_name", Type: types.String()},
	{Name: "email", Type: types.String().AsEmail(), ReadOptional: true},
	{Name: "gender", Type: types.String().WithValues("female", "male")},
	{Name: "birth_date", Type: types.Date()},
	{Name: "current_age", Type: types.Int(8)},
	{Name: "phone", Type: types.String().AsPhone()},
	{Name: "site_url", Type: types.String().AsURL()},
	{Name: "address", Type: types.Object([]types.Property{
		{Name: "street", Type: types.String()},
		{Name: "municipality", Type: types.String()},
		{Name: "state_prov", Type: types.String()},
		{Name: "postal_code", Type: types.String()},
		{Name: "country", Type: types.String().AsCountry(types.ISO3166Alpha2)},
	})},
	{Name: "photo", Type: types.String().AsURL()},
})
