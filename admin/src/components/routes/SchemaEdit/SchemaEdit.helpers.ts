import { ObjectType, Property } from '../../../lib/api/types/types';
import { PropertyToEdit } from './useSchemaEdit';

interface EditableProperty extends Property {
	indentation: number;
	root: string;
	isEditable?: boolean;
}

type EditableSchema = Record<string, EditableProperty>;

// TODO: see comment on flattenSchema in transformedPipeline.ts.
const transformSchema = (schema: ObjectType): EditableSchema | null => {
	if (schema == null || schema.kind !== 'object') return null;
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
			if (property.type.kind === 'object') {
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
		if (property.type.kind === 'object') {
			const flattenedSubProperties = flattenSubProperties(property.name, indentation, property.type.properties!);
			transformed = { ...transformed, ...flattenedSubProperties };
		}
	}

	return transformed;
};

const normalizeSchema = (schema: EditableSchema): ObjectType => {
	const normalized: ObjectType = { kind: 'object', properties: [] };
	for (const k in schema) {
		if (!Object.prototype.hasOwnProperty.call(schema, k)) {
			continue;
		}
		const property = schema[k];
		const p = { ...property };
		delete p.indentation;
		delete p.root;
		delete p.isEditable;
		if (p.displayName === '') {
			delete p.displayName;
		}

		if (property.type.kind === 'object') {
			// Empty the properties; they will be populated with the edited
			// subproperties.
			p.type = { ...property.type, properties: [] };
		}
		if (property.isEditable) {
			delete p.prefilled;
			delete p.role;
			delete p.createRequired;
			delete p.updateRequired;
		}

		const isFirstLevelProperty = property.indentation === 0;
		if (isFirstLevelProperty) {
			normalized.properties.push(p);
			continue;
		}

		const parents = k.split('.').slice(0, -1);
		let subProperties = normalized.properties;
		for (let i = 0; i < parents.length; i++) {
			const key = parents.slice(0, i + 1).join('.');
			const name = schema[key].name;
			const typ = subProperties.find((subProperty) => subProperty.name === name).type as ObjectType;
			subProperties = typ.properties;
		}
		subProperties.push(p);
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
		displayName: '',
		description: '',
		isEditable: true,
	};
};

const getParentPropertyKey = (propertyKey: string): string => {
	const separatorIndex = propertyKey.lastIndexOf('.');
	return separatorIndex === -1 ? '' : propertyKey.slice(0, separatorIndex);
};

const getPropertyInsertionAnchor = (
	schema: EditableSchema,
	parentKey: string,
	selectedPropertyKey?: string,
): string | null => {
	if (selectedPropertyKey == null || selectedPropertyKey === parentKey) {
		return null;
	}
	const parentPrefix = parentKey === '' ? '' : `${parentKey}.`;
	if (!selectedPropertyKey.startsWith(parentPrefix)) {
		return null;
	}
	const directChildFragment = selectedPropertyKey.slice(parentPrefix.length).split('.')[0];
	const directChildKey = `${parentPrefix}${directChildFragment}`;
	if (schema[directChildKey] == null) {
		return null;
	}
	let anchorKey = directChildKey;
	for (const propertyKey of Object.keys(schema)) {
		if (propertyKey.startsWith(`${directChildKey}.`)) {
			anchorKey = propertyKey;
		}
	}
	return anchorKey;
};

export {
	transformSchema,
	normalizeSchema,
	EditableSchema,
	EditableProperty,
	getParentPropertyKey,
	getPropertyInsertionAnchor,
	newPropertyToEdit,
};
