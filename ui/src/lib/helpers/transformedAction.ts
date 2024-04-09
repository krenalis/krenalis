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
import { Filter, UIValues } from '../../types/external/api';
import { Compression } from '../../types/external/connection';
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

interface TransformedMatchingProperties {
	Internal: string;
	External: string;
}

type ActionTypeField =
	| 'DisplayedID'
	| 'Filter'
	| 'Mapping'
	| 'MatchingProperties'
	| 'ExportOnDuplicatedUsers'
	| 'ExportMode'
	| 'Query'
	| 'File'
	| 'Table';

interface TransformedActionType {
	Name: string;
	Description: string;
	Target: ActionTarget;
	EventType: string;
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
	UniqueIDColumn?: string | null;
	UpdatedAtColumn?: string | null;
	UpdatedAtFormat?: string | null;
	DisplayedID?: string | null;
	ExportMode?: ExportMode | null;
	MatchingProperties?: TransformedMatchingProperties | null;
	ExportOnDuplicatedUsers?: boolean | null;
	Compression?: Compression;
	Connector?: number;
}

const hasTransformationFunction = (action: ActionToSet) => {
	return action.transformation?.Function != null;
};

const hasValidTransformation = (action: ActionToSet) => {
	return (
		(hasTransformationFunction(action) || action.transformation?.Mapping != null) &&
		action.inSchema?.properties.length > 0 &&
		action.outSchema?.properties.length > 0
	);
};

const validateTransformation = (
	connection: TransformedConnection,
	actionType: TransformedActionType,
	action: ActionToSet,
) => {
	if (connection.isSource) {
		if (connection.isApp) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw new Error('Action must have a valid transformation');
				}
			}
		} else if (connection.isDatabase) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw new Error('Action must have a valid transformation');
				}
			}
		} else if (connection.isFileStorage) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw new Error('Action must have a valid transformation');
				}
			}
		} else if (connection.isMobile || connection.isServer || connection.isWebsite) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (hasValidTransformation(action)) {
					if (hasTransformationFunction(action)) {
						throw new Error(`Action supports only transformations via mapping`);
					}
				}
			}
			if (actionType.Target === 'Events') {
				if (hasValidTransformation(action)) {
					throw new Error('Action does not support transformations');
				}
			}
		}
	} else {
		if (connection.isApp) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw new Error('Action must have a valid transformation');
				}
			}
		} else if (connection.isDatabase) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw new Error('Action must have a valid transformation');
				}
			}
		} else if (connection.isFileStorage) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (hasValidTransformation(action)) {
					throw new Error('Action does not support transformations');
				}
			}
		}
	}
};

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
	let actionMapping = action.Transformation.Mapping;
	if (action.Transformation.Function == null && actionMapping == null) {
		// Mappings are selected but there is nothing mapped.
		actionMapping = {};
	}

	if (
		action.UpdatedAtFormat != null &&
		action.UpdatedAtFormat != '' &&
		action.UpdatedAtFormat.startsWith("'") &&
		action.UpdatedAtFormat.endsWith("'")
	) {
		action.UpdatedAtFormat = action.UpdatedAtFormat.substring(1, action.UpdatedAtFormat.length - 1);
	}

	let transformedMatchingProperties: TransformedMatchingProperties;
	if (action.MatchingProperties) {
		transformedMatchingProperties = {
			Internal: action.MatchingProperties.Internal,
			External: action.MatchingProperties.External.name,
		};
	}

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
			Mapping: actionMapping != null ? transformActionMapping(actionMapping, outputSchema) : null,
			Function: action.Transformation.Function,
		},
		Query: action.Query,
		Path: action.Path,
		Table: action.Table,
		Sheet: action.Sheet,
		UniqueIDColumn: action.UniqueIDColumn,
		UpdatedAtColumn: action.UpdatedAtColumn,
		UpdatedAtFormat: action.UpdatedAtFormat,
		DisplayedID: action.DisplayedID,
		ExportMode: action.ExportMode,
		MatchingProperties: transformedMatchingProperties,
		ExportOnDuplicatedUsers: action.ExportOnDuplicatedUsers,
		Connector: action.Connector,
		Compression: action.Compression,
	};
};

const transformInActionToSet = async (
	action: TransformedAction,
	values: UIValues,
	actionType: TransformedActionType,
	api: API,
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
		if (expressions.length > 0) {
			mapping = mappingToSave;
		}
		inSchema = inputSchema;
		outSchema = outputSchema;
	}

	if (action.Transformation.Function != null) {
		inSchema = actionType.InputSchema;
		outSchema = actionType.OutputSchema;
		func = {
			Source: action.Transformation.Function.Source.trim(),
			Language: action.Transformation.Function.Language,
		};
	}

	if (connection.isDestination && connection.isFileStorage && actionType.Target === 'Users') {
		outSchema = actionType.InputSchema;
	}

	let matchingProperties: MatchingProperties;
	if (action.MatchingProperties != null) {
		const internal = action.MatchingProperties.Internal;
		const external = action.MatchingProperties.External;
		if (internal === '' || external === '') {
			throw 'Matching properties cannot be empty';
		}

		const flattenedInputMatchingSchema = flattenSchema(actionType.InputMatchingSchema);
		const flattenedOutputMatchingSchema = flattenSchema(actionType.OutputMatchingSchema);

		const doesInternalExist = flattenedInputMatchingSchema[internal] != null;
		if (!doesInternalExist) {
			throw `Matching property "${internal}" does not exist`;
		}
		const doesExternalExist = flattenedOutputMatchingSchema[external] != null;
		if (!doesExternalExist) {
			throw `Matching property "${external}" does not exist`;
		}

		const fullExternalProperty = actionType.OutputMatchingSchema.properties.find((p) => p.name === external);
		matchingProperties = {
			Internal: internal,
			External: fullExternalProperty,
		};

		// Add the internal matching property to the in schema of the action.
		const isInternalAlreadyInActionSchema = inSchema.properties!.findIndex((p) => p.name === internal) !== -1;
		if (!isInternalAlreadyInActionSchema) {
			const flattenedInputMatchingSchema = flattenSchema(actionType.InputMatchingSchema);
			inSchema.properties.push(flattenedInputMatchingSchema[internal].full);
		}
	}

	let timestampFormat: string | undefined;
	if (connection.isSource && (connection.isDatabase || connection.isFileStorage)) {
		if (action.UniqueIDColumn == null || action.UniqueIDColumn === '') {
			throw 'User identifier cannot be empty';
		}
		const isAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === action.UniqueIDColumn) !== -1;
		if (!isAlreadyInSchema) {
			const uniqueIDColumnProperty = flattenedInputSchema[action.UniqueIDColumn];
			if (uniqueIDColumnProperty == null) {
				throw 'User identifier must be a valid property';
			}
			inSchema.properties.push(uniqueIDColumnProperty.full);
		}

		if (action.UpdatedAtColumn) {
			const isAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === action.UpdatedAtColumn) !== -1;
			if (!isAlreadyInSchema) {
				const timestampColumnProperty = flattenedInputSchema[action.UpdatedAtColumn];
				if (timestampColumnProperty == null) {
					throw 'Timestamp must be a valid property';
				}
				inSchema.properties.push(timestampColumnProperty.full);
			}
			if (
				action.UpdatedAtFormat &&
				action.UpdatedAtFormat !== 'ISO8601' &&
				action.UpdatedAtFormat !== 'Excel' &&
				action.UpdatedAtFormat !== 'DateTime' &&
				action.UpdatedAtFormat !== 'DateOnly'
			) {
				// wrap the format in single quotes.
				timestampFormat = `'${action.UpdatedAtFormat}'`;
			} else {
				timestampFormat = action.UpdatedAtFormat;
			}
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

	if (action.Sheet != null) {
		const s = action.Sheet;
		if (s.length < 1 || s.length > 31) {
			throw 'Sheet must be in range [1, 31]';
		}
		if (s.startsWith("'") || s.endsWith("'")) {
			throw 'Sheet must not start or end with a single quote';
		}
		const forbiddenChars = /[*\/:?[\]\\]/;
		if (forbiddenChars.test(s)) {
			throw 'Sheet must be valid';
		}
	}

	if (
		connection.isSource &&
		(connection.isMobile || connection.isServer || connection.isWebsite) &&
		(actionType.Target === 'Users' || actionType.Target === 'Groups')
	) {
		inSchema = null;
	}

	const actionToSet: ActionToSet = {
		name: action.Name,
		enabled: action.Enabled,
		filter: action.Filter,
		inSchema: inSchema && inSchema.properties.length > 0 ? inSchema : null,
		outSchema: outSchema && outSchema.properties.length > 0 ? outSchema : null,
		transformation: {
			Mapping: mapping,
			Function: func,
		},
		query: query!,
		path: action.Path,
		tableName: action.Table,
		sheet: action.Sheet,
		exportMode: action.ExportMode,
		UniqueIDColumn: action.UniqueIDColumn,
		UpdatedAtColumn: action.UpdatedAtColumn,
		UpdatedAtFormat: timestampFormat,
		DisplayedID: action.DisplayedID,
		matchingProperties: matchingProperties,
		exportOnDuplicatedUsers: action.ExportOnDuplicatedUsers,
		Compression: action.Compression,
		Connector: action.Connector,
		Settings: values,
	};

	try {
		validateTransformation(connection, actionType, actionToSet);
	} catch (err) {
		throw err;
	}

	return actionToSet;
};

const computeDefaultAction = (
	actionType: ActionType | TransformedActionType,
	connection: TransformedConnection,
	outputSchema: ObjectType,
	fields: ActionTypeField[],
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
	if (fields.includes('DisplayedID')) {
		action.DisplayedID = '';
	}
	if (fields.includes('Query')) {
		action.Query = connection.connector.sampleQuery;
	}
	if (fields.includes('File')) {
		action.Path = '';
		action.UniqueIDColumn = '';
		action.UpdatedAtColumn = '';
		action.UpdatedAtFormat = '';
		action.Sheet = null;
		action.Compression = '';
		action.Connector = 0;
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
	if (fields.includes('ExportOnDuplicatedUsers')) {
		action.ExportOnDuplicatedUsers = false;
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
		(connection.type === 'FileStorage' && connection.role === 'Source') ||
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
		fields.push('ExportOnDuplicatedUsers');
		fields.push('ExportMode');
		fields.push('Filter');
	}
	if (
		connection.role === 'Source' &&
		(connection.type === 'App' ||
			connection.type === 'FileStorage' ||
			connection.type === 'Database' ||
			connection.isEventBased)
	) {
		fields.push('DisplayedID');
	}
	if (connection.type === 'Database' && connection.role === 'Source') {
		fields.push('Query');
	}
	if (connection.type === 'FileStorage') {
		if (connection.role === 'Destination') {
			fields.push('Filter');
		}
		fields.push('File');
	}
	if (connection.type === 'Database' && connection.role === 'Destination') {
		fields.push('Table');
	}
	return fields;
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
	doesTimestampNeedFormat,
};

export type { TransformedMapping, TransformedActionType, TransformedAction, ActionTypeField };
