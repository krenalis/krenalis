//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package diffschemas

import (
	"fmt"
	"slices"
	"strings"

	"chichi/apis/datastore/warehouses"
	"chichi/connector/types"
)

// Diff returns the differences between oldSchema and newSchema.
//
// rePaths is a mapping containing the renamed property paths, where the key is
// the new property path and its value is the old property path. In case of new
// properties created with the same name of already existent properties, the
// value must be the untyped nil. rePaths cannot contain keys with the same path
// as their value. Any property path which does not refer to changed properties
// is ignored.
//
// In case the difference between old schema and new schema cannot be computed,
// an error is returned.
func Diff(oldSchema, newSchema types.Type, rePaths map[string]any, path string) ([]warehouses.AlterSchemaOperation, error) {

	if oldSchema.Kind() != types.ObjectKind {
		panic("not an Object")
	}
	if newSchema.Kind() != types.ObjectKind {
		panic("not an Object")
	}

	oldPropsByName := map[string]types.Property{}
	for _, p := range oldSchema.Properties() {
		oldPropsByName[p.Name] = p
	}

	newPropsByName := map[string]types.Property{}
	for _, p := range newSchema.Properties() {
		newPropsByName[p.Name] = p
	}

	operations := []warehouses.AlterSchemaOperation{}

	// Given two schemas, OldSchema and NewSchema, we define OldNames and
	// NewNames as the sets of property names in the two schemas.
	//
	// Then, let's define three sets:
	//
	// - AddedNames ≜ NewNames - OldNames
	// - DroppedNames ≜ OldNames - NewNames
	// - KeptNames ≜ OldNames ∩ NewNames
	//
	// AddedNames contains:
	//
	// - the names of the properties added to the NewSchema
	// - the names of renamed properties
	//
	// DroppedNames contains:
	//
	// - the names of properties dropped
	// - the names of renamed properties (the same ones present in AddedNames)
	//
	// KeptNames contains:
	//
	// - the names of unchanged properties
	// - the names of removed and recreated properties (as new properties) with
	//   the same name

	oldNames := oldSchema.PropertiesNames()
	newNames := newSchema.PropertiesNames()

	addedNames := difference(newNames, oldNames)
	droppedNames := difference(oldNames, newNames)
	keptNames := intersection(oldNames, newNames)

	// Iterate over AddedNames.
	newNameOf := map[string]string{}
	for _, addedName := range addedNames {
		newPath := appendPath(path, addedName)
		if oldPath, ok := rePaths[newPath].(string); ok {
			// Property has been renamed.
			oldName := propPathToName(oldPath)
			oldProp := oldPropsByName[oldName]
			newProp := newPropsByName[addedName]
			if !oldProp.Type.EqualTo(newProp.Type) {
				return nil, fmt.Errorf("error on property %q (renamed to %q): type changes are not supported", appendPath(path, oldName), appendPath(path, addedName))
			}
			if oldProp.Nullable != newProp.Nullable {
				return nil, fmt.Errorf("error on property %q (renamed to %q): nullability changes are not supported", appendPath(path, oldName), appendPath(path, addedName))
			}
			newNameOf[oldName] = addedName
			operations = append(operations, warehouses.AlterSchemaOperation{
				Operation: warehouses.OperationRenameProperty,
				Path:      oldPath,
				Name:      addedName,
			})
		} else {
			// Property has been added.
			p := newPropsByName[addedName]
			operations = append(operations, warehouses.AlterSchemaOperation{
				Operation: warehouses.OperationAddProperty,
				Path:      appendPath(path, addedName),
				Type:      p.Type,
				Nullable:  p.Nullable,
			})
		}
	}

	// Iterate over DroppedNames.
	dropped := map[string]bool{}
	for _, droppedName := range droppedNames {
		if _, ok := newNameOf[droppedName]; ok {
			// Rename operation already added above.
		} else {
			dropped[droppedName] = true
			droppedProp := oldPropsByName[droppedName]
			if droppedProp.Type.Kind() == types.ObjectKind {
				// TODO(Gianluca): see https://github.com/open2b/chichi/issues/581.
				return nil, fmt.Errorf("dropping of Object properties is currently" +
					" not supported (see the issue https://github.com/open2b/chichi/issues/581)")
			}
			operations = append(operations, warehouses.AlterSchemaOperation{
				Operation: warehouses.OperationDropProperty,
				Path:      appendPath(path, droppedName),
			})
		}
	}

	// Iterate over KeptNames.
	for _, keptName := range keptNames {
		oldProp := oldPropsByName[keptName]
		newProp := newPropsByName[keptName]
		keptPath := appendPath(path, keptName)
		if v, ok := rePaths[keptPath]; ok && v == nil {
			operations = append(operations,
				warehouses.AlterSchemaOperation{
					Operation: warehouses.OperationDropProperty,
					Path:      keptPath,
				},
				warehouses.AlterSchemaOperation{
					Operation: warehouses.OperationAddProperty,
					Path:      keptPath,
					Type:      newProp.Type,
					Nullable:  newProp.Nullable,
				})
			continue
		}
		if oldProp.Type.Kind() == types.ObjectKind && newProp.Type.Kind() == types.ObjectKind {
			ops, err := Diff(oldProp.Type, newProp.Type, rePaths, appendPath(path, keptName))
			if err != nil {
				return nil, err
			}
			operations = append(operations, ops...)
			continue
		}
		if !oldProp.Type.EqualTo(newProp.Type) {
			return nil, fmt.Errorf("error on property %q: type changes are not supported", appendPath(path, oldProp.Name))
		}
		if oldProp.Nullable != newProp.Nullable {
			return nil, fmt.Errorf("error on property %q: nullability changes are not supported", appendPath(path, oldProp.Name))
		}
	}

	// Check if the ordering of the properties in the new schema is correct,
	// that is:
	//
	// * old properties must be in the same position as before (after applying
	//   renamings)
	// * new properties are appended at the end of the properties
	//
	oldNamesUpdated := []string{}
	for _, oldName := range oldNames {
		// Dropped properties must not be taken in account.
		if dropped[oldName] {
			continue
		}
		// Renamed properties should be compared with their new name, not the
		// old one.
		if newName, ok := newNameOf[oldName]; ok {
			oldNamesUpdated = append(oldNamesUpdated, newName)
			continue
		}
		// Add the property name as it is.
		oldNamesUpdated = append(oldNamesUpdated, oldName)
	}
	for i, oldName := range oldNamesUpdated {
		newName := newNames[i] // len(newNames) is always >= len(oldNamesUpdated)
		if oldName != newName {
			return nil, fmt.Errorf("properties order has changed (expected property %q, got %q)", oldName, newName)
		}
		// Other properties present in newNames and not present in
		// oldNamesUpdated does not required checking, as they are new
		// properties and they are necessarily to the end of the properties
		// list.
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

// propPathToName returns the name of the given property path.
func propPathToName(path string) string {
	parts := strings.Split(path, ".")
	return parts[len(parts)-1]
}
