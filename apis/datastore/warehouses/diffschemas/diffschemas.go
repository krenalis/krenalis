//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package diffschemas

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"
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
		if err := validPropertyForDiff(p); err != nil {
			return nil, fmt.Errorf("old schema is not valid for diff: %s", err)
		}
		oldPropsByName[p.Name] = p
	}

	newPropsByName := map[string]types.Property{}
	for _, p := range newSchema.Properties() {
		if err := validPropertyForDiff(p); err != nil {
			return nil, fmt.Errorf("new schema is not valid for diff: %s", err)
		}
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
	// - New properties with the same name as a deleted property. They appear in
	//   "rePaths" (the key is the name of the created property, the value is
	//   nil).
	// - Deleted properties whose name has been reused by a renamed property.
	//   They appear in "rePaths" (the key is the name of the property that
	//   "occupied the name", the value is the name of the deleted property).

	oldNames := oldSchema.PropertiesNames()
	newNames := newSchema.PropertiesNames()

	addedNames := difference(newNames, oldNames)
	droppedNames := difference(oldNames, newNames)
	keptNames := intersection(oldNames, newNames)

	// Keep track of property renamings, it will be useful later to determine
	// whether the ordering has changed or not.
	newNameOf := map[string]string{}

	// Iterate over AddedNames.
	for _, addedName := range addedNames {

		// Renamed properties, whose new name did not already exist in the
		// schema. They appear in "rePaths" (the key is the new name, the
		// value is the old name).
		newPath := appendPath(path, addedName)
		if oldPath, ok := rePaths[newPath].(string); ok {
			oldName := propPathToName(oldPath)
			oldProp := oldPropsByName[oldName]
			newProp := newPropsByName[addedName]
			if !oldProp.Type.EqualTo(newProp.Type) {
				return nil, fmt.Errorf("error on property %q (renamed to %q): type changes are not supported", appendPath(path, oldName), appendPath(path, addedName))
			}
			if oldProp.Nullable != newProp.Nullable {
				return nil, fmt.Errorf("error on property %q (renamed to %q): nullability changes are not supported", appendPath(path, oldName), appendPath(path, addedName))
			}
			if newProp.Type.Kind() == types.ObjectKind {
				return nil, fmt.Errorf("renaming of Object properties is currently not supported (see https://github.com/open2b/chichi/issues/691)")
			}
			newNameOf[oldName] = addedName
			operations = append(operations, warehouses.AlterSchemaOperation{
				Operation: warehouses.OperationRenameProperty,
				Path:      oldPath,
				NewPath:   appendPath(path, addedName),
			})
			continue
		}

		// New properties, whose name did not already exist in the schema.
		// They do not appear in "rePaths".
		if path != "" {
			return nil, fmt.Errorf("cannot add properties to already existent Object properties")
		}
		p := newPropsByName[addedName]
		if containsNullableObject(p) {
			return nil, fmt.Errorf("nullable properties with type Object are not supported")
		}
		operations = append(operations, warehouses.AlterSchemaOperation{
			Operation: warehouses.OperationAddProperty,
			Path:      appendPath(path, addedName),
			Type:      p.Type,
			Nullable:  p.Nullable,
		})
	}

	// Iterate over DroppedNames.
	dropped := map[string]bool{}
	for _, droppedName := range droppedNames {

		// Renamed properties, whose old name has not been reused by any
		// property. They appear in "rePaths" (the key is the new name, the
		// value is the old name).
		// They have been already handled by the code above.
		alreadyHandled := false
		droppedPath := appendPath(path, droppedName)
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
		dropped[droppedName] = true
		droppedProp := oldPropsByName[droppedName]
		if droppedProp.Type.Kind() == types.ObjectKind {
			for _, p := range propertyPaths(droppedProp.Type) {
				operations = append(operations, warehouses.AlterSchemaOperation{
					Operation: warehouses.OperationDropProperty,
					Path:      appendPath(path, appendPath(droppedName, p)),
				})
			}
		} else {
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

		// New properties with the same name as a deleted property. They appear
		// in "rePaths" (the key is the name of the created property, the value
		// is nil).
		if v, ok := rePaths[keptPath]; ok && v == nil {
			dropped[keptName] = true
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

		// Deleted properties whose name has been reused by a renamed property.
		// They appear in "rePaths" (the key is the name of the property that
		// "occupied the name", the value is the name of the deleted property).
		if oldPath, ok := rePaths[keptPath].(string); ok {
			dropped[keptName] = true
			operations = append(operations, warehouses.AlterSchemaOperation{
				Operation: warehouses.OperationDropProperty,
				Path:      keptPath,
			})
			if !oldProp.Type.EqualTo(newProp.Type) {
				return nil, fmt.Errorf("error on property %q: type changes are not supported", appendPath(path, oldProp.Name))
			}
			if oldProp.Nullable != newProp.Nullable {
				return nil, fmt.Errorf("error on property %q: nullability changes are not supported", appendPath(path, oldProp.Name))
			}
			if newProp.Type.Kind() == types.ObjectKind {
				return nil, fmt.Errorf("renaming of Object properties is currently not supported (see https://github.com/open2b/chichi/issues/691)")
			}
			newNameOf[propPathToName(oldPath)] = keptName
			operations = append(operations, warehouses.AlterSchemaOperation{
				Operation: warehouses.OperationRenameProperty,
				Path:      oldPath,
				NewPath:   keptPath,
			})
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
	// - old properties must be in the same position as before (after applying
	//   renamings)
	// - new properties are appended at the end of the properties
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

// containsNullableObject reports whether p, or one of its sub-properties, have
// type Object and are nullable.
func containsNullableObject(p types.Property) bool {
	if p.Type.Kind() != types.ObjectKind {
		return false
	}
	if p.Nullable {
		return true
	}
	for _, subP := range p.Type.Properties() {
		if containsNullableObject(subP) {
			return true
		}
	}
	return false
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

func propertyPaths(obj types.Type) []string {
	paths := []string{}
	for _, p := range obj.Properties() {
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

// validPropertyForDiff validates the fields of the property p, determining if
// their value is allowed in a schema on which the diff must be calculated.
func validPropertyForDiff(p types.Property) error {
	if p.Placeholder != nil {
		return errors.New("property cannot have a placeholder")
	}
	if p.Role != types.BothRole {
		return errors.New("property cannot specify a role")
	}
	if p.Required {
		return errors.New("property cannot be 'required'")
	}
	return nil
}
