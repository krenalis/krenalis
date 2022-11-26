//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package types

import (
	"errors"
	"sort"
)

var errInvalidSchemaSyntax = errors.New("invalid schema syntax")

// Schema represents a schema.
type Schema struct {
	properties []Property
}

// Property represents a schema property.
type Property struct {
	Name        string
	Aliases     []string
	Label       string
	Description string
	Role        Role
	Type        Type
}

// MustSchemaOf is like SchemaOf but panics instead of returning an error.
func MustSchemaOf(properties []Property) Schema {
	schema, err := SchemaOf(properties)
	if err != nil {
		panic(err.Error())
	}
	return schema
}

// SchemaOf returns a new schema with the given properties.
// It returns an error if properties is empty, or if a property name or alias
// is empty or repeated, or if a property string field is not UTF-8 encoded, or
// if a property type and role are not valid.
func SchemaOf(properties []Property) (Schema, error) {
	if len(properties) == 0 {
		return Schema{}, errors.New("no property in schema")
	}
	exists := make(map[string]struct{}, len(properties))
	ps := make([]Property, len(properties))
	for i, property := range properties {
		if property.Name == "" {
			return Schema{}, errors.New("empty property name")
		}
		normalizedName := normalizedUTF8(property.Name)
		if _, ok := exists[normalizedName]; ok {
			return Schema{}, errors.New("property name is repeated")
		}
		exists[normalizedName] = struct{}{}
		var aliases []string
		if len(property.Aliases) > 0 {
			aliases = make([]string, len(property.Aliases))
			for i, alias := range property.Aliases {
				if alias == "" {
					return Schema{}, errors.New("empty property alias")
				}
				aliases[i] = normalizedUTF8(alias)
				if _, ok := exists[aliases[i]]; ok {
					return Schema{}, errors.New("property alias already named")
				}
				exists[aliases[i]] = struct{}{}
			}
			sort.Strings(aliases)
		}
		if property.Role < BothRole || property.Role > DestinationRole {
			return Schema{}, errors.New("invalid property role")
		}
		if !property.Type.Valid() {
			return Schema{}, errors.New("invalid property type")
		}
		ps[i] = Property{
			Name:        normalizedName,
			Aliases:     aliases,
			Label:       normalizedUTF8(property.Label),
			Description: normalizedUTF8(property.Description),
			Role:        property.Role,
			Type:        property.Type,
		}
	}
	return Schema{ps}, nil
}

// AsRole returns a schema with the properties of schema but that are
// compatible with role. It returns schema if all properties are compatible and
// an invalid schema if there are no compatible properties.
//
// Panics if schema or role are not valid or role is Both.
func (schema Schema) AsRole(role Role) Schema {
	if !schema.Valid() {
		panic("schema is not valid")
	}
	if role < BothRole || role > DestinationRole {
		panic("role is not valid")
	}
	if role == BothRole {
		return schema
	}
	start := 0
	var properties []Property
	for i, p := range schema.properties {
		if p.Role == BothRole || p.Role == role {
			continue
		}
		if start < i {
			properties = append(properties, schema.properties[start:i]...)
		}
		start = i + 1
	}
	if properties == nil {
		return schema
	}
	if start < len(properties) {
		properties = append(properties, properties[start:]...)
	}
	return Schema{properties}
}

// Properties returns the properties of schema.
// Panics if schema is not a valid schema.
func (schema Schema) Properties() []Property {
	if !schema.Valid() {
		panic("schema is not valid")
	}
	properties := make([]Property, len(schema.properties))
	copy(properties, schema.properties)
	return properties
}

// PropertiesNames returns the names of the properties.
// Panics if schema is not a valid schema.
func (schema Schema) PropertiesNames() []string {
	if !schema.Valid() {
		panic("schema is not valid")
	}
	names := make([]string, len(schema.properties))
	for i, p := range schema.properties {
		names[i] = p.Name
	}
	return names
}

// Valid reports whether schema is valid.
func (schema Schema) Valid() bool {
	return schema.properties != nil
}
