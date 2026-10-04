// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import "github.com/krenalis/krenalis/tools/types"

var destinationSchema = types.Object([]types.Property{
	{Name: "first_name", Type: types.String(), CreateRequired: true},
	{Name: "last_name", Type: types.String(), CreateRequired: true},
	{Name: "email", Type: types.String().AsEmail()},
	{Name: "gender", Type: types.String().WithValues("female", "male"), CreateRequired: true},
	{Name: "birth_date", Type: types.Date(), CreateRequired: true},
	{Name: "current_age", Type: types.Int(8), CreateRequired: true},
	{Name: "phone", Type: types.String().AsPhone(), CreateRequired: true},
	{Name: "site_url", Type: types.String().AsURL(), CreateRequired: true},
	{Name: "address", Type: types.Object([]types.Property{
		{Name: "street", Type: types.String(), CreateRequired: true},
		{Name: "municipality", Type: types.String(), CreateRequired: true},
		{Name: "state_prov", Type: types.String(), CreateRequired: true},
		{Name: "postal_code", Type: types.String(), CreateRequired: true},
		{Name: "country", Type: types.String().AsCountry(types.ISO3166Alpha2), CreateRequired: true},
	}), CreateRequired: true},
	{Name: "photo", Type: types.String().AsURL(), CreateRequired: true},
})
