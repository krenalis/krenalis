import {
	Action,
	ActionTarget,
	ActionToSet,
	ActionType,
	ExportMode,
	ExpressionToBeExtracted,
	Filter,
	FilterCondition,
	FilterOperator,
	Mapping,
	MatchingProperties,
	SchedulePeriod,
	TransformationFunction,
	TransformationPurpose,
} from '../api/types/action';
import { ConnectorValues } from '../api/types/responses';
import { Compression } from '../api/types/connection';
import Type, { ArrayType, FloatType, IntType, ObjectType, Property, UintType } from '../api/types/types';
import API from '../api/api';
import TransformedConnection, { isSourceEventConnection } from './connection';
import { filterOrderingPropertySchema } from '../../components/helpers/getSchemaComboboxItems';
import {
	formatText,
	isDate,
	isDateTime,
	isDecimal,
	isFloat,
	isInet,
	isInt,
	isUint,
	isUUID,
	isValidPropertyPath,
	isYear,
	parseText,
} from '../../utils/filters';

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

const FILTER_OPERATORS: FilterOperator[] = [
	'is',
	'is not',
	'is less than',
	'is less than or equal to',
	'is greater than',
	'is greater than or equal to',
	'is between',
	'is not between',
	'contains',
	'does not contain',
	'is one of',
	'is not one of',
	'starts with',
	'ends with',
	'is before',
	'is on or before',
	'is after',
	'is on or after',
	'is true',
	'is false',
	'is null',
	'is not null',
	'exists',
	'does not exist',
];

const typesByFilterOperator: string[][] = [
	['Int', 'Uint', 'Float', 'Decimal', 'DateTime', 'Date', 'Time', 'Year', 'UUID', 'JSON', 'Inet', 'Text'], // is
	['Int', 'Uint', 'Float', 'Decimal', 'DateTime', 'Date', 'Time', 'Year', 'UUID', 'JSON', 'Inet', 'Text'], // is not
	['Int', 'Uint', 'Float', 'Decimal', 'JSON', 'Text'], // is less than
	['Int', 'Uint', 'Float', 'Decimal', 'JSON', 'Text'], // is less than or equal to
	['Int', 'Uint', 'Float', 'Decimal', 'JSON', 'Text'], // is greater than
	['Int', 'Uint', 'Float', 'Decimal', 'JSON', 'Text'], // is greater than or equal to
	['Int', 'Uint', 'Float', 'Decimal', 'Year', 'DateTime', 'Date', 'Time', 'JSON', 'Text'], // is between
	['Int', 'Uint', 'Float', 'Decimal', 'Year', 'DateTime', 'Date', 'Time', 'JSON', 'Text'], // is not between
	['JSON', 'Text', 'Array'], // contains
	['JSON', 'Text', 'Array'], // does not contain
	['Int', 'Uint', 'Float', 'Decimal', 'Year', 'DateTime', 'Date', 'Time', 'JSON', 'Text'], // is one of
	['Int', 'Uint', 'Float', 'Decimal', 'Year', 'DateTime', 'Date', 'Time', 'JSON', 'Text'], // is not one of
	['JSON', 'Text'], // starts with
	['JSON', 'Text'], // ends with
	['DateTime', 'Date', 'Time', 'Year'], // is before
	['DateTime', 'Date', 'Time', 'Year'], // is on or before
	['DateTime', 'Date', 'Time', 'Year'], // is after
	['DateTime', 'Date', 'Time', 'Year'], // is on or after
	['Boolean', 'JSON'], // is true
	['Boolean', 'JSON'], // is false
	[
		'Boolean',
		'Int',
		'Uint',
		'Float',
		'Decimal',
		'DateTime',
		'Date',
		'Year',
		'Time',
		'UUID',
		'JSON',
		'Inet',
		'Text',
		'Array',
		'Object',
		'Map',
	], // is null
	[
		'Boolean',
		'Int',
		'Uint',
		'Float',
		'Decimal',
		'DateTime',
		'Date',
		'Year',
		'Time',
		'UUID',
		'JSON',
		'Inet',
		'Text',
		'Array',
		'Object',
		'Map',
	], // is not null
	['JSON'], // exists
	['JSON'], // does not exist
];

type TransformedExportMode = 'Create and update' | 'Create only' | 'Update only';

const EXPORT_MODE_OPTIONS: Record<ExportMode, TransformedExportMode> = {
	CreateOrUpdate: 'Create and update',
	CreateOnly: 'Create only',
	UpdateOnly: 'Update only',
};

interface TransformedProperty {
	value: string;
	createRequired: boolean;
	updateRequired: boolean;
	readOptional: boolean;
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
	| 'FileOrderingProperty'
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
	TableKeyProperty?: string | null;
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

const getCompatibleFilterOperators = (property: TransformedProperty): number[] => {
	if (property == null) {
		return [];
	}
	const operators: number[] = [];
	for (const i of Object.keys(FILTER_OPERATORS)) {
		// 'is null' and 'is not null' are compatible only with nullable
		// properties or JSON type properties.
		if (FILTER_OPERATORS[i] === 'is null' || FILTER_OPERATORS[i] === 'is not null') {
			const isNullable = property.full.nullable === true;
			const isJSON = property.type === 'JSON';
			if (!isNullable && !isJSON) {
				continue;
			}
		}

		// 'contains' and 'does not contain' should only be shown if the type of
		// the Array element is supported by the 'is' operator.
		if (
			(FILTER_OPERATORS[i] === 'contains' || FILTER_OPERATORS[i] === 'does not contain') &&
			property.type === 'Array'
		) {
			const elementType = (property.full.type as ArrayType).elementType;
			const isOperatorIndex = FILTER_OPERATORS.findIndex((op) => op === 'is');
			if (!typesByFilterOperator[isOperatorIndex].includes(elementType.name)) {
				continue;
			}
		}

		if (typesByFilterOperator[i].includes(property.type)) {
			operators.push(Number(i));
		}
	}
	return operators;
};

const isUnaryOperator = (operator: string): boolean => {
	return (
		operator === 'is true' ||
		operator === 'is false' ||
		operator === 'is null' ||
		operator === 'is not null' ||
		operator === 'exists' ||
		operator === 'does not exist'
	);
};

const isBetweenOperator = (operator: string): boolean => {
	return operator === 'is between' || operator === 'is not between';
};

const isOneOfOperator = (operator: string): boolean => {
	return operator === 'is one of' || operator === 'is not one of';
};

const splitPropertyAndPath = (propertyName: string, flatSchema: TransformedMapping): [string, string] => {
	const name = propertyName.trim();

	const split = name.split('.');
	let base = '';
	for (const s of split) {
		let b = '';
		if (base === '') {
			b = s;
		} else {
			b = `${base}.${s}`;
		}
		if (flatSchema[b] != null) {
			base = b;
		} else {
			break;
		}
	}

	let path = '';
	if (base !== '') {
		path = name.replace(base, '');
		if (path !== '' && path.startsWith('.')) {
			// remove the initial period.
			path = path.slice(1);
		}
	}

	const property = flatSchema[base];
	const isJSON = property?.type === 'JSON';
	if (!isJSON && path !== '') {
		// handle cases where the user has typed an invalid subproperty and for
		// this reason the subproperty name was incorrectly considered as a
		// path.
		return ['', ''];
	}

	return [base, path];
};

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
			readOptional: property.readOptional,
			createRequired: property.createRequired,
			updateRequired: property.updateRequired,
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

	if (action.Filter) {
		const conditions: FilterCondition[] = [];
		for (const condition of action.Filter.Conditions) {
			let cond = { ...condition };
			let values: string[] | null = [];
			if (condition.Values == null) {
				values = null;
			} else {
				for (const v of condition.Values) {
					const formatted = formatText(v);
					values.push(formatted);
				}
			}
			cond.Values = values;
			conditions.push(cond);
		}
		action.Filter.Conditions = conditions;
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
		TableKeyProperty: action.TableKeyProperty,
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
	trimFunction: boolean = false,
): Promise<ActionToSet> => {
	let mapping: Mapping;
	let inSchema: ObjectType;
	let outSchema: ObjectType;
	let func: TransformationFunction;
	let query: string;

	const isDestinationFileOnUsers =
		connection.isDestination && connection.isFileStorage && actionType.Target === 'Users';

	const flattenedInputSchema = flattenSchema(actionType.InputSchema);
	const flattenedOutputSchema = flattenSchema(actionType.OutputSchema);

	if (action.Transformation.Mapping != null) {
		const inputSchema: ObjectType = { name: 'Object', properties: [] };
		const outputSchema: ObjectType = { name: 'Object', properties: [] };
		const mappingToSave = {};
		const expressions: ExpressionToBeExtracted[] = [];
		const purpose: TransformationPurpose =
			action.ExportMode != null && action.ExportMode === 'UpdateOnly' ? 'Update' : 'Create';
		for (const k in action.Transformation.Mapping) {
			const v = action.Transformation.Mapping[k];
			if (v.value === '') {
				if (purpose === 'Update' && v.updateRequired) {
					throw new Error(
						`Property "${k}" is required for the update. Indicate an expression for this property.`,
					);
				} else if (purpose === 'Create' && v.createRequired) {
					throw new Error(
						`Property "${k}" is required for creation. Indicate an expression for this property.`,
					);
				}
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
	} else if (action.Transformation.Function != null) {
		inSchema = actionType.InputSchema;
		outSchema = actionType.OutputSchema;
		if (action.MatchingProperties?.External) {
			// recompute the out schema to prevent updates in place on the
			// version used by the UI.
			const s = { name: 'Object', properties: [] };
			for (const p of outSchema.properties) {
				if (p.name !== action.MatchingProperties.External) {
					s.properties.push(p);
				}
			}
			outSchema = s as ObjectType;
		}
		let source = action.Transformation.Function.Source;
		if (trimFunction) {
			source = source.trim();
		}
		func = {
			Source: source,
			Language: action.Transformation.Function.Language,
			PreserveJSON: action.Transformation.Function.PreserveJSON,
			InProperties: inSchema.properties === null ? [] : inSchema.properties.map((p) => p.name),
			OutProperties: outSchema.properties!.map((p) => p.name),
		};
	} else if (isDestinationFileOnUsers) {
		inSchema = actionType.InputSchema;
		outSchema = null; // TODO(Gianluca): it this necessary?
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

		const fullExternalProperty = flattenedOutputMatchingSchema[external].full;
		matchingProperties = {
			Internal: internal,
			External: fullExternalProperty,
		};

		// Add the internal matching property to the in schema of the action.
		const isInternalAlreadyInActionSchema = inSchema.properties!.findIndex((p) => p.name === internal) !== -1;
		if (!isInternalAlreadyInActionSchema) {
			inSchema.properties.push(flattenedInputMatchingSchema[internal].full);
		}

		if (action.ExportMode === 'CreateOnly' || action.ExportMode === 'CreateOrUpdate') {
			// add the external matching property (not directly the one from the
			// output matching schema, but instead the corresponding "writable"
			// one in the output schema of the transformation) to the out schema
			// of the action.
			let externalPropertyToAdd: Property;
			const p = flattenSchema(actionType.OutputSchema)[external]?.full;
			if (p?.type.name === fullExternalProperty.type.name) {
				externalPropertyToAdd = p;
			}
			if (externalPropertyToAdd == null) {
				throw new Error(`External matching property "${external}" does not exist in the output schema`);
			}
			const isAlreadyInSchema = outSchema.properties!.findIndex((p) => p.name === external) !== -1;
			if (isAlreadyInSchema) {
				throw new Error(`External matching property cannot be used in the transformation`);
			}
			if (!isAlreadyInSchema) {
				outSchema.properties.push(externalPropertyToAdd);
			}
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

	let filter: Filter = null;
	if (action.Filter != null) {
		if (inSchema == null) {
			inSchema = { name: 'Object', properties: [] };
		}

		let f = { Logical: action.Filter.Logical, Conditions: [] };

		// Exclude conditions that have empty properties.
		let conditions = action.Filter.Conditions.filter((condition) => condition.Property !== '');

		for (const condition of conditions) {
			const propertyName = condition.Property;
			const [base, path] = splitPropertyAndPath(propertyName, flattenedInputSchema);
			const property = flattenedInputSchema[base];

			if (property == null) {
				throw new Error(`Property "${propertyName}" does not exist`);
			}

			if (property.type === 'JSON' && path.trim() !== '') {
				const isValid = isValidPropertyPath(path);
				if (!isValid) {
					throw new Error(`Property path "${path}" of filter condition is not valid`);
				}
			}

			if (condition.Operator == '') {
				throw new Error(`Operator of filter condition is required`);
			}

			let isJsonOrText = property.type === 'JSON' || property.type === 'Text';
			if (property.type === 'Array') {
				const typ = property.full.type as ArrayType;
				if (typ.elementType.name === 'JSON' || typ.elementType.name === 'Text') {
					isJsonOrText = true;
				}
			}

			let values: string[] | null = [];
			if (isJsonOrText && condition.Values != null) {
				for (const [i, v] of condition.Values.entries()) {
					if ((i === 0 && v === '') || (i === 1 && v === '' && isBetweenOperator(condition.Operator))) {
						throw new Error(`The filter value on the property "${propertyName}" cannot be empty`);
					}
					if (v === '') {
						// discard empty values.
						continue;
					}
					let parsed: string;
					try {
						parsed = parseText(v);
					} catch (err) {
						throw new Error(`Value "${v}" of filter condition is not valid: ${err.message}`);
					}
					values.push(parsed);
				}
			} else {
				values = condition.Values;
			}

			try {
				validateFilterConditionValues(property.full.type, condition.Values, propertyName);
			} catch (err) {
				throw err;
			}

			const rootProperty = flattenedInputSchema[property.root].full;
			const isPropertyAlreadyInSchema =
				inSchema.properties!.findIndex((p) => p.name === rootProperty.name) !== -1;
			if (!isPropertyAlreadyInSchema) {
				inSchema.properties.push(rootProperty);
			}

			const c: FilterCondition = { Property: condition.Property, Operator: condition.Operator, Values: values };
			f.Conditions.push(c);
		}

		if (f.Conditions.length > 0) {
			filter = f;
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

	if (action.FileOrderingPropertyPath != null) {
		const p = action.FileOrderingPropertyPath;
		if (p === '') {
			throw new Error('File ordering property cannot be empty');
		}
		const filteredSchema = filterOrderingPropertySchema(actionType.InputSchema);
		if (filteredSchema != null) {
			if (filteredSchema[p] == null) {
				throw new Error(`File ordering property "${p}" does not exist in the user schema`);
			}
		}
	}

	const isDatabaseExportOnUsers =
		connection.type === 'Database' && connection.role === 'Destination' && actionType.Target === 'Users';

	if (isDatabaseExportOnUsers) {
		// the table key property must be defined for database type actions that
		// export users.
		if (action.TableKeyProperty == null || action.TableKeyProperty === '') {
			throw new Error('Table key property cannot be empty');
		}

		// the table key property must be a valid property.
		const property = flattenedOutputSchema[action.TableKeyProperty];
		if (property == null) {
			throw new Error('Table key property must be a valid property');
		}

		// the table key property must necessarily be transformed.
		if (mapping == null && func == null) {
			throw new Error('Table key property must be transformed');
		} else if (mapping != null) {
			if (!Object.keys(mapping).includes(action.TableKeyProperty)) {
				throw new Error('Table key property must be transformed');
			}
		} else if (func != null) {
			if (!func.OutProperties.includes(action.TableKeyProperty)) {
				throw new Error('Table key property must be transformed');
			}
		}

		// ensure that the properties are required for creation in the output schema,
		// and that the table key property is nullable.
		for (let i = 0; i < outSchema.properties.length; i++) {
			const p = outSchema.properties[i];
			p.updateRequired = true;
			if (p.name === action.TableKeyProperty) {
				p.nullable = false;
			}
		}
	} else {
		// the table key property must be empty for actions that are not
		// database type actions that export users.
		if (action.TableKeyProperty != null && action.TableKeyProperty !== '') {
			throw new Error('Table key property must be empty for this kind of action');
		}
	}

	// In cases where the input schema refers to events, that is when:
	//
	//  - user identities are imported from events
	//  - events are imported into the data warehouse
	//  - events are dispatched to apps
	//
	// the input schema must be nil, which means the schema of the events.
	let importEventsIntoWarehouse = connection.isSource && connection.isEventBased && actionType.Target == 'Events';
	let dispatchEventsToApps = connection.isDestination && connection.type == 'App' && actionType.Target == 'Events';
	let importIdentitiesFromEvents = connection.isSource && connection.isEventBased && actionType.Target == 'Users';
	if (importIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps) {
		inSchema = null;
	}

	const actionToSet: ActionToSet = {
		name: action.Name,
		enabled: action.Enabled,
		filter: filter,
		inSchema: inSchema && inSchema.properties.length > 0 ? inSchema : null,
		outSchema: outSchema && outSchema.properties.length > 0 ? outSchema : null,
		transformation: {
			Mapping: mapping,
			Function: func,
		},
		query: query!,
		path: action.Path,
		tableName: action.Table,
		tableKeyProperty: action.TableKeyProperty,
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
		action.Connector = '';
	}
	if (fields.includes('FileOrderingProperty')) {
		action.FileOrderingPropertyPath = '';
	}
	if (fields.includes('Table')) {
		action.Table = '';
		action.TableKeyProperty = '';
	}
	if (fields.includes('ExportMode')) {
		action.ExportMode = Object.keys(EXPORT_MODE_OPTIONS)[0] as ExportMode;
	}
	if (fields.includes('MatchingProperties')) {
		action.MatchingProperties = { Internal: '', External: '' };
	}
	if (fields.includes('ExportOnDuplicatedUsers')) {
		action.ExportOnDuplicatedUsers = false;
	}
	return action;
};

const computeActionTypeFields = (connection: TransformedConnection, actionType: ActionType) => {
	const fields: ActionTypeField[] = [];
	// Filters are always allowed except for actions that import users from
	// databases.
	if (!(connection.role === 'Source' && connection.type === 'Database' && actionType.Target === 'Users')) {
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
			if (actionType.Target === 'Users') {
				fields.push('FileOrderingProperty');
			}
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

const getTransformationFunctionParameterName = (
	connection: TransformedConnection,
	actionType: TransformedActionType,
): String => {
	if (isSourceEventConnection(connection.role, connection.type) && actionType.Target === 'Users') {
		return 'event';
	}
	if (actionType.Target === 'Users') {
		return 'user';
	} else if (actionType.Target === 'Events') {
		return 'event';
	} else if (actionType.Target === 'Groups') {
		return 'group';
	}
};

const validateFilterConditionValues = (type: Type, values: string[] | null, propertyName: string) => {
	const throwIfInvalid = (isValid: boolean, typeName: string) => {
		if (!isValid) {
			throw new Error(`The filter value on the property "${propertyName}" is not a valid ${typeName}`);
		}
	};

	if (values == null) {
		return;
	}

	for (const v of values) {
		if (type.name === 'Int') {
			throwIfInvalid(isInt(v), type.name);
		} else if (type.name === 'Uint') {
			throwIfInvalid(isUint(v), type.name);
		} else if (type.name === 'Float') {
			throwIfInvalid(isFloat(v, type.bitSize), type.name);
		} else if (type.name === 'Decimal') {
			throwIfInvalid(isDecimal(v), type.name);
		} else if (type.name === 'DateTime') {
			throwIfInvalid(isDateTime(v), type.name);
		} else if (type.name === 'Date') {
			throwIfInvalid(isDate(v), type.name);
		} else if (type.name === 'Year') {
			throwIfInvalid(isYear(v), type.name);
		} else if (type.name === 'UUID') {
			throwIfInvalid(isUUID(v), type.name);
		} else if (type.name === 'Inet') {
			throwIfInvalid(isInet(v), type.name);
		} else if (type.name === 'Array') {
			if (type.elementType.name !== 'JSON' && type.elementType.name !== 'Text') {
				validateFilterConditionValues(type.elementType, [v], propertyName);
			}
		}
	}
};

export {
	SCHEDULE_PERIODS,
	FILTER_OPERATORS,
	EXPORT_MODE_OPTIONS,
	flattenSchema,
	computeDefaultAction,
	computeActionTypeFields,
	transformActionType,
	transformAction,
	transformInActionToSet,
	getCompatibleFilterOperators,
	isUnaryOperator,
	isBetweenOperator,
	isOneOfOperator,
	splitPropertyAndPath,
	doesLastChangeTimePropertyNeedFormat,
	getTransformationFunctionParameterName,
};

export type { TransformedMapping, TransformedProperty, TransformedActionType, TransformedAction, ActionTypeField };
