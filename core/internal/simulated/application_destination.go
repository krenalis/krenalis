// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"bytes"
	"fmt"
	"maps"

	"github.com/krenalis/krenalis/tools/base58"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func validateDestinationAttributes(attributes map[string]any, create bool) error {
	if len(attributes) == 0 {
		return fmt.Errorf("record attributes are empty")
	}
	properties := destinationSchema.Properties()
	for name, value := range attributes {
		property, ok := properties.ByName(name)
		if !ok {
			return fmt.Errorf("destination property %q is not writable", name)
		}
		if name == "address" {
			if err := validateAddressAttributes(value, create); err != nil {
				return err
			}
			continue
		}
		if _, err := types.MarshalValidate(value, property.Type); err != nil {
			return fmt.Errorf("property %q is invalid: %w", name, err)
		}
	}
	if !create {
		return nil
	}
	for _, property := range properties.All() {
		if !property.CreateRequired {
			continue
		}
		if _, ok := attributes[property.Name]; !ok {
			return fmt.Errorf("required create property %q is missing", property.Name)
		}
	}
	return nil
}

func validateAddressAttributes(value any, create bool) error {
	address, ok := value.(map[string]any)
	if !ok || address == nil {
		return fmt.Errorf("property %q is invalid", "address")
	}
	property, _ := destinationSchema.Properties().ByName("address")
	for name, value := range address {
		leaf, ok := property.Type.Properties().ByName(name)
		if !ok {
			return fmt.Errorf("destination property %q is not writable", "address."+name)
		}
		if _, err := types.MarshalValidate(value, leaf.Type); err != nil {
			return fmt.Errorf("property %q is invalid: %w", "address."+name, err)
		}
	}
	if !create {
		return nil
	}
	for _, leaf := range property.Type.Properties().All() {
		if _, ok := address[leaf.Name]; !ok {
			return fmt.Errorf("required create property %q is missing", "address."+leaf.Name)
		}
	}
	return nil
}

func newDestinationRecordDocument(id string, attributes map[string]any) (json.Value, error) {
	if err := validateDestinationAttributes(attributes, true); err != nil {
		return nil, err
	}
	document := maps.Clone(attributes)
	document["customer_id"] = id
	return types.MarshalValidate(document, sourceSchema)
}

func updateDestinationRecordDocument(id string, stored json.Value, patch map[string]any) (json.Value, error) {
	attributes, err := types.Decode[map[string]any](bytes.NewReader(stored), sourceSchema)
	if err != nil {
		return nil, fmt.Errorf("stored record is invalid: %w", err)
	}
	customerID, ok := attributes["customer_id"].(string)
	if !ok || customerID != id {
		return nil, fmt.Errorf("stored customer_id does not match external ID")
	}
	if err := validateDestinationAttributes(patch, false); err != nil {
		return nil, err
	}
	document := maps.Clone(attributes)
	if address, ok := attributes["address"].(map[string]any); ok {
		document["address"] = maps.Clone(address)
	}
	for name, value := range patch {
		if name != "address" {
			document[name] = value
			continue
		}
		current, ok := document["address"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("stored address is invalid")
		}
		maps.Copy(current, value.(map[string]any))
	}
	return types.MarshalValidate(document, sourceSchema)
}

func generatedDestinationRecordID() string {
	return base58.Generate(12)
}
