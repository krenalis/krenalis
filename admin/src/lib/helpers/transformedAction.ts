import {
	Action,
	ActionTarget,
	ActionType,
	ExportMode,
	Mapping,
	MatchingProperties,
	SchedulePeriod,
	Transformation,
} from '../../types/external/action';
import { Filter } from '../../types/external/api';
import { ActionSchemasResponse } from '../../types/external/api';
import Type, { FloatType, IntType, ObjectType, Property, UintType } from '../../types/external/types';
import TransformedConnection from './transformedConnection';

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

type TransformedExportMode = 'Create and update' | 'Create only' | 'Update only';

const EXPORT_MODE_OPTIONS: Record<ExportMode, TransformedExportMode> = {
	CreateOrUpdate: 'Create and update',
	CreateOnly: 'Create only',
	UpdateOnly: 'Update only',
};

interface TransformedProperty {
	value: string;
	required: boolean;
	type: string;
	label: string;
	size: number | null;
	full: Property;
	indentation?: number;
	root?: string;
	error?: string;
	disabled?: boolean;
}

type TransformedMapping = Record<string, TransformedProperty>;

type ActionTypeField =
	| 'Filter'
	| 'Mapping'
	| 'MatchingProperties'
	| 'ExportMode'
	| 'Query'
	| 'Path'
	| 'Sheet'
	| 'Table';

interface TransformedActionType {
	Name: string;
	Description: string;
	Target: ActionTarget;
	EventType: string;
	MissingSchema: boolean;
	InputSchema: ObjectType;
	OutputSchema: ObjectType;
	InputMatchingSchema: ObjectType | null;
	OutputMatchingSchema: ObjectType | null;
	Fields: ActionTypeField[];
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
	Filter: Filter | null;
	Mapping: TransformedMapping | null;
	Transformation: Transformation | null;
	Query?: string | null;
	Path?: string | null;
	Table?: string | null;
	Sheet?: string | null;
	IdentityColumn?: string | null;
	TimestampColumn?: string | null;
	TimestampFormat?: string | null;
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
		const flat = {
			value: property.placeholder || '',
			required: property.required,
			type: property.type.name,
			label: property.label,
			size: null,
			full: { ...property },
		};
		if (flat.type === 'Int' || flat.type === 'Uint' || flat.type === 'Float') {
			const prop = property.type as IntType | UintType | FloatType;
			flat.size = prop.bitSize;
		}
		return flat;
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

const transformActionType = (
	actionType: ActionType,
	inputSchema: ObjectType,
	outputSchema: ObjectType,
	inputMatchingSchema: ObjectType,
	outputMatchingSchema: ObjectType,
	fields: ActionTypeField[],
): TransformedActionType => {
	return {
		Name: actionType.Name,
		Description: actionType.Description,
		Target: actionType.Target,
		EventType: actionType.EventType,
		MissingSchema: actionType.MissingSchema,
		InputSchema: inputSchema,
		OutputSchema: outputSchema,
		InputMatchingSchema: inputMatchingSchema,
		OutputMatchingSchema: outputMatchingSchema,
		Fields: fields,
	};
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

const transformAction = (action: Action, outputSchema: ObjectType): TransformedAction => {
	return {
		ID: action.ID,
		Connection: action.Connection,
		Target: action.Target,
		Name: action.Name,
		Enabled: action.Enabled,
		EventType: action.EventType,
		Running: action.Running,
		ScheduleStart: action.ScheduleStart,
		SchedulePeriod: action.SchedulePeriod,
		InSchema: action.InSchema,
		OutSchema: action.OutSchema,
		Filter: action.Filter,
		Mapping: action.Mapping != null ? transformActionMapping(action.Mapping, outputSchema) : null,
		Transformation: action.Transformation,
		Query: action.Query,
		Path: action.Path,
		Table: action.Table,
		Sheet: action.Sheet,
		IdentityColumn: action.IdentityColumn,
		TimestampColumn: action.TimestampColumn,
		TimestampFormat: action.TimestampFormat,
		ExportMode: action.ExportMode,
		MatchingProperties: action.MatchingProperties,
	};
};

const computeDefaultAction = (
	actionType: ActionType,
	connection: TransformedConnection,
	outputSchema: ObjectType,
	fields: string[],
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
		action.Query = connection.connector.sampleQuery;
	}
	if (fields.includes('Path')) {
		action.Path = '';
		action.IdentityColumn = '';
		action.TimestampColumn = '';
		action.TimestampFormat = '';
	}
	if (fields.includes('Sheet')) {
		action.Sheet = '';
	}
	if (fields.includes('Table')) {
		action.Table = '';
	}
	if (fields.includes('ExportMode')) {
		action.ExportMode = Object.keys(EXPORT_MODE_OPTIONS)[0] as ExportMode;
	}
	if (fields.includes('MatchingProperties')) {
		action.MatchingProperties = { Internal: null, External: null };
	}
	return action;
};

const computeActionTypeFields = (
	connection: TransformedConnection,
	actionType: ActionType,
	schemas: ActionSchemasResponse,
) => {
	const fields: ActionTypeField[] = [];
	if (
		(connection.type === 'App' && connection.role === 'Destination' && actionType.Target === 'Events') ||
		(connection.type === 'Database' && connection.role === 'Destination')
	) {
		fields.push('Filter');
	}
	if (
		(connection.type === 'App' && schemas.In != null && schemas.Out != null) ||
		connection.type === 'Database' ||
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
	if (connection.type === 'Database' && connection.role === 'Destination') {
		fields.push('Table');
	}
	return fields;
};

const isIdentifierProperty = (name: string, identifiers: string[] | null): boolean => {
	if (identifiers == null) {
		return false;
	}
	if (identifiers.includes(name)) {
		return true;
	}
	let isIdentifierParent = false;
	for (const identifier of identifiers) {
		if (identifier.includes('.')) {
			const parent = identifier.split('.')[0];
			if (name === parent) {
				isIdentifierParent = true;
				break;
			}
		}
	}
	if (isIdentifierParent) {
		return true;
	}
	return false;
};

export {
	SCHEDULE_PERIODS,
	EXPORT_MODE_OPTIONS,
	flattenSchema,
	computeDefaultAction,
	computeActionTypeFields,
	transformActionType,
	transformAction,
	isIdentifierProperty,
};

export type { TransformedMapping, TransformedActionType, TransformedAction, ActionTypeField };
