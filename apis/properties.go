//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"encoding/json"
	"sort"
)

type Properties struct {
	*APIs
}

// UserSchemaProperties returns the name of the properties of the user schema
// for the given account.
//
// TODO(Gianluca): return properties with the same ordering of the schema,
// instead of sorting them alphabetically.
func (properties *Properties) UserSchemaProperties(account int) ([]string, error) {
	schema, err := properties.Schemas.Get(account, "user")
	if err != nil {
		return nil, err
	}
	var v struct {
		Properties map[string]any
	}
	err = json.Unmarshal([]byte(schema), &v)
	if err != nil {
		return nil, err
	}
	props := make([]string, 0, len(v.Properties))
	for name := range v.Properties {
		props = append(props, name)
	}
	sort.Strings(props)
	return props, nil
}
