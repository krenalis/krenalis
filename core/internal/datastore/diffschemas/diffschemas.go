// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package diffschemas

import (
	"fmt"
	"slices"
	"strings"

	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

// Diff returns the differences between oldSchema and newSchema.
//
// rePaths is a mapping containing the renamed property paths, where the key is
// the new property path and its value is the old property path. In case of new
// properties created with the same name of already existent properties, the
// value must be the untyped nil. rePaths cannot contain keys with the same path
// as their value. Any property paths referenced in rePaths that do not refer to
// properties in schemas are ignored.
//
// The Diff function assumes that both the oldSchema and newSchema comply with
// the requirements of data warehouse schemas and that they do not contain
// unsupported types or properties; however, they may contain properties and
// types that are invalid for specific warehouse platforms; related errors will
// then be returned by the platforms.
//
// In case the difference between old schema and new schema cannot be computed,
// an error is returned.
func Diff(oldSchema, newSchema types.Type, rePaths map[string]any, path string) ([]warehouses.AlterOperation, error) {

	if oldSchema.Kind() != types.ObjectKind {
		panic("not an object")
	}
	if newSchema.Kind() != types.ObjectKind {
		panic("not an object")
	}

	oldProperties := oldSchema.Properties()
	newProperties := newSchema.Properties()

	operations := []warehouses.AlterOperation{}

	// Given two schemas, OldSchema and NewSchema, we define OldNames and
	// NewNames as the sets of property names in the two schemas.
	//
	// Then, let's define three sets:
	//
	// - AddedNames ≜ NewNames - OldNames
	// - DroppedNames ≜ OldNames - NewNames
	// - KeptNames ≜ OldNames ∩ NewNames
	//
	// AddedNames contains the added names, thus contains:
	//
	// - New properties, whose name did not already exist in the schema. They do
	//   not appear in "rePaths".
	// - Renamed properties, whose new name did not already exist in the schema.
	//   They appear in "rePaths" (the key is the new name, the value is the old
	//   name).
	//
	// DroppedNames contains the dropped names, thus contains:
	//
	// - Deleted properties, whose name has not been reused by any property.
	//   They do not appear in "rePaths".
	// - Renamed properties, whose old name has not been reused by any property.
	//   They appear in "rePaths" (the key is the new name, the value is the old
	//   name).
	//
	// KeptNames contains the names that remained unchanged, thus contains:
	//
	// - Unchanged properties, that are properties that have not been
	//   added/dropped or renamed. They do not appear in "rePaths".
	// - New properties with the same name of a deleted property. They appear in
	//   "rePaths" (the key is the name of the created property, the value is
	//   nil).
	// - New properties with the same name of a renamed property. They appear in
	//   "rePaths" (the key is the name of the created property, the value is
	//   nil), just like new properties with the same name of a deleted
	//   property, but they also appear in "rePaths" as value where the key is
	//   the new name of the renamed property.
	// - Deleted properties whose name has been reused by a renamed property.
	//   They appear in "rePaths" (the key is the name of the property that
	//   "occupied the name", the value is the name of the deleted property).

	oldNames := oldSchema.Properties().Names()
	newNames := newSchema.Properties().Names()

	addedNames := difference(newNames, oldNames)
	droppedNames := difference(oldNames, newNames)
	keptNames := intersection(oldNames, newNames)

	// Keep track of property renaming, it will be useful later to determine
	// whether the ordering has changed or not.
	newNameOf := map[string]string{}

	// Iterate over AddedNames.
	for _, addedName := range addedNames {

		newPath := appendPath(path, addedName)

		// Since newPath is an added name, the rePaths cannot indicate that it
		// has been created with the same name of an already existent property.
		if v, ok := rePaths[newPath]; ok && v == nil {
			return nil, fmt.Errorf("rePaths cannot contain {..., %q: null, ...}, as there are no properties named %q in the old schema", newPath, newPath)
		}

		// Renamed properties, whose new name did not already exist in the
		// schema. They appear in "rePaths" (the key is the new name, the
		// value is the old name).
		if oldPath, ok := rePaths[newPath].(string); ok {
			oldName := propPathToName(oldPath)
			oldProp, _ := oldProperties.ByName(oldName)
			newProp, _ := newProperties.ByName(addedName)
			newNameOf[oldName] = addedName
			if newProp.Type.Kind() == types.ObjectKind {
				if !types.Equal(oldProp.Type, newProp.Type) {
					return nil, fmt.Errorf("it is not possible to rename an object property (%q, renamed to %q) and simultaneously make changes to its descendant properties", appendPath(path, oldName), appendPath(path, addedName))

				}
				for _, c := range util.PropertiesToColumns(newProp.Type.Properties()) {
					operations = append(operations, warehouses.AlterOperation{
						Operation: warehouses.OperationRenameColumn,
						Column:    pathToColumn(oldPath) + "_" + c.Name,
						NewColumn: pathToColumn(appendPath(path, addedName)) + "_" + c.Name,
					})
				}
			} else {
				if !types.Equal(oldProp.Type, newProp.Type) {
					return nil, fmt.Errorf("error on property %q (renamed to %q): type changes are not supported", appendPath(path, oldName), appendPath(path, addedName))
				}
				operations = append(operations, warehouses.AlterOperation{
					Operation: warehouses.OperationRenameColumn,
					Column:    pathToColumn(oldPath),
					NewColumn: pathToColumn(appendPath(path, addedName)),
				})
			}
			continue
		}

		// New properties, whose name did not already exist in the schema.
		// They do not appear in "rePaths".
		p, _ := newProperties.ByName(addedName)
		if p.Type.Kind() == types.ObjectKind {
			for _, c := range util.PropertiesToColumns(p.Type.Properties()) {
				operations = append(operations, warehouses.AlterOperation{
					Operation: warehouses.OperationAddColumn,
					Column:    pathToColumn(appendPath(path, addedName)) + "_" + c.Name,
					Type:      c.Type,
				})
			}
		} else {
			operations = append(operations, warehouses.AlterOperation{
				Operation: warehouses.OperationAddColumn,
				Column:    pathToColumn(appendPath(path, addedName)),
				Type:      p.Type,
			})
		}
	}

	// Iterate over DroppedNames.
	for _, droppedName := range droppedNames {

		droppedPath := appendPath(path, droppedName)

		// Since the path no longer exists in the new schema, in any case
		// rePaths can contain it.
		if _, ok := rePaths[droppedPath]; ok {
			return nil, fmt.Errorf("rePaths cannot contain %q, as this property no longer exists in the new schema", droppedPath)
		}

		// Renamed properties, whose old name has not been reused by any
		// property. They appear in "rePaths" (the key is the new name, the
		// value is the old name).
		// They have been already handled by the code above.
		alreadyHandled := false
		for _, v := range rePaths {
			if v == droppedPath {
				alreadyHandled = true
				break
			}
		}
		if alreadyHandled {
			continue
		}

		// Deleted properties, whose name has not been reused by any property.
		// They do not appear in "rePaths".
		droppedProp, _ := oldProperties.ByName(droppedName)
		if droppedProp.Type.Kind() == types.ObjectKind {
			for _, p := range propertyPaths(droppedProp.Type) {
				operations = append(operations, warehouses.AlterOperation{
					Operation: warehouses.OperationDropColumn,
					Column:    pathToColumn(appendPath(path, appendPath(droppedName, p))),
				})
			}
		} else {
			operations = append(operations, warehouses.AlterOperation{
				Operation: warehouses.OperationDropColumn,
				Column:    pathToColumn(appendPath(path, droppedName)),
			})
		}
	}

	// Iterate over KeptNames.
	for _, keptName := range keptNames {

		oldProp, _ := oldProperties.ByName(keptName)
		newProp, _ := newProperties.ByName(keptName)
		keptPath := appendPath(path, keptName)

		var renamed bool
		for _, v := range rePaths {
			if v == keptPath {
				renamed = true
				break
			}
		}

		// If a property "bar" was renamed to "foo" (that is, if the rePaths
		// contain {..., "foo": "bar", ...}) and "bar" was retained in the new
		// schema, the possibilities are:
		//
		// 1) "bar" comes from a property that was renamed
		// 2) "bar" is a new property
		//
		// In both cases, "bar" must appear in the rePaths, otherwise there is
		// some kind of inconsistency.
		if _, ok := rePaths[keptPath]; !ok && renamed {
			return nil, fmt.Errorf("property %q has been renamed and still appears in the new schema,"+
				" so it means that it must be declared in rePaths (as a renamed property, or as a new property)", keptPath)
		}

		if v, ok := rePaths[keptPath]; ok && v == nil {
			if renamed {
				// New properties with the same name of a renamed property. They
				// appear in "rePaths" (the key is the name of the created
				// property, the value is nil), just like new properties with
				// the same name of a deleted property, but they also appear in
				// "rePaths" as value where the key is the new name of the
				// renamed property.
				//
				// The Rename operation has already been added in the block that
				// handles AddedNames, so there is nothing to do here. The code
				// outside this 'if' will only handle adding the Add operation.
			} else {
				// New properties with the same name as a deleted property. They
				// appear in "rePaths" (the key is the name of the created
				// property, the value is nil).
				operations = append(operations,
					warehouses.AlterOperation{
						Operation: warehouses.OperationDropColumn,
						Column:    pathToColumn(keptPath),
					})
			}
			if newProp.Type.Kind() == types.ObjectKind {
				for _, c := range util.PropertiesToColumns(newProp.Type.Properties()) {
					operations = append(operations, warehouses.AlterOperation{
						Operation: warehouses.OperationAddColumn,
						Column:    pathToColumn(appendPath(path, keptPath)) + "_" + c.Name,
						Type:      c.Type,
					})
				}
			} else {
				operations = append(operations,
					warehouses.AlterOperation{
						Operation: warehouses.OperationAddColumn,
						Column:    pathToColumn(keptPath),
						Type:      newProp.Type,
					})
			}
			continue
		}

		// Deleted properties whose name has been reused by a renamed property.
		// They appear in "rePaths" (the key is the name of the property that
		// "occupied the name", the value is the name of the deleted property).
		if oldPath, ok := rePaths[keptPath].(string); ok {
			operations = append(operations, warehouses.AlterOperation{
				Operation: warehouses.OperationDropColumn,
				Column:    pathToColumn(keptPath),
			})
			if !types.Equal(oldProp.Type, newProp.Type) {
				return nil, fmt.Errorf("error on property %q: type changes are not supported", appendPath(path, oldProp.Name))
			}
			newNameOf[propPathToName(oldPath)] = keptName
			if newProp.Type.Kind() == types.ObjectKind {
				for _, c := range util.PropertiesToColumns(newProp.Type.Properties()) {
					operations = append(operations, warehouses.AlterOperation{
						Operation: warehouses.OperationRenameColumn,
						Column:    pathToColumn(oldPath) + "_" + c.Name,
						NewColumn: pathToColumn(keptPath) + "_" + c.Name,
					})
				}
			} else {
				operations = append(operations, warehouses.AlterOperation{
					Operation: warehouses.OperationRenameColumn,
					Column:    pathToColumn(oldPath),
					NewColumn: pathToColumn(keptPath),
				})
			}
			continue
		}

		// Unchanged properties, that are properties that have not been
		// added/dropped or renamed. They do not appear in "rePaths".

		if oldProp.Type.Kind() == types.ObjectKind && newProp.Type.Kind() == types.ObjectKind {
			ops, err := Diff(oldProp.Type, newProp.Type, rePaths, appendPath(path, keptName))
			if err != nil {
				return nil, err
			}
			operations = append(operations, ops...)
			continue
		}

		if !types.Equal(oldProp.Type, newProp.Type) {
			return nil, fmt.Errorf("error on property %q: type changes are not supported", appendPath(path, oldProp.Name))
		}

	}

	return operations, nil
}

// appendPath appends name to path. If path is empty, name is returned.
func appendPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

// difference returns set1 - set2.
func difference(set1, set2 []string) []string {
	m := make(map[string]struct{}, len(set2))
	for _, s := range set2 {
		m[s] = struct{}{}
	}
	diff := []string{}
	for _, s := range set1 {
		if _, ok := m[s]; !ok {
			diff = append(diff, s)
		}
	}
	return diff
}

// intersection returns set1 ∩ set2.
func intersection(set1, set2 []string) []string {
	inters := []string{}
	for _, s := range set1 {
		if slices.Contains(set2, s) {
			inters = append(inters, s)
		}
	}
	return inters
}

// pathToColumn returns the column name relative to the given path.
func pathToColumn(path string) string {
	return strings.ReplaceAll(path, ".", "_")
}

// propPathToName returns the name of the given property path.
func propPathToName(path string) string {
	parts := strings.Split(path, ".")
	return parts[len(parts)-1]
}

func propertyPaths(obj types.Type) []string {
	paths := []string{}
	for _, p := range obj.Properties().All() {
		if p.Type.Kind() == types.ObjectKind {
			for _, sub := range propertyPaths(p.Type) {
				paths = append(paths, appendPath(p.Name, sub))
			}
		} else {
			paths = append(paths, p.Name)
		}
	}
	return paths
}
