// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"strings"

	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

// flatter allows flattening a map[string]any containing profile schema
// attributes into a map[string]any representing profile table columns.
type flatter struct {
	name       string
	column     warehouses.Column
	properties []*flatter
}

// newFlattener returns a new flattener that flattens properties according to
// the given schema and mapping from properties to the respective columns.
func newFlatter(schema types.Type, columnByProperty map[string]warehouses.Column) *flatter {
	flatters := map[string]*flatter{
		"": {properties: []*flatter{}},
	}
	for path, property := range schema.Properties().WalkAll() {
		base := ""
		if i := strings.LastIndex(path, "."); i > 0 {
			base = path[:i]
		}
		parent := flatters[base]
		node := &flatter{name: property.Name}
		if property.Type.Kind() == types.ObjectKind {
			node.properties = []*flatter{}
			flatters[path] = node
		} else {
			node.column = columnByProperty[path]
		}
		parent.properties = append(parent.properties, node)

	}
	return flatters[""]
}

// flat flats proprieties.
func (f *flatter) flat(properties map[string]any) {
	f.flatRec(true, properties, properties)
}

func (f *flatter) flatRec(isRoot bool, root, properties map[string]any) {
	for _, ff := range f.properties {
		v, ok := properties[ff.name]
		if !ok {
			continue
		}
		if ff.properties == nil {
			if !isRoot {
				root[ff.column.Name] = v
			}
		} else {
			ff.flatRec(false, root, v.(map[string]any))
			if isRoot {
				delete(properties, ff.name)
			}
		}
	}
}
