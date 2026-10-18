// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"fmt"

	"github.com/krenalis/krenalis/tools/types"
)

// USADemoGenerationPolicy identifies records from the internal USA demo.
const USADemoGenerationPolicy = "generation-us-demo-v1"

var usaDemoSourceSchema = func() types.Type {

	properties := sourceSchema.Properties().Slice()
	for i := range properties {
		if properties[i].Name == "annual_income_eur" {
			properties[i].Name = "annual_income_usd"
		}
	}
	properties = append(properties,
		types.Property{Name: "country", Type: types.String().WithValues("US"), ReadOptional: true},
		types.Property{Name: "currency", Type: types.String().WithValues("USD"), ReadOptional: true},
		types.Property{Name: "photo_id", Type: types.String().WithMaxLength(11), ReadOptional: true},
	)

	return types.Object(properties)
}()

// usaSourceSchema describes the approved M2 Source contact omissions without
// changing the historical USA demo or Italian schemas.
var usaSourceSchema = func() types.Type {
	properties := usaDemoSourceSchema.Properties().Slice()
	for i := range properties {
		switch properties[i].Name {
		case "first_name", "last_name", "phone":
			properties[i].ReadOptional = true
		}
	}
	return types.Object(properties)
}()

func (app *Application) recordSchema() (types.Type, error) {

	if app.store == nil || app.store.account == "" {
		return sourceSchema, nil
	}
	workspace, ok := app.store.state.Workspace(app.store.workspace)
	if !ok {
		return types.Type{}, fmt.Errorf("simulated application workspace is unavailable")
	}
	account, ok := workspace.SimulatedSystem(app.store.account)
	if !ok {
		return types.Type{}, fmt.Errorf("simulated application account is unavailable")
	}
	if account.GenerationPolicyVersion == USAGenerationPolicy {
		return usaSourceSchema, nil
	}
	if account.GenerationPolicyVersion == USADemoGenerationPolicy {
		return usaDemoSourceSchema, nil
	}
	if _, hasUS := account.Countries.Get([]string{"US"}); hasUS {
		return types.Type{}, fmt.Errorf("unsupported USA application generation policy")
	}

	return sourceSchema, nil
}
