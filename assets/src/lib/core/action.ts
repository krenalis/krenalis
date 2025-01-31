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
	Matching,
	SchedulePeriod,
	TransformationFunction,
} from '../api/types/action';
import { ConnectorSettings } from '../api/types/responses';
import { Compression } from '../api/types/connection';
import Type, { ArrayType, FloatType, IntType, ObjectType, Property, TextType, UintType } from '../api/types/types';
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

const SCHEDULE_PERIODS: SchedulePeriod[] = ['Off', '5m', '15m', '30m', '1h', '2h', '3h', '6h', '8h', '12h', '24h'];

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
	size: number | null;
	full: Property;
	indentation?: number;
	root?: string;
	error?: string;
	disabled?: boolean;
}

type TransformedMapping = Record<string, TransformedProperty>;

interface TransformedTransformation {
	mapping: TransformedMapping | null;
	function: TransformationFunction | null;
}

type ActionTypeField =
	| 'Filter'
	| 'Transformation'
	| 'Matching'
	| 'ExportOnDuplicates'
	| 'ExportMode'
	| 'Query'
	| 'File'
	| 'OrderBy'
	| 'TableName';

interface TransformedActionType {
	name: string;
	description: string;
	target: ActionTarget;
	eventType: string;
	inputSchema: ObjectType;
	outputSchema: ObjectType;
	inputMatchingSchema: ObjectType | null;
	outputMatchingSchema: ObjectType | null;
	fields: ActionTypeField[];
}

interface TransformedAction {
	id?: number;
	connection?: number;
	target?: ActionTarget;
	name: string;
	enabled: boolean;
	eventType?: string | null;
	running?: boolean;
	scheduleStart?: number | null;
	schedulePeriod?: SchedulePeriod | null;
	inSchema: ObjectType | null;
	outSchema: ObjectType | null;
	filter: Filter | null;
	transformation: TransformedTransformation | null;
	query?: string | null;
	path?: string | null;
	table?: string | null;
	tableKey?: string | null;
	sheet?: string | null;
	identityProperty?: string | null;
	lastChangeTimeProperty?: string | null;
	lastChangeTimeFormat?: string | null;
	orderBy?: string | null;
	exportMode?: ExportMode | null;
	matching?: Matching | null;
	exportOnDuplicates?: boolean | null;
	compression?: Compression;
	format?: string;
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

const hasValidTransformation = (action: ActionToSet) => {
	return action.transformation?.function != null || action.transformation?.mapping != null;
};

const errInvalidTransformation = new Error('Action must have a valid transformation');
const errTransformationNotSupported = new Error('Action does not support transformations');

const validateTransformation = (
	connection: TransformedConnection,
	actionType: TransformedActionType,
	action: ActionToSet,
) => {
	if (connection.isSource) {
		if (connection.isApp) {
			if (actionType.target === 'Users' || actionType.target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isDatabase) {
			if (actionType.target === 'Users' || actionType.target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isFileStorage) {
			if (actionType.target === 'Users' || actionType.target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isMobile || connection.isServer || connection.isWebsite) {
			if (actionType.target === 'Events') {
				if (hasValidTransformation(action)) {
					throw errTransformationNotSupported;
				}
			}
		}
	} else {
		if (connection.isApp) {
			if (actionType.target === 'Users' || actionType.target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
			if (actionType.target === 'Events' && actionType.outputSchema == null) {
				if (hasValidTransformation(action)) {
					throw errTransformationNotSupported;
				}
			}
		} else if (connection.isDatabase) {
			if (actionType.target === 'Users' || actionType.target === 'Groups') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isFileStorage) {
			if (actionType.target === 'Users' || actionType.target === 'Groups') {
				if (hasValidTransformation(action)) {
					throw errTransformationNotSupported;
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
		name: actionType.name,
		description: actionType.description,
		target: actionType.target,
		eventType: actionType.eventType,
		inputSchema: inputSchema,
		outputSchema: outputSchema,
		inputMatchingSchema: inputMatchingSchema,
		outputMatchingSchema: outputMatchingSchema,
		fields: fields,
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
	let actionMapping = action.transformation.mapping;
	if (action.transformation.function == null && actionMapping == null) {
		// Mappings are selected but there is nothing mapped.
		actionMapping = {};
	}

	if (
		action.lastChangeTimeFormat != null &&
		action.lastChangeTimeFormat != '' &&
		action.lastChangeTimeFormat.startsWith("'") &&
		action.lastChangeTimeFormat.endsWith("'")
	) {
		action.lastChangeTimeFormat = action.lastChangeTimeFormat.substring(1, action.lastChangeTimeFormat.length - 1);
	}

	if (action.filter) {
		const conditions: FilterCondition[] = [];
		for (const condition of action.filter.conditions) {
			let cond = { ...condition };
			let values: string[] | null = [];
			if (condition.values == null) {
				values = null;
			} else {
				for (const v of condition.values) {
					const formatted = formatText(v);
					values.push(formatted);
				}
			}
			cond.values = values;
			conditions.push(cond);
		}
		action.filter.conditions = conditions;
	}

	return {
		id: action.id,
		connection: action.connection,
		target: action.target,
		name: action.name,
		enabled: action.enabled,
		eventType: action.eventType,
		running: action.running,
		scheduleStart: action.scheduleStart,
		schedulePeriod: action.schedulePeriod,
		inSchema: action.inSchema,
		outSchema: action.outSchema,
		filter: action.filter,
		transformation: {
			mapping: actionMapping != null ? transformActionMapping(actionMapping, outputSchema) : null,
			function: action.transformation.function,
		},
		query: action.query,
		path: action.path,
		table: action.table,
		tableKey: action.tableKey,
		sheet: action.sheet,
		identityProperty: action.identityProperty,
		lastChangeTimeProperty: action.lastChangeTimeProperty,
		lastChangeTimeFormat: action.lastChangeTimeFormat,
		exportMode: action.exportMode,
		matching: action.matching,
		exportOnDuplicates: action.exportOnDuplicates,
		format: action.format,
		compression: action.compression,
		orderBy: action.orderBy,
	};
};

const transformInActionToSet = async (
	action: TransformedAction,
	formatSettings: ConnectorSettings,
	actionType: TransformedActionType,
	api: API,
	connection: TransformedConnection,
	trimFunction: boolean,
	selectedInPaths: string[],
	selectedOutPaths: string[],
): Promise<ActionToSet> => {
	let mapping: Mapping;
	let inSchema: ObjectType;
	let outSchema: ObjectType;
	let func: TransformationFunction;
	let query: string;

	const isDestinationFileOnUsers =
		connection.isDestination && connection.isFileStorage && actionType.target === 'Users';

	const flattenedInputSchema = flattenSchema(actionType.inputSchema);
	const flattenedOutputSchema = flattenSchema(actionType.outputSchema);

	// Remove the placeholders from the output schema.
	for (const p in flattenedOutputSchema) {
		if (flattenedOutputSchema[p].full.placeholder) {
			delete flattenedOutputSchema[p].full.placeholder;
		}
	}

	const allowsConstantTransformation =
		(connection.isSource && connection.isEventBased && actionType.target === 'Users') ||
		(connection.isDestination && connection.isApp && actionType.target === 'Events');
	if (action.transformation.mapping != null) {
		const inputSchema: ObjectType = { name: 'Object', properties: [] };
		const outputSchema: ObjectType = { name: 'Object', properties: [] };
		const mappingToSave = {};
		const expressions: ExpressionToBeExtracted[] = [];

		const keys = Object.keys(action.transformation.mapping);
		for (const k of keys) {
			// The property must be mapped if it is required and it is a
			// first-level property, or one of its siblings has been
			// mapped.
			const p = action.transformation.mapping[k];
			if (p.value === '') {
				const hasRequired =
					action.exportMode != null &&
					((p.createRequired && action.exportMode.includes('Create')) ||
						(p.updateRequired && action.exportMode.includes('Update')));

				const isFirstLevel = p.indentation === 0;

				const siblings: string[] = [];
				for (const key of keys) {
					const prop = action.transformation.mapping[key];
					if (prop.root === p.root && prop.indentation === p.indentation && key !== k) {
						siblings.push(key);
					}
				}
				const hasMappedSiblings =
					siblings.findIndex((k) => action.transformation.mapping[k].value !== '') !== -1;

				const isRequired = hasRequired && (isFirstLevel || hasMappedSiblings);
				const isInMatching = action.matching != null && action.matching.out === k;
				if (isRequired && !isInMatching) {
					throw new Error(`Property "${k}" is required. Indicate an expression for this property.`);
				}
				continue;
			}
			if (p.error && p.error !== '') {
				throw new Error(`Please fix the errors in the mapping`);
			}
			const property = flattenedOutputSchema![k];
			const fullProperty = property.full;
			const parentProperty = flattenedOutputSchema![property.root!].full;
			expressions.push({
				value: p.value,
				type: fullProperty!.type,
			});
			mappingToSave[k] = p.value;
			const isKeyPropertyAlreadyInSchema = outputSchema.properties!.find((p) => p.name === parentProperty!.name);
			if (!isKeyPropertyAlreadyInSchema) {
				outputSchema.properties!.push(parentProperty);
			}
		}
		let inputProperties: string[];
		try {
			inputProperties = await api.expressionsProperties(expressions, actionType.inputSchema);
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
		if (inputProperties.length === 0 && !allowsConstantTransformation) {
			throw new Error(
				'There are no properties in the mapping expressions; use at least one property in an expression',
			);
		}
		if (expressions.length > 0) {
			mapping = mappingToSave;
		}
		inSchema = inputSchema;
		outSchema = outputSchema;
	} else if (action.transformation.function != null) {
		const inputSchema: ObjectType = { name: 'Object', properties: [] };
		const outputSchema: ObjectType = { name: 'Object', properties: [] };

		const inPaths: string[] = [];
		for (const p of selectedInPaths) {
			// Add the property to the input schema of the action.
			const property = flattenedInputSchema![p];
			const parentProperty = flattenedInputSchema![property.root!].full;
			const alreadyInSchema = inputSchema.properties!.find((p) => p.name === parentProperty!.name);
			if (!alreadyInSchema) {
				inputSchema.properties!.push(parentProperty);
			}
			// Add the property to the input properties of the
			// transformation function.
			const isParentSelected = selectedInPaths.findIndex((prop) => p.startsWith(`${prop}.`)) !== -1;
			if (isParentSelected) {
				continue;
			}
			inPaths.push(p);
		}

		if (inPaths.length === 0 && !allowsConstantTransformation) {
			throw new Error('You must select at least one input property');
		}

		const outPaths: string[] = [];
		for (const p of selectedOutPaths) {
			// Add the property to the output schema of the action.
			const property = flattenedOutputSchema![p];
			const parentProperty = flattenedOutputSchema![property.root!].full;
			const alreadyInSchema = outputSchema.properties!.find((p) => p.name === parentProperty!.name);
			if (!alreadyInSchema) {
				outputSchema.properties!.push(parentProperty);
			}
			// Add the property to the output properties of the
			// transformation function.
			const isParentSelected = selectedOutPaths.findIndex((prop) => p.startsWith(`${prop}.`)) !== -1;
			if (isParentSelected) {
				continue;
			}
			outPaths.push(p);
		}

		if (outPaths.length === 0) {
			throw new Error('You must select at least one output property');
		}

		const keys = Object.keys(flattenedOutputSchema);
		for (const k of keys) {
			// The property must be selected if it is required and it is
			// a first-level property, or one of its siblings has been
			// selected.
			const p = flattenedOutputSchema[k];

			const isSelected = selectedOutPaths.findIndex((prop) => prop === k) !== -1;
			const isParentSelected =
				selectedOutPaths.findIndex((prop) => {
					k.startsWith(`${prop}.`);
				}) !== -1;

			const hasRequired =
				action.exportMode != null &&
				((p.createRequired && action.exportMode.includes('Create')) ||
					(p.updateRequired && action.exportMode.includes('Update')));

			const isFirstLevel = p.indentation === 0;

			const selectedSiblings: string[] = [];
			const parentName = k.slice(0, k.lastIndexOf('.'));
			for (const path of selectedOutPaths) {
				const hasSameParent = path.startsWith(`${parentName}.`);
				if (hasSameParent) {
					const suffix = path.slice(`${parentName}.`.length);
					const isLowerLevel = suffix.includes('.');
					if (!isLowerLevel) {
						selectedSiblings.push(path);
					}
				}
			}

			const isRequired = hasRequired && (isFirstLevel || selectedSiblings.length > 0);
			const isInMatching = action.matching != null && action.matching.out === k;
			if (isRequired && !isSelected && !isParentSelected && !isInMatching) {
				throw new Error(`Property "${k}" is required and you must pass it in the transformation function`);
			}
			continue;
		}

		let source = action.transformation.function.source;
		if (trimFunction) {
			source = source.trim();
		}

		func = {
			source: source,
			language: action.transformation.function.language,
			preserveJSON: action.transformation.function.preserveJSON,
			inPaths: inPaths,
			outPaths: outPaths,
		};

		inSchema = inputSchema;
		outSchema = outputSchema;
	} else if (isDestinationFileOnUsers) {
		inSchema = actionType.inputSchema;
		outSchema = null; // TODO(Gianluca): it this necessary?
	}

	if (action.matching != null) {
		const inMatching = action.matching.in;
		const outMatching = action.matching.out;
		if (inMatching === '' || outMatching === '') {
			throw new Error('Matching properties cannot be empty');
		}

		const flattenedInputMatchingSchema = flattenSchema(actionType.inputMatchingSchema);
		const flattenedOutputMatchingSchema = flattenSchema(actionType.outputMatchingSchema);

		// Check that the properties used for matching actually exist in
		// the corresponding schemas.
		const inMatchingProperty = flattenedInputMatchingSchema[inMatching];
		const doesInExist = inMatchingProperty != null;
		if (!doesInExist) {
			throw new Error(`Matching property "${inMatching}" does not exist`);
		}

		const outMatchingProperty = flattenedOutputMatchingSchema[outMatching];
		const doesOutExist = outMatchingProperty != null;
		if (!doesOutExist) {
			throw new Error(`Matching property "${outMatching}" does not exist`);
		}

		// Check that properties have types supported for matching and
		// that the in matching property is convertible to the out
		// matching property.
		try {
			validateMatching(inMatchingProperty.full, outMatchingProperty.full);
		} catch (err) {
			throw err;
		}

		// Add the in matching property to the input schema of the
		// action, if it does not already exist.
		const exists = inSchema.properties!.findIndex((p) => p.name === inMatching) !== -1;
		if (!exists) {
			inSchema.properties.push(flattenedInputMatchingSchema[inMatching].full);
		}

		// Add the out matching property to the output schema of the action.
		{
			// The out matching property must necessarily also be
			// contained in the output schema of the connection in the
			// case where the mode is "CreateOnly" or "CreateOrUpdate",
			// whereas it may not be there in the case of ‘UpdateOnly’.
			const a = outMatchingProperty.full;
			const b = flattenedOutputSchema[outMatching]?.full;
			const existsInOutputSchema =
				b != null && outPathsTypesAreEqual(a.type, b.type) && a.nullable === b.nullable;
			let p: Property;
			if (existsInOutputSchema) {
				p = {
					...b,
					readOptional: a.readOptional,
				};
			} else {
				if (action.exportMode === 'CreateOnly' || action.exportMode === 'CreateOrUpdate') {
					throw new Error(`External matching property "${outMatching}" does not exist`);
				} else {
					p = a;
				}
			}
			const isAlreadyInSchema = outSchema.properties!.findIndex((p) => p.name === outMatching) !== -1;
			if (isAlreadyInSchema) {
				throw new Error(`External matching property cannot be used in the transformation`);
			}
			outSchema.properties.push(p);
		}
	}

	if (connection.isSource && (connection.isDatabase || connection.isFileStorage)) {
		if (action.identityProperty == null || action.identityProperty === '') {
			throw new Error('User identifier cannot be empty');
		}
		const isAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === action.identityProperty) !== -1;
		if (!isAlreadyInSchema) {
			const identityProperty = flattenedInputSchema[action.identityProperty];
			if (identityProperty == null) {
				throw new Error('User identifier must be a valid property');
			}
			inSchema.properties.push(identityProperty.full);
		}

		if (action.lastChangeTimeProperty) {
			const isAlreadyInSchema =
				inSchema.properties!.findIndex((p) => p.name === action.lastChangeTimeProperty) !== -1;
			if (!isAlreadyInSchema) {
				const lastChangeTimeProperty = flattenedInputSchema[action.lastChangeTimeProperty];
				if (lastChangeTimeProperty == null) {
					throw new Error('LastChangeTimeProperty must be a valid property');
				}
				inSchema.properties.push(lastChangeTimeProperty.full);
			}
			if (doesLastChangeTimePropertyNeedFormat(action.lastChangeTimeProperty, actionType.inputSchema)) {
				if (action.lastChangeTimeFormat !== 'ISO8601' && action.lastChangeTimeFormat !== 'Excel') {
					// the format is custom.
					try {
						validateCustomLastChangeTimeFormat(action.lastChangeTimeFormat);
					} catch (err) {
						throw err;
					}
				}
			}
		}
	}

	let filter: Filter = null;
	if (action.filter != null) {
		if (inSchema == null) {
			inSchema = { name: 'Object', properties: [] };
		}

		let f = { logical: action.filter.logical, conditions: [] };

		// Exclude conditions that have empty properties.
		let conditions = action.filter.conditions.filter((condition) => condition.property !== '');

		for (const condition of conditions) {
			const propertyName = condition.property;
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

			if (condition.operator == '') {
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
			if (isJsonOrText && condition.values != null) {
				for (const [i, v] of condition.values.entries()) {
					if ((i === 0 && v === '') || (i === 1 && v === '' && isBetweenOperator(condition.operator))) {
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
				values = condition.values;
			}

			try {
				validateFilterConditionValues(property.full.type, condition.values, propertyName);
			} catch (err) {
				throw err;
			}

			const rootProperty = flattenedInputSchema[property.root].full;
			const isPropertyAlreadyInSchema =
				inSchema.properties!.findIndex((p) => p.name === rootProperty.name) !== -1;
			if (!isPropertyAlreadyInSchema) {
				inSchema.properties.push(rootProperty);
			}

			const c: FilterCondition = { property: condition.property, operator: condition.operator, values: values };
			f.conditions.push(c);
		}

		if (f.conditions.length > 0) {
			filter = f;
		}
	}

	if (action.query != null) {
		query = action.query.trim();
	}

	if (action.sheet != null) {
		const s = action.sheet;
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

	if (action.orderBy != null) {
		const p = action.orderBy;
		if (p === '') {
			throw new Error('File ordering property cannot be empty');
		}
		const filteredSchema = filterOrderingPropertySchema(actionType.inputSchema);
		if (filteredSchema != null) {
			if (filteredSchema[p] == null) {
				throw new Error(`File ordering property "${p}" does not exist in the user schema`);
			}
		}
	}

	const isDatabaseExportOnUsers =
		connection.connector.type === 'Database' && connection.role === 'Destination' && actionType.target === 'Users';

	if (isDatabaseExportOnUsers) {
		// the table key must be defined for database type actions that
		// export users.
		if (action.tableKey == null || action.tableKey === '') {
			throw new Error('Table key cannot be empty');
		}

		// the table key must be a valid property.
		const property = flattenedOutputSchema[action.tableKey];
		if (property == null) {
			throw new Error('Table key must be a valid property');
		}

		// the table key must necessarily be transformed.
		if (mapping == null && func == null) {
			throw new Error('Table key must be transformed');
		} else if (mapping != null) {
			if (!Object.keys(mapping).includes(action.tableKey)) {
				throw new Error('Table key must be transformed');
			}
		} else if (func != null) {
			if (!func.outPaths.includes(action.tableKey)) {
				throw new Error('Table key must be transformed');
			}
		}

		// ensure that the table key is always non-nullable.
		for (let i = 0; i < outSchema.properties.length; i++) {
			const p = outSchema.properties[i];
			if (p.name === action.tableKey) {
				p.nullable = false;
				break;
			}
		}
	} else {
		// the table key must be empty for actions that are not
		// database type actions that export users.
		if (action.tableKey != null && action.tableKey !== '') {
			throw new Error('Table key must be empty for this kind of action');
		}
	}

	// In cases where the input schema refers to events, that is when:
	//
	//  - user identities are imported from events
	//  - events are imported into the data warehouse
	//  - events are dispatched to apps
	//
	// the input schema must be nil, which means the schema of the events.
	let importEventsIntoWarehouse = connection.isSource && connection.isEventBased && actionType.target == 'Events';
	let dispatchEventsToApps =
		connection.isDestination && connection.connector.type == 'App' && actionType.target == 'Events';
	let importIdentitiesFromEvents = connection.isSource && connection.isEventBased && actionType.target == 'Users';
	if (importIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps) {
		inSchema = null;
	}

	const actionToSet: ActionToSet = {
		name: action.name,
		enabled: action.enabled,
		filter: filter,
		inSchema: inSchema && inSchema.properties.length > 0 ? inSchema : null,
		outSchema: outSchema && outSchema.properties.length > 0 ? outSchema : null,
		transformation: {
			mapping: mapping,
			function: func,
		},
		query: query!,
		path: action.path,
		tableName: action.table,
		tableKey: action.tableKey,
		sheet: action.sheet,
		identityProperty: action.identityProperty,
		lastChangeTimeProperty: action.lastChangeTimeProperty,
		lastChangeTimeFormat: action.lastChangeTimeFormat,
		compression: action.compression,
		orderBy: action.orderBy,
		format: action.format,
		formatSettings: formatSettings,
	};

	if (action.matching != null) {
		actionToSet.matching = action.matching;
	}

	if (action.exportOnDuplicates != null) {
		actionToSet.exportOnDuplicates = action.exportOnDuplicates;
	}

	if (action.exportMode != null) {
		actionToSet.exportMode = action.exportMode;
	}

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
		name: actionType.name,
		// The action is enabled by default only for batch operations importing or exporting users.
		enabled:
			actionType.target == 'Users' && (connection.isApp || connection.isDatabase || connection.isFileStorage),
		filter: null,
		transformation: {
			mapping: flattenSchema(outputSchema),
			function: null,
		},
		inSchema: null,
		outSchema: null,
	};
	if (fields.includes('Query')) {
		action.query = connection.connector.asSource.sampleQuery;
		action.identityProperty = '';
		action.lastChangeTimeProperty = '';
		action.lastChangeTimeFormat = '';
	}
	if (fields.includes('File')) {
		action.path = '';
		action.identityProperty = '';
		action.lastChangeTimeProperty = '';
		action.lastChangeTimeFormat = '';
		action.sheet = null;
		action.compression = '';
		action.format = '';
	}
	if (fields.includes('OrderBy')) {
		action.orderBy = '';
	}
	if (fields.includes('TableName')) {
		action.table = '';
		action.tableKey = '';
	}
	if (fields.includes('ExportMode')) {
		action.exportMode = Object.keys(EXPORT_MODE_OPTIONS)[0] as ExportMode;
	}
	if (fields.includes('Matching')) {
		action.matching = { in: '', out: '' };
	}
	if (fields.includes('ExportOnDuplicates')) {
		action.exportOnDuplicates = false;
	}
	return action;
};

const computeActionTypeFields = (
	connection: TransformedConnection,
	actionType: ActionType,
	outpuSchema: ObjectType,
) => {
	const fields: ActionTypeField[] = [];

	// Filters are always allowed except for actions that import users
	// from databases.
	if (!(connection.role === 'Source' && connection.connector.type === 'Database' && actionType.target === 'Users')) {
		fields.push('Filter');
	}

	if (connection.connector.type === 'App') {
		if (connection.role === 'Source') {
			fields.push('Transformation');
		} else {
			if (actionType.target === 'Users' || actionType.target === 'Groups') {
				fields.push('Transformation');
			} else if (actionType.target === 'Events' && outpuSchema != null) {
				fields.push('Transformation');
			}
		}
	} else if (connection.connector.type === 'Database') {
		fields.push('Transformation');
	} else if (connection.connector.type === 'FileStorage' && connection.role === 'Source') {
		fields.push('Transformation');
	} else if (
		connection.connector.type === 'Mobile' ||
		connection.connector.type === 'Server' ||
		connection.connector.type === 'Website'
	) {
		if (connection.role === 'Source' && (actionType.target === 'Users' || actionType.target === 'Groups')) {
			fields.push('Transformation');
		}
	}

	if (
		connection.connector.type === 'App' &&
		connection.role === 'Destination' &&
		(actionType.target === 'Users' || actionType.target === 'Groups')
	) {
		fields.push('Matching');
		fields.push('ExportOnDuplicates');
		fields.push('ExportMode');
		fields.push('Filter');
	}

	if (connection.connector.type === 'Database' && connection.role === 'Source') {
		fields.push('Query');
	}

	if (connection.connector.type === 'FileStorage') {
		if (connection.role === 'Destination') {
			fields.push('Filter');
			if (actionType.target === 'Users') {
				fields.push('OrderBy');
			}
		}
		fields.push('File');
	}

	if (connection.connector.type === 'Database' && connection.role === 'Destination') {
		fields.push('TableName');
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
	if (isSourceEventConnection(connection.role, connection.connector.type) && actionType.target === 'Users') {
		return 'event';
	}
	if (actionType.target === 'Users') {
		return 'user';
	} else if (actionType.target === 'Events') {
		return 'event';
	} else if (actionType.target === 'Groups') {
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

const validateMatching = (inMatching: Property, outMatching: Property) => {
	const inTyp = inMatching.type.name;
	if (inTyp !== 'Int' && inTyp !== 'Uint' && inTyp !== 'Text' && inTyp !== 'UUID') {
		throw new Error(`Matching property cannot be of type "${inTyp}"`);
	}

	// Check that the in property can be converted to the type of the
	// out property.
	const exTyp = outMatching.type.name;
	const conversionError = new Error(`Matching property of type "${inTyp}" cannot be converted to type "${exTyp}"`);

	if (inTyp === 'Int') {
		if (exTyp !== 'Int' && exTyp !== 'Uint' && exTyp !== 'Text') {
			throw conversionError;
		}
	} else if (inTyp === 'Uint') {
		if (exTyp !== 'Int' && exTyp !== 'Uint' && exTyp !== 'Text') {
			throw conversionError;
		}
	} else if (inTyp === 'Text') {
		if (exTyp !== 'Int' && exTyp !== 'Uint' && exTyp !== 'UUID' && exTyp !== 'Text') {
			throw conversionError;
		}
	} else if (inTyp === 'UUID') {
		if (exTyp !== 'UUID' && exTyp !== 'Text') {
			throw conversionError;
		}
	}
};

const outPathsTypesAreEqual = (externalTyp: Type, outTyp: Type): boolean => {
	if (externalTyp.name !== outTyp.name) {
		return false;
	}

	if (externalTyp.name === 'Int' || externalTyp.name === 'Uint') {
		const outT = outTyp as IntType | UintType;
		return (
			externalTyp.bitSize === outT.bitSize &&
			externalTyp.minimum === outT.minimum &&
			externalTyp.maximum === outT.maximum
		);
	} else if (externalTyp.name === 'Text') {
		const outT = outTyp as TextType;
		return (
			externalTyp.byteLen === outT.byteLen &&
			externalTyp.charLen === outT.charLen &&
			externalTyp.regexp === outT.regexp &&
			JSON.stringify(externalTyp.values) === JSON.stringify(outT.values)
		);
	}

	return true;
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
	validateMatching,
	outPathsTypesAreEqual,
};

export type { TransformedMapping, TransformedProperty, TransformedActionType, TransformedAction, ActionTypeField };
