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
} from '../api/types/action';
import { Filter, ConnectorValues } from '../api/types/responses';
import { Compression } from '../api/types/connection';
import { FloatType, IntType, ObjectType, Property, UintType } from '../api/types/types';
import API from '../api/api';
import TransformedConnection from './connection';

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
	| 'Filter'
	| 'Transformation'
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
	IdentityProperty?: string | null;
	LastChangeTimeProperty?: string | null;
	LastChangeTimeFormat?: string | null;
	FileOrderingPropertyPath?: string | null;
	ExportMode?: ExportMode | null;
	MatchingProperties?: TransformedMatchingProperties | null;
	ExportOnDuplicatedUsers?: boolean | null;
	Compression?: Compression;
	Connector?: string;
}

const hasTransformationFunction = (action: ActionToSet) => {
	return action.transformation?.Function != null;
};

const hasTransformationMapping = (action: ActionToSet) => {
	return action.transformation?.Mapping != null;
};

const hasSchemas = (action: ActionToSet) => {
	if (action.inSchema == null || action.outSchema == null) {
		return false;
	}
	return action.inSchema.properties.length > 0 && action.outSchema.properties.length > 0;
};

const hasEmptyMapping = (action: ActionToSet) => {
	return (
		(!hasTransformationMapping(action) && !hasTransformationFunction(action)) ||
		(hasTransformationMapping(action) && !hasSchemas(action))
	);
};

const hasValidTransformation = (action: ActionToSet) => {
	return (hasTransformationFunction(action) || hasTransformationMapping(action)) && hasSchemas(action);
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
					if (hasEmptyMapping(action)) {
						throw new Error(
							'There are no properties in the mapping expressions; use at least one property in an expression',
						);
					}
					throw new Error('Action must have a valid transformation');
				}
			}
		} else if (connection.isDatabase) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (!hasValidTransformation(action)) {
					if (hasEmptyMapping(action)) {
						throw new Error(
							'There are no properties in the mapping expressions; use at least one property in an expression',
						);
					}
					throw new Error('Action must have a valid transformation');
				}
			}
		} else if (connection.isFileStorage) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (!hasValidTransformation(action)) {
					if (hasEmptyMapping(action)) {
						throw new Error(
							'There are no properties in the mapping expressions; use at least one property in an expression',
						);
					}
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
					if (hasEmptyMapping(action)) {
						throw new Error(
							'There are no properties in the mapping expressions; use at least one property in an expression',
						);
					}
					throw new Error('Action must have a valid transformation');
				}
			}
		} else if (connection.isDatabase) {
			if (actionType.Target === 'Users' || actionType.Target === 'Groups') {
				if (!hasValidTransformation(action)) {
					if (hasEmptyMapping(action)) {
						throw new Error(
							'There are no properties in the mapping expressions; use at least one property in an expression',
						);
					}
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
		action.LastChangeTimeFormat != null &&
		action.LastChangeTimeFormat != '' &&
		action.LastChangeTimeFormat.startsWith("'") &&
		action.LastChangeTimeFormat.endsWith("'")
	) {
		action.LastChangeTimeFormat = action.LastChangeTimeFormat.substring(1, action.LastChangeTimeFormat.length - 1);
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
		IdentityProperty: action.IdentityProperty,
		LastChangeTimeProperty: action.LastChangeTimeProperty,
		LastChangeTimeFormat: action.LastChangeTimeFormat,
		FileOrderingPropertyPath: action.FileOrderingPropertyPath,
		ExportMode: action.ExportMode,
		MatchingProperties: transformedMatchingProperties,
		ExportOnDuplicatedUsers: action.ExportOnDuplicatedUsers,
		Connector: action.Connector,
		Compression: action.Compression,
	};
};

const transformInActionToSet = async (
	action: TransformedAction,
	uiValues: ConnectorValues,
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
				throw new Error(`Please fix the errors in the mapping`);
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
			throw new Error('Matching properties cannot be empty');
		}

		const flattenedInputMatchingSchema = flattenSchema(actionType.InputMatchingSchema);
		const flattenedOutputMatchingSchema = flattenSchema(actionType.OutputMatchingSchema);

		const doesInternalExist = flattenedInputMatchingSchema[internal] != null;
		if (!doesInternalExist) {
			throw new Error(`Matching property "${internal}" does not exist`);
		}
		const doesExternalExist = flattenedOutputMatchingSchema[external] != null;
		if (!doesExternalExist) {
			throw new Error(`Matching property "${external}" does not exist`);
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

	if (connection.isSource && (connection.isDatabase || connection.isFileStorage)) {
		if (action.IdentityProperty == null || action.IdentityProperty === '') {
			throw new Error('User identifier cannot be empty');
		}
		const isAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === action.IdentityProperty) !== -1;
		if (!isAlreadyInSchema) {
			const identityProperty = flattenedInputSchema[action.IdentityProperty];
			if (identityProperty == null) {
				throw new Error('User identifier must be a valid property');
			}
			inSchema.properties.push(identityProperty.full);
		}

		if (action.LastChangeTimeProperty) {
			const isAlreadyInSchema =
				inSchema.properties!.findIndex((p) => p.name === action.LastChangeTimeProperty) !== -1;
			if (!isAlreadyInSchema) {
				const lastChangeTimeProperty = flattenedInputSchema[action.LastChangeTimeProperty];
				if (lastChangeTimeProperty == null) {
					throw new Error('LastChangeTimeProperty must be a valid property');
				}
				inSchema.properties.push(lastChangeTimeProperty.full);
			}
			if (doesLastChangeTimePropertyNeedFormat(action.LastChangeTimeProperty, actionType.InputSchema)) {
				if (action.LastChangeTimeFormat !== 'ISO8601' && action.LastChangeTimeFormat !== 'Excel') {
					// the format is custom.
					try {
						validateCustomLastChangeTimeFormat(action.LastChangeTimeFormat);
					} catch (err) {
						throw err;
					}
				}
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
					throw new Error('Filter property must be a valid property');
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
			throw new Error('Sheet must be in range [1, 31]');
		}
		if (s.startsWith("'") || s.endsWith("'")) {
			throw new Error('Sheet must not start or end with a single quote');
		}
		const forbiddenChars = /[*\/:?[\]\\]/;
		if (forbiddenChars.test(s)) {
			throw new Error('Sheet must be valid');
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
		FileOrderingPropertyPath: action.FileOrderingPropertyPath,
		exportMode: action.ExportMode,
		IdentityProperty: action.IdentityProperty,
		LastChangeTimeProperty: action.LastChangeTimeProperty,
		LastChangeTimeFormat: action.LastChangeTimeFormat,
		matchingProperties: matchingProperties,
		exportOnDuplicatedUsers: action.ExportOnDuplicatedUsers,
		Compression: action.Compression,
		Connector: action.Connector,
		UIValues: uiValues,
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
	if (fields.includes('Query')) {
		action.Query = connection.connector.sampleQuery;
		action.IdentityProperty = '';
		action.LastChangeTimeProperty = '';
		action.LastChangeTimeFormat = '';
	}
	if (fields.includes('File')) {
		action.Path = '';
		action.IdentityProperty = '';
		action.LastChangeTimeProperty = '';
		action.LastChangeTimeFormat = '';
		action.Sheet = null;
		action.Compression = '';
		action.FileOrderingPropertyPath = '';
		action.Connector = '';
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
		(connection.type === 'Database' && connection.role === 'Destination') ||
		((connection.type === 'Mobile' || connection.type === 'Server' || connection.type === 'Website') &&
			connection.role === 'Source' &&
			(actionType.Target === 'Users' || actionType.Target === 'Groups'))
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
		fields.push('Transformation');
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

const doesLastChangeTimePropertyNeedFormat = (lastChangeTimeProperty: string, schema: ObjectType): boolean => {
	if (lastChangeTimeProperty == null || lastChangeTimeProperty === '') {
		return false;
	}
	const flatInputSchema = flattenSchema(schema);
	const p = flatInputSchema[lastChangeTimeProperty];
	if (p == null) {
		return false;
	}
	const type = p.type;
	return type === 'JSON' || type === 'Text';
};

const validateCustomLastChangeTimeFormat = (format: string) => {
	if (format === '') {
		throw new Error('Last change time format cannot be empty');
	}
	if (Array.from(format).length > 64) {
		throw new Error('Last change time format is longer than 64 characters');
	}
	if (!format.includes('%')) {
		throw new Error(`Last change time format "${format}" is not a valid format`);
	}
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
	doesLastChangeTimePropertyNeedFormat,
};

export type { TransformedMapping, TransformedProperty, TransformedActionType, TransformedAction, ActionTypeField };
