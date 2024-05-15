import Type, { ObjectType, Property, Role } from '../../../types/external/types';
import { PropertyToEdit } from './useSchemaEdit';

interface EditableProperty {
	indentation: number;
	root: string;
	name: string;
	label: string;
	description: string;
	placeholder: string;
	role: Role;
	type: Type;
	required: boolean;
	nullable: boolean;
	isEditable?: boolean;
}

type EditableSchema = Record<string, EditableProperty>;

// TODO: see comment on flattenSchema in transformedAction.ts.
const transformSchema = (schema: ObjectType): EditableSchema | null => {
	if (schema == null || schema.name !== 'Object') return null;
	const flattenSubProperties = (parentName: string, parentIndentation: number, properties: Property[]) => {
		let flattenedSubProperties = {};
		parentIndentation += 1;
		for (const property of properties) {
			const name = `${parentName}.${property.name}`;
			const flattened: EditableProperty = {
				...property,
				indentation: parentIndentation,
				root: name.substring(0, name.indexOf('.')),
			};
			flattenedSubProperties[name] = flattened;
			if (property.type.name === 'Object') {
				const flattenedProperties = flattenSubProperties(name, parentIndentation, property.type.properties!);
				flattenedSubProperties = { ...flattenedSubProperties, ...flattenedProperties };
			}
		}
		return flattenedSubProperties;
	};

	let transformed = {};
	for (const property of schema.properties!) {
		const indentation = 0;
		const flattened: EditableProperty = {
			...property,
			indentation: indentation,
			root: property.name,
		};
		transformed[property.name] = flattened;
		if (property.type.name === 'Object') {
			const flattenedSubProperties = flattenSubProperties(property.name, indentation, property.type.properties!);
			transformed = { ...transformed, ...flattenedSubProperties };
		}
	}

	return transformed;
};

const normalizeSchema = (schema: EditableSchema): ObjectType => {
	const normalized: ObjectType = { name: 'Object', properties: [] };
	for (const k in schema) {
		if (!schema.hasOwnProperty(k)) {
			continue;
		}
		const property = schema[k];
		const isFirstLevelProperty = property.indentation === 0;
		if (isFirstLevelProperty) {
			const typ = property.type;
			if (typ.name === 'Object') {
				// empty the properties, they will be populated with the
				// edited subproperties.
				typ.properties = [];
			}
			const p: any = {
				name: property.name,
				label: property.label,
				description: property.description,
				type: typ,
				nullable: property.nullable,
			};
			if (!property.isEditable) {
				p.placeholder = property.placeholder;
				p.role = property.role;
				p.required = property.required;
			}
			normalized.properties.push(p);
		} else {
			const parents = k.split('.').slice(0, -1);
			let subProperties = normalized.properties;
			for (let i = 0; i < parents.length; i++) {
				const key = parents.slice(0, i + 1).join('.');
				const name = schema[key].name;
				const typ = subProperties.find((p) => p.name === name).type as ObjectType;
				subProperties = typ.properties;
			}
			const typ = property.type;
			if (typ.name === 'Object') {
				// empty the properties, they will be populated with the
				// edited subproperties.
				typ.properties = [];
			}
			const subP: any = {
				name: property.name,
				label: property.label,
				description: property.description,
				type: typ,
				nullable: property.nullable,
			};
			if (!property.isEditable) {
				subP.placeholder = property.placeholder;
				subP.role = property.role;
				subP.required = property.required;
			}
			subProperties.push(subP);
		}
	}
	return normalized;
};

const newPropertyToEdit = (parentKey: string, indentation: number, root: string): PropertyToEdit => {
	return {
		parentKey: parentKey,
		indentation: indentation,
		root: root,
		name: '',
		nullable: false,
		type: null,
		label: '',
		description: '',
		isEditable: true,
	};
};

export { transformSchema, normalizeSchema, EditableSchema, EditableProperty, newPropertyToEdit };
