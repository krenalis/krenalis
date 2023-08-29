import {
	ActionFilter,
	ActionTarget,
	ActionType,
	ExportMode,
	Mapping,
	MatchingProperties,
	SchedulePeriod,
	Transformation,
} from '../../types/external/action';
import { ActionSchemasResponse } from '../../types/external/api';
import Type, { ObjectType, Property } from '../../types/external/types';
import TransformedConnection from './transformedConnection';
import { DEFAULT_IDENTIFIERS_MAPPING, TransformedIdentifiers } from './transformedIdentifiers';

const SCHEDULE_PERIODS = {
	5: '5m',
	15: '15m',
	30: '30m',
	60: '1h',
	120: '2h',
	180: '3h',
	360: '6h',
	480: '8h',
	720: '12h',
	1440: '24h',
};

type TransformedExportMode = 'Create only' | 'Update only' | 'Create and update';

const EXPORT_MODE_OPTIONS: Record<ExportMode, TransformedExportMode> = {
	CreateOnly: 'Create only',
	UpdateOnly: 'Update only',
	CreateOrUpdate: 'Create and update',
};

interface TransformedProperty {
	value: string;
	required: boolean;
	type: string;
	label: string;
	full: Property;
	indentation?: number;
	root?: string;
	error?: string;
	disabled?: boolean;
}

type TransformedMapping = Record<string, TransformedProperty>;

interface TransformedActionType {
	Name: string;
	Description: string;
	Target: ActionTarget;
	EventType: string;
	MissingSchema: boolean;
	InputSchema: ObjectType;
	OutputSchema: ObjectType;
	Fields: string[];
}

interface TransformedAction {
	ID?: number;
	Connection?: number;
	Target?: ActionTarget;
	Name: string;
	Enabled: boolean;
	EventType?: string | null;
	Running?: boolean;
	ScheduleStart?: number | null;
	SchedulePeriod?: SchedulePeriod | null;
	InSchema: Type | null;
	OutSchema: Type | null;
	Filter: ActionFilter | null;
	Mapping: TransformedMapping | null;
	Transformation: Transformation | null;
	Identifiers?: TransformedIdentifiers | null;
	Query?: string | null;
	Path?: string | null;
	Table?: string | null;
	Sheet?: string | null;
	ExportMode?: ExportMode | null;
	MatchingProperties?: MatchingProperties | null;
}

// TODO: do not set the value and the required values here (this should only
// return the flattened schema). Add a new 'getDefaultMapping' function that
// takes the flatten schema and add values, and disableds. In
// 'transformActionMapping' set the values or set ''. This should only return
// the list of flattened keys mapping to the full property object.
const flattenSchema = (schema: ObjectType): TransformedMapping | null => {
	if (schema == null || schema.name !== 'Object') return null;

	const flattenProperty = (property: Property): TransformedProperty => {
		return {
			value: property.placeholder || '',
			required: property.required,
			type: property.type.name,
			label: property.label,
			full: { ...property },
		};
	};

	const flattenSubProperties = (parentName: string, parentIndentation: number, properties: Property[]) => {
		let flattenedSubProperties = {};
		parentIndentation += 1;
		for (const property of properties) {
			const name = `${parentName}.${property.name}`;
			const flattened = flattenProperty(property);
			flattened.indentation = parentIndentation;
			flattened.root = name.substring(0, name.indexOf('.'));
			flattenedSubProperties[name] = flattened;
			if (property.type.name === 'Object') {
				const flattenedProperties = flattenSubProperties(name, parentIndentation, property.type.properties!);
				flattenedSubProperties = { ...flattenedSubProperties, ...flattenedProperties };
			}
		}
		return flattenedSubProperties;
	};

	let flattenedSchema = {};
	for (const property of schema.properties!) {
		const indentation = 0;
		const flattened = flattenProperty(property);
		flattened.indentation = indentation;
		flattened.root = property.name;
		flattenedSchema[property.name] = flattened;
		if (property.type.name === 'Object') {
			const flattenedSubProperties = flattenSubProperties(property.name, indentation, property.type.properties!);
			flattenedSchema = { ...flattenedSchema, ...flattenedSubProperties };
		}
	}

	return flattenedSchema;
};

const transformActionMapping = (mapping: Mapping, outputSchema: ObjectType): TransformedMapping => {
	const properties = flattenSchema(outputSchema)!;
	for (const propertyName in properties) {
		const isPropertyMapped = mapping[propertyName] != null;
		if (isPropertyMapped) {
			const mappedValue = mapping[propertyName];
			properties[propertyName].value = mappedValue;

			// Disable family properties with different indentation.
			const { root, indentation } = properties[propertyName];
			for (const name in properties) {
				const isFamilyProperty = properties[name].root === root;
				const hasDifferentIndentation = properties[name].indentation !== indentation;
				if (isFamilyProperty && hasDifferentIndentation) {
					properties[name].disabled = true;
				}
			}
		}
	}

	return properties;
};

const computeDefaultAction = (
	actionType: ActionType,
	outputSchema: ObjectType,
	fields: string[]
): TransformedAction => {
	const action: TransformedAction = {
		Name: actionType.Name,
		Enabled: false,
		Filter: null,
		Mapping: flattenSchema(outputSchema),
		InSchema: null,
		OutSchema: null,
		Transformation: null,
	};
	if (fields.includes('Query')) {
		action.Query = '';
	}
	if (fields.includes('Path')) {
		action.Path = '';
	}
	if (fields.includes('Sheet')) {
		action.Sheet = '';
	}
	if (fields.includes('ExportMode')) {
		action.ExportMode = Object.keys(EXPORT_MODE_OPTIONS)[0] as ExportMode;
	}
	if (fields.includes('MatchingProperties')) {
		action.MatchingProperties = { Internal: '', External: '' };
	}
	if (fields.includes('Identifiers')) {
		action.Identifiers = JSON.parse(JSON.stringify(DEFAULT_IDENTIFIERS_MAPPING));
	}
	return action;
};

const computeActionTypeFields = (
	connection: TransformedConnection,
	actionType: ActionType,
	schemas: ActionSchemasResponse
) => {
	const fields: string[] = [];
	if (connection.type === 'App' && connection.role === 'Destination' && actionType.Target === 'Events') {
		fields.push('Filter');
	}
	if (
		(connection.type === 'App' && schemas.In != null && schemas.Out != null) ||
		(connection.type === 'Database' && connection.role === 'Source') ||
		(connection.type === 'File' && connection.role === 'Source') ||
		((connection.type === 'Mobile' || connection.type === 'Server' || connection.type === 'Website') &&
			connection.role === 'Source' &&
			(actionType.Target === 'Users' || actionType.Target === 'Groups'))
	) {
		fields.push('Mapping');
	}
	if (
		connection.type === 'App' &&
		connection.role === 'Destination' &&
		(actionType.Target === 'Users' || actionType.Target === 'Groups')
	) {
		fields.push('MatchingProperties');
		fields.push('ExportMode');
		fields.push('Filter');
	}

	if (connection.type === 'Database' && connection.role === 'Source') {
		fields.push('Query');
	}
	if (connection.type === 'File') {
		if (connection.role === 'Destination') {
			fields.push('Filter');
		}
		fields.push('Path');
		if (connection.connector.hasSheets) {
			fields.push('Sheet');
		}
	}
	if (
		connection.role === 'Source' &&
		(connection.type === 'App' ||
			connection.type === 'Database' ||
			connection.type === 'File' ||
			connection.type === 'Mobile' ||
			connection.type === 'Server' ||
			connection.type === 'Website') &&
		(actionType.Target === 'Users' || actionType.Target === 'Groups')
	) {
		fields.push('Identifiers');
	}
	return fields;
};

export {
	SCHEDULE_PERIODS,
	EXPORT_MODE_OPTIONS,
	flattenSchema,
	transformActionMapping,
	computeDefaultAction,
	computeActionTypeFields,
	TransformedMapping,
	TransformedActionType,
	TransformedAction,
};
