//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"strings"
	"sync"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"
)

// flatter allows flattening a map[string]any containing user schema properties
// into a map[string]any representing user table columns.
type flatter struct {
	name       string
	column     meergo.Column
	properties []*flatter
}

// newFlattener returns a new flattener that flattens properties according to
// the given schema and mapping from properties to the respective columns.
func newFlatter(schema types.Type, columnByProperty map[string]meergo.Column) *flatter {
	flatters := map[string]*flatter{
		"": {properties: []*flatter{}},
	}
	for path, property := range types.WalkAll(schema) {
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

// flat flats proprieties and updates the columns argument with the columns in
// properties.
func (f *flatter) flat(properties map[string]any, columns map[string]meergo.Column) {
	f.flatRec(true, properties, properties, columns)
}

func (f *flatter) flatRec(isRoot bool, root, properties map[string]any, columns map[string]meergo.Column) {
	for _, ff := range f.properties {
		v, ok := properties[ff.name]
		if !ok {
			continue
		}
		if ff.properties == nil {
			if !isRoot {
				root[ff.column.Name] = v
				delete(root, ff.name)
			}
			if _, ok := columns[ff.column.Name]; !ok {
				columns[ff.column.Name] = ff.column
			}
		} else {
			ff.flatRec(false, root, v.(map[string]any), columns)
		}
	}
}

// flat flats proprieties and updates the columns argument with the columns in
// properties.
func (f *flatter) flatSync(properties map[string]any, columns *sync.Map) {
	f.flatSyncRec(true, properties, properties, columns)
}

func (f *flatter) flatSyncRec(isRoot bool, root, properties map[string]any, columns *sync.Map) {
	for _, ff := range f.properties {
		v, ok := properties[ff.name]
		if !ok {
			continue
		}
		if ff.properties == nil {
			if !isRoot {
				root[ff.column.Name] = v
				delete(root, ff.name)
			}
			columns.LoadOrStore(ff.column.Name, ff.column)
		} else {
			ff.flatSyncRec(false, root, v.(map[string]any), columns)
		}
	}
}
