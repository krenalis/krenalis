//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package datastore

import (
	"encoding/json"
	"fmt"
	"log"
)

// Index is an index on a Redis instance.
//
// The index uses these kinds of keys:
//
//   - props:<property>:<property value>  → [<user GID>, ...]   For properties with non-zero values
//   - props:<property>:-                 → [<user GID>, ...]   For properties with zero values
//   - user_prop_keys:<user GID>          → [<key 1>, ...]      For holding the keys of the properties of an user

// intersection returns the intersection between a and b.
// The elements in the returned slice are ordered as they appear in a.
func intersection[T comparable](a, b []T) []T {
	out := []T{}
	for _, v := range a {
		for _, w := range b {
			if w == v {
				out = append(out, v)
				break
			}
		}
	}
	return out
}

func jsonDeserialize(raw string) any {
	// TODO: improve or remove this function.
	var v any
	err := json.Unmarshal([]byte(raw), &v)
	if err != nil {
		log.Panic(err)
	}
	return v
}

func jsonSerialize(v any) string {
	// TODO: improve or remove this function.
	data, err := json.Marshal(v)
	if err != nil {
		log.Panic(err)
	}
	return string(data)
}

func zero(v any) bool {
	return v == "" || v == 0 || v == nil
}

// Functions for retrieving index keys.

func propsKey(property string, value any) string {
	v := jsonSerialize(value)
	return fmt.Sprintf("props:%s:%s", property, v)
}

func propsZeroKey(property string) string {
	return fmt.Sprintf("props:%s:-", property)
}

func userPropsKeysKey(gid int) string {
	return fmt.Sprintf("user_prop_keys:%d", gid)
}
