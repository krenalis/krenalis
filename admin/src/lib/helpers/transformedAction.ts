import {
	Action,
	ActionTarget,
	ActionToSet,
	ActionType,
	ExportMode,
	ExpressionToBeExtracted,
	Mapping,
	MatchingProperties,
	SchedulePeriod,
	TransformationFunction,
} from '../../types/external/action';
import { Filter } from '../../types/external/api';
import { AnonymousIdentifiers } from '../../types/external/identifiers';
import { FloatType, IntType, ObjectType, Property, UintType } from '../../types/external/types';
import API from '../api/api';
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

interface TransformedTransformation {
	Mapping: TransformedMapping | null;
	Function: TransformationFunction | null;
}

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
	InSchema: ObjectType | null;
	OutSchema: ObjectType | null;
	Filter: Filter | null;
	Transformation: TransformedTransformation | null;
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
	fields: ActionTypeField[],
	inputSchema: ObjectType,
	outputSchema: ObjectType,
	inputMatchingSchema: ObjectType,
	outputMatchingSchema: ObjectType,
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
		Transformation: {
			Mapping:
				action.Transformation.Mapping != null
					? transformActionMapping(action.Transformation.Mapping, outputSchema)
					: null,
			Function: action.Transformation.Function,
		},
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

const transformInActionToSet = async (
	action: TransformedAction,
	actionType: TransformedActionType,
	api: API,
	anonymousIdentifiers: AnonymousIdentifiers,
	connection: TransformedConnection,
): Promise<ActionToSet> => {
	let mapping: Mapping;
	let inSchema: ObjectType;
	let outSchema: ObjectType;
	let func: TransformationFunction;
	let query: string;

	const flattenedInputSchema = flattenSchema(actionType.InputSchema);
	const flattenedOutputSchema = flattenSchema(actionType.OutputSchema);

	if (action.Transformation.Mapping != null) {
		const inputSchema: ObjectType = { name: 'Object', properties: [] };
		const outputSchema: ObjectType = { name: 'Object', properties: [] };
		const mappingToSave = {};
		const expressions: ExpressionToBeExtracted[] = [];
		for (const k in action.Transformation.Mapping) {
			const v = action.Transformation.Mapping[k];
			if (v.value === '') {
				continue;
			}
			if (v.error && v.error !== '') {
				throw `Please fix the errors in the mapping`;
			}
			const property = flattenedOutputSchema![k];
			const fullProperty = property.full;
			const parentProperty = flattenedOutputSchema![property.root!].full;
			expressions.push({
				value: v.value,
				type: fullProperty!.type,
			});
			mappingToSave[k] = v.value;
			const isKeyPropertyAlreadyInSchema = outputSchema.properties!.find((p) => p.name === parentProperty!.name);
			if (!isKeyPropertyAlreadyInSchema) {
				outputSchema.properties!.push(parentProperty);
			}
		}
		let inputProperties: string[];
		try {
			inputProperties = await api.expressionsProperties(expressions, actionType.InputSchema);
		} catch (err) {
			throw err;
		}
		for (const prop of inputProperties) {
			const parentName = prop.split('.')[0];
			const isPropertyAlreadyInSchema = inputSchema.properties!.find((p) => p.name === parentName);
			if (!isPropertyAlreadyInSchema) {
				const fullProperty = flattenedInputSchema![parentName].full;
				inputSchema.properties!.push(fullProperty);
			}
		}
		mapping = mappingToSave;
		inSchema = inputSchema;
		outSchema = outputSchema;
	}

	if (action.Transformation.Function != null) {
		inSchema = actionType.InputSchema;
		outSchema = { name: 'Object', properties: [] };
		for (const property of actionType.OutputSchema.properties!) {
			const isIdentifier = isIdentifierProperty(property.name, anonymousIdentifiers.Priority);
			if (!isIdentifier) {
				outSchema.properties!.push(property);
			}
		}
		func = {
			Source: action.Transformation.Function.Source.trim(),
			Language: action.Transformation.Function.Language,
		};
	}

	if (connection.isSource && connection.isDatabase && actionType.Target === 'Users') {
		const idProperty = actionType.InputSchema.properties.find((property) => property.name === 'id');
		if (idProperty == null) {
			throw 'Schema must contain the "id" property';
		} else {
			const isIDPropertyAlreadyInSchema = inSchema.properties.findIndex((p) => p.name === 'id') !== -1;
			if (!isIDPropertyAlreadyInSchema) {
				inSchema.properties.push(idProperty);
			}
		}
		const timestampProperty = actionType.InputSchema.properties.find((property) => property.name === 'timestamp');
		if (timestampProperty != null) {
			const isTimestampPropertyAlreadyInSchema =
				inSchema.properties.findIndex((p) => p.name === 'timestamp') !== -1;
			if (!isTimestampPropertyAlreadyInSchema) {
				inSchema.properties.push(timestampProperty);
			}
		}
	}

	if (action.MatchingProperties != null) {
		const internal = action.MatchingProperties.Internal;
		if (internal === '' || action.MatchingProperties.External == null) {
			throw 'Matching properties cannot be empty';
		}
		const isInternalAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === internal) !== -1;
		if (!isInternalAlreadyInSchema) {
			const flattenedInputMatchingSchema = flattenSchema(actionType.InputMatchingSchema);
			inSchema.properties.push(flattenedInputMatchingSchema[internal].full);
		}
	}

	if (action.IdentityColumn != null && action.IdentityColumn !== '') {
		const isPropertyAlreadyInSchema =
			inSchema.properties!.findIndex((p) => p.name === action.IdentityColumn) !== -1;
		if (!isPropertyAlreadyInSchema) {
			const identityColumnProperty = flattenedInputSchema[action.IdentityColumn];
			if (identityColumnProperty == null) {
				throw 'Identity must be a valid property';
			}
			inSchema.properties.push(identityColumnProperty.full);
		}
	}

	if (action.TimestampColumn != null && action.TimestampColumn !== '') {
		const isPropertyAlreadyInSchema =
			inSchema.properties!.findIndex((p) => p.name === action.TimestampColumn) !== -1;
		if (!isPropertyAlreadyInSchema) {
			const timestampColumnProperty = flattenedInputSchema[action.TimestampColumn];
			if (timestampColumnProperty == null) {
				throw 'Timestamp must be a valid property';
			}
			inSchema.properties.push(timestampColumnProperty.full);
		}
	}

	if (action.Filter != null) {
		if (inSchema == null) {
			inSchema = { name: 'Object', properties: [] };
		}
		for (const condition of action.Filter.Conditions) {
			const propertyName = condition.Property;
			const isPropertyAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === propertyName) !== -1;
			if (!isPropertyAlreadyInSchema) {
				const property = flattenedInputSchema[propertyName];
				if (property == null) {
					throw 'Filter property must be a valid property';
				}
				inSchema.properties.push(property.full);
			}
		}
	}

	if (action.Query != null) {
		query = action.Query.trim();
	}

	const actionToSet: ActionToSet = {
		name: action.Name,
		enabled: action.Enabled,
		filter: action.Filter,
		inSchema: inSchema && inSchema.properties.length > 0 ? inSchema : null,
		outSchema: outSchema && outSchema.properties.length > 0 ? outSchema : null,
		transformation: {
			Mapping: mapping!,
			Function: func,
		},
		query: query!,
		path: action.Path,
		tableName: action.Table,
		sheet: action.Sheet,
		exportMode: action.ExportMode,
		IdentityColumn: action.IdentityColumn,
		TimestampColumn: action.TimestampColumn,
		TimestampFormat: action.TimestampFormat,
		matchingProperties: action.MatchingProperties,
	};

	return actionToSet;
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
		Transformation: {
			Mapping: flattenSchema(outputSchema),
			Function: null,
		},
		InSchema: null,
		OutSchema: null,
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

const computeActionTypeFields = (connection: TransformedConnection, actionType: ActionType) => {
	const fields: ActionTypeField[] = [];
	if (
		(connection.type === 'App' && connection.role === 'Destination' && actionType.Target === 'Events') ||
		(connection.type === 'Database' && connection.role === 'Destination')
	) {
		fields.push('Filter');
	}
	if (
		connection.type === 'App' ||
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

const doesTimestampNeedFormat = (timestampColumn: string, schema: ObjectType): boolean => {
	let needFormat = true;
	if (timestampColumn == null || timestampColumn === '') {
		needFormat = false;
	} else {
		const flatInputSchema = flattenSchema(schema);
		const timestampProperty = flatInputSchema[timestampColumn];
		if (timestampProperty == null) {
			needFormat = false;
		} else {
			const timestampType = timestampProperty.type;
			if (timestampType !== 'JSON' && timestampType !== 'Text') {
				needFormat = false;
			}
		}
	}
	return needFormat;
};

export {
	SCHEDULE_PERIODS,
	EXPORT_MODE_OPTIONS,
	flattenSchema,
	computeDefaultAction,
	computeActionTypeFields,
	transformActionType,
	transformAction,
	transformInActionToSet,
	isIdentifierProperty,
	doesTimestampNeedFormat,
};

export type { TransformedMapping, TransformedActionType, TransformedAction, ActionTypeField };
