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
import Type, {
	ArrayType,
	FloatType,
	IntType,
	MapType,
	ObjectType,
	Property,
	TextType,
	UintType,
} from '../api/types/types';
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
	['int', 'uint', 'float', 'decimal', 'datetime', 'date', 'time', 'year', 'uuid', 'json', 'inet', 'text'], // is
	['int', 'uint', 'float', 'decimal', 'datetime', 'date', 'time', 'year', 'uuid', 'json', 'inet', 'text'], // is not
	['int', 'uint', 'float', 'decimal', 'json', 'text'], // is less than
	['int', 'uint', 'float', 'decimal', 'json', 'text'], // is less than or equal to
	['int', 'uint', 'float', 'decimal', 'json', 'text'], // is greater than
	['int', 'uint', 'float', 'decimal', 'json', 'text'], // is greater than or equal to
	['int', 'uint', 'float', 'decimal', 'year', 'datetime', 'date', 'time', 'json', 'text'], // is between
	['int', 'uint', 'float', 'decimal', 'year', 'datetime', 'date', 'time', 'json', 'text'], // is not between
	['json', 'text', 'array'], // contains
	['json', 'text', 'array'], // does not contain
	['int', 'uint', 'float', 'decimal', 'year', 'datetime', 'date', 'time', 'json', 'text'], // is one of
	['int', 'uint', 'float', 'decimal', 'year', 'datetime', 'date', 'time', 'json', 'text'], // is not one of
	['json', 'text'], // starts with
	['json', 'text'], // ends with
	['datetime', 'date', 'time', 'year'], // is before
	['datetime', 'date', 'time', 'year'], // is on or before
	['datetime', 'date', 'time', 'year'], // is after
	['datetime', 'date', 'time', 'year'], // is on or after
	['boolean', 'json'], // is true
	['boolean', 'json'], // is false
	[
		'boolean',
		'int',
		'uint',
		'float',
		'decimal',
		'datetime',
		'date',
		'year',
		'time',
		'uuid',
		'json',
		'inet',
		'text',
		'array',
		'object',
		'map',
	], // is null
	[
		'boolean',
		'int',
		'uint',
		'float',
		'decimal',
		'datetime',
		'date',
		'year',
		'time',
		'uuid',
		'json',
		'inet',
		'text',
		'array',
		'object',
		'map',
	], // is not null
	['json'], // exists
	['json'], // does not exist
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
	| 'UpdateOnDuplicates'
	| 'ExportMode'
	| 'Query'
	| 'File'
	| 'OrderBy'
	| 'TableName'
	| 'Incremental';

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
	target: ActionTarget;
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
	tableName?: string | null;
	tableKey?: string | null;
	sheet?: string | null;
	identityColumn?: string | null;
	lastChangeTimeColumn?: string | null;
	lastChangeTimeFormat?: string | null;
	incremental?: boolean | null;
	orderBy?: string | null;
	exportMode?: ExportMode | null;
	matching?: Matching | null;
	updateOnDuplicates?: boolean | null;
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
		// properties or json type properties.
		if (FILTER_OPERATORS[i] === 'is null' || FILTER_OPERATORS[i] === 'is not null') {
			const isNullable = property.full.nullable === true;
			const isJSON = property.type === 'json';
			if (!isNullable && !isJSON) {
				continue;
			}
		}

		// 'contains' and 'does not contain' should only be shown if the type of
		// the array element is supported by the 'is' operator.
		if (
			(FILTER_OPERATORS[i] === 'contains' || FILTER_OPERATORS[i] === 'does not contain') &&
			property.type === 'array'
		) {
			const elementType = (property.full.type as ArrayType).elementType;
			const isOperatorIndex = FILTER_OPERATORS.findIndex((op) => op === 'is');
			if (!typesByFilterOperator[isOperatorIndex].includes(elementType.kind)) {
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
	const isJSON = property?.type === 'json';
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
			if (actionType.target === 'User' || actionType.target === 'Group') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isDatabase) {
			if (actionType.target === 'User' || actionType.target === 'Group') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isFileStorage) {
			if (actionType.target === 'User' || actionType.target === 'Group') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isSDK) {
			if (actionType.target === 'Event') {
				if (hasValidTransformation(action)) {
					throw errTransformationNotSupported;
				}
			}
		}
	} else {
		if (connection.isApp) {
			if (actionType.target === 'User' || actionType.target === 'Group') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
			if (actionType.target === 'Event' && actionType.outputSchema == null) {
				if (hasValidTransformation(action)) {
					throw errTransformationNotSupported;
				}
			}
		} else if (connection.isDatabase) {
			if (actionType.target === 'User' || actionType.target === 'Group') {
				if (!hasValidTransformation(action)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isFileStorage) {
			if (actionType.target === 'User' || actionType.target === 'Group') {
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
const flattenSchema = (typ: ObjectType | ArrayType | MapType): TransformedMapping | null => {
	if (typ == null || !isRecursiveType(typ)) {
		return null;
	}

	const flattenProperty = (property: Property): TransformedProperty => {
		const flat = {
			value: property.placeholder || '',
			readOptional: property.readOptional,
			createRequired: property.createRequired,
			updateRequired: property.updateRequired,
			type: property.type.kind,
			size: null,
			full: { ...property },
		};
		if (flat.type === 'int' || flat.type === 'uint' || flat.type === 'float') {
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
			if (property.type.kind === 'object') {
				const flattenedProperties = flattenSubProperties(name, parentIndentation, property.type.properties!);
				flattenedSubProperties = { ...flattenedSubProperties, ...flattenedProperties };
			}
		}
		return flattenedSubProperties;
	};

	const clonedTyp = structuredClone(typ);

	let properties: Property[] = [];
	if (clonedTyp.kind === 'object') {
		properties = clonedTyp.properties;
	} else {
		const t = clonedTyp as ArrayType | MapType;
		const elementTyp = t.elementType as ObjectType;
		properties = elementTyp.properties;
	}

	let flattenedSchema = {};
	for (const property of properties) {
		const indentation = 0;
		const flattened = flattenProperty(property);
		flattened.indentation = indentation;
		flattened.root = property.name;
		flattenedSchema[property.name] = flattened;
		if (property.type.kind === 'object') {
			const flattenedSubProperties = flattenSubProperties(property.name, indentation, property.type.properties!);
			flattenedSchema = { ...flattenedSchema, ...flattenedSubProperties };
		}
	}

	return flattenedSchema;
};

const isRecursiveType = (type: Type): boolean => {
	return (
		type.kind === 'object' || ((type.kind === 'array' || type.kind === 'map') && type.elementType.kind === 'object')
	);
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
	const s = flattenSchema(outputSchema)!;
	for (const path in s) {
		const value = mapping[path];
		if (value != null && value !== '') {
			s[path].value = value;
			// Disable the properties in the hierarchy.
			const { ancestors: parents, descendants: children } = getHierarchicalPaths(path, s);
			for (const p of [...parents, ...children]) {
				s[p].disabled = true;
			}
		}
	}
	return s;
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
		tableName: action.tableName,
		tableKey: action.tableKey,
		sheet: action.sheet,
		identityColumn: action.identityColumn,
		lastChangeTimeColumn: action.lastChangeTimeColumn,
		lastChangeTimeFormat: action.lastChangeTimeFormat,
		incremental: action.incremental,
		exportMode: action.exportMode,
		matching: action.matching,
		updateOnDuplicates: action.updateOnDuplicates,
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
		connection.isDestination && connection.isFileStorage && actionType.target === 'User';

	const flattenedInputSchema = flattenSchema(actionType.inputSchema);
	const flattenedOutputSchema = flattenSchema(actionType.outputSchema);

	// Remove the placeholders from the output schema.
	for (const p in flattenedOutputSchema) {
		if (flattenedOutputSchema[p].full.placeholder) {
			delete flattenedOutputSchema[p].full.placeholder;
		}
	}

	const allowsConstantTransformation =
		(connection.isSource && connection.isEventBased && actionType.target === 'User') ||
		(connection.isDestination && connection.isApp && actionType.target === 'Event');

	if (action.transformation.mapping != null) {
		const inputSchema: ObjectType = { kind: 'object', properties: [] };
		const outputSchema: ObjectType = { kind: 'object', properties: [] };
		const mappingToSave = {};
		const expressions: ExpressionToBeExtracted[] = [];

		const keys = Object.keys(action.transformation.mapping);
		for (const k of keys) {
			// The property must be mapped if it is required and it is a
			// first-level property, or if one of its siblings has been
			// mapped.
			const pair = action.transformation.mapping[k];
			const isFirstLevel = pair.indentation === 0;
			if (pair.value === '') {
				const hasRequired =
					(actionType.target === 'Event' && (pair.createRequired || pair.updateRequired)) ||
					(action.exportMode != null &&
						((pair.createRequired && action.exportMode.includes('Create')) ||
							(pair.updateRequired && action.exportMode.includes('Update'))));

				const siblings: string[] = [];
				for (const key of keys) {
					const p = action.transformation.mapping[key];
					if (p.root === pair.root && p.indentation === pair.indentation && key !== k) {
						siblings.push(key);
					}
				}
				const hasMappedSiblings =
					siblings.findIndex((k) => action.transformation.mapping[k].value !== '') !== -1;

				const isRequired = hasRequired && (isFirstLevel || hasMappedSiblings);
				const isInMatching = action.matching != null && action.matching.out === k; // Check if is used in the matching properties.
				if (isRequired && !isInMatching) {
					throw new Error(`Property "${k}" is required. Indicate an expression for this property.`);
				}
				continue;
			}

			// Check if there are UI errors in the mapping.
			if (pair.error && pair.error !== '') {
				throw new Error(`Please fix the errors in the mapping`);
			}

			const mapped = flattenedOutputSchema![k].full;
			expressions.push({
				value: pair.value,
				type: mapped!.type,
			});

			mappingToSave[k] = pair.value;

			addPropertyToSchema(k, mapped, outputSchema, flattenedOutputSchema, isFirstLevel);
		}

		let inputPaths: string[];
		try {
			inputPaths = await api.expressionsProperties(expressions, actionType.inputSchema);
		} catch (err) {
			throw err;
		}
		if (inputPaths.length === 0 && !allowsConstantTransformation) {
			throw new Error(
				'There are no properties in the mapping expressions; use at least one property in an expression',
			);
		}
		for (const path of inputPaths) {
			const property = flattenedInputSchema[path].full;
			addPropertyToSchema(path, property, inputSchema, flattenedInputSchema, !path.includes('.'));
		}

		if (expressions.length > 0) {
			mapping = mappingToSave;
		}

		inSchema = inputSchema;
		outSchema = outputSchema;
	} else if (action.transformation.function != null) {
		const inputSchema: ObjectType = { kind: 'object', properties: [] };
		const outputSchema: ObjectType = { kind: 'object', properties: [] };

		const inPaths: string[] = [];
		for (const p of selectedInPaths) {
			// Add the property to the input schema of the action.
			const property = flattenedInputSchema![p].full;
			addPropertyToSchema(p, property, inputSchema, flattenedInputSchema, !p.includes('.'));

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
			const property = flattenedOutputSchema![p].full;
			addPropertyToSchema(p, property, outputSchema, flattenedOutputSchema, !p.includes('.'));

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
			// contained in the destination schema in the case where the
			// mode is "CreateOnly" or "CreateOrUpdate", whereas it may
			// not be there in the case of ‘UpdateOnly’.
			const a = outMatchingProperty.full;
			const b = flattenedOutputSchema[outMatching]?.full;
			const existsInOutputSchema =
				b != null && propertyTypesAreEqual(a.type, b.type) && a.nullable === b.nullable;
			let p: Property;
			if (existsInOutputSchema) {
				p = {
					...b,
					readOptional: a.readOptional,
				};
			} else {
				if (action.exportMode === 'CreateOnly' || action.exportMode === 'CreateOrUpdate') {
					throw new Error(
						`${actionType.target} cannot be created but can be updated, as the "${action.matching.out}" property of ${connection.name} is read-only`,
					);
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
		if (action.identityColumn == null || action.identityColumn === '') {
			throw new Error('User identifier cannot be empty');
		}
		const isAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === action.identityColumn) !== -1;
		if (!isAlreadyInSchema) {
			const identityColumn = flattenedInputSchema[action.identityColumn];
			if (identityColumn == null) {
				throw new Error('User identifier must be a valid property');
			}
			inSchema.properties.push(identityColumn.full);
		}

		if (action.lastChangeTimeColumn) {
			const isAlreadyInSchema =
				inSchema.properties!.findIndex((p) => p.name === action.lastChangeTimeColumn) !== -1;
			if (!isAlreadyInSchema) {
				const lastChangeTimeColumn = flattenedInputSchema[action.lastChangeTimeColumn];
				if (lastChangeTimeColumn == null) {
					throw new Error('Last change time must be a valid column');
				}
				inSchema.properties.push(lastChangeTimeColumn.full);
			}
			if (doesLastChangeTimeColumnNeedFormat(action.lastChangeTimeColumn, actionType.inputSchema)) {
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
			inSchema = { kind: 'object', properties: [] };
		}

		let f = { logical: action.filter.logical, conditions: [] };

		// Exclude conditions that have empty properties.
		let conditions = action.filter.conditions.filter((condition) => condition.property !== '');

		for (const condition of conditions) {
			const propertyName = condition.property;
			const [base, path] = splitPropertyAndPath(propertyName, flattenedInputSchema);
			const property = flattenedInputSchema[base];

			if (property == null) {
				throw new Error(`Property "${propertyName}" of filter condition does not exist`);
			}

			if (property.type === 'json' && path.trim() !== '') {
				const isValid = isValidPropertyPath(path);
				if (!isValid) {
					throw new Error(`Property path "${path}" of filter condition is not valid`);
				}
			}

			if (condition.operator == '') {
				throw new Error(`Operator of filter condition is required`);
			}

			let isJsonOrText = property.type === 'json' || property.type === 'text';
			if (property.type === 'array') {
				const typ = property.full.type as ArrayType;
				if (typ.elementType.kind === 'json' || typ.elementType.kind === 'text') {
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
		connection.connector.type === 'Database' && connection.role === 'Destination' && actionType.target === 'User';

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

		// the table key must necessarily be transformed and there must
		// be another transformed property in addition to it.
		if (mapping == null && func == null) {
			throw new Error('Table key must be transformed');
		} else if (mapping != null) {
			const mappedPaths = Object.keys(mapping);
			if (!mappedPaths.includes(action.tableKey)) {
				throw new Error('Table key must be transformed');
			}
			if (mappedPaths.length === 1) {
				throw new Error('Another property must be transformed in addition to the table key property');
			}
		} else if (func != null) {
			if (!func.outPaths.includes(action.tableKey)) {
				throw new Error('Table key must be transformed');
			}
			if (func.outPaths.length === 1) {
				throw new Error('Another property must be transformed in addition to the table key property');
			}
		}

		// ensure that the table key is always create required, non-update
		// required and non-nullable.
		for (let i = 0; i < outSchema.properties.length; i++) {
			const p = outSchema.properties[i];
			if (p.name === action.tableKey) {
				p.createRequired = true;
				p.updateRequired = false;
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
	let importEventsIntoWarehouse = connection.isSource && connection.isEventBased && actionType.target == 'Event';
	let dispatchEventsToApps =
		connection.isDestination && connection.connector.type == 'App' && actionType.target == 'Event';
	let importIdentitiesFromEvents = connection.isSource && connection.isEventBased && actionType.target == 'User';
	if (importIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps) {
		inSchema = null;
	}

	let incremental = action.incremental;
	if (connection.isSource && (connection.isDatabase || connection.isFileStorage)) {
		// If last change time is not set the import cannot be
		// incremental.
		if (action.lastChangeTimeColumn === '') {
			incremental = false;
		}
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
		tableName: action.tableName,
		tableKey: action.tableKey,
		sheet: action.sheet,
		identityColumn: action.identityColumn,
		lastChangeTimeColumn: action.lastChangeTimeColumn,
		lastChangeTimeFormat: action.lastChangeTimeFormat,
		incremental: incremental,
		compression: action.compression,
		orderBy: action.orderBy,
		format: action.format,
		formatSettings: formatSettings,
	};

	if (action.matching != null) {
		actionToSet.matching = action.matching;
	}

	if (action.updateOnDuplicates != null) {
		let updateOnDuplicates = action.updateOnDuplicates;
		if (!action.exportMode.includes('Update')) {
			// If export mode is "CreateOnly", `updateOnDuplicates` is
			// not taken into consideration.
			updateOnDuplicates = false;
		}
		actionToSet.updateOnDuplicates = updateOnDuplicates;
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
		target: actionType.target,
		name: actionType.name,
		// The action is enabled by default only for batch operations importing or exporting users.
		enabled:
			actionType.target == 'User' && (connection.isApp || connection.isDatabase || connection.isFileStorage),
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
		action.identityColumn = '';
		action.lastChangeTimeColumn = '';
		action.lastChangeTimeFormat = '';
	}
	if (fields.includes('File')) {
		action.path = '';
		action.identityColumn = '';
		action.lastChangeTimeColumn = '';
		action.lastChangeTimeFormat = '';
		action.sheet = null;
		action.compression = '';
		action.format = '';
	}
	if (fields.includes('OrderBy')) {
		action.orderBy = '';
	}
	if (fields.includes('TableName')) {
		action.tableName = '';
		action.tableKey = '';
	}
	if (fields.includes('ExportMode')) {
		action.exportMode = Object.keys(EXPORT_MODE_OPTIONS)[0] as ExportMode;
	}
	if (fields.includes('Matching')) {
		action.matching = { in: '', out: '' };
	}
	if (fields.includes('UpdateOnDuplicates')) {
		action.updateOnDuplicates = false;
	}
	if (fields.includes('Incremental')) {
		action.incremental = false;
	}
	return action;
};

const hasFilters = (connection: TransformedConnection, target: ActionTarget) => {
	// Filters are always allowed except for actions that import users
	// from databases.
	return !(connection.role === 'Source' && connection.connector.type === 'Database' && target === 'User');
};

const computeActionTypeFields = (connection: TransformedConnection, actionType: ActionType) => {
	const fields: ActionTypeField[] = [];

	if (hasFilters(connection, actionType.target)) {
		fields.push('Filter');
	}

	const type = connection.connector.type;

	if (type === 'App') {
		fields.push('Transformation');
	} else if (type === 'Database') {
		fields.push('Transformation');
	} else if (type === 'FileStorage' && connection.role === 'Source') {
		fields.push('Transformation');
	} else if (type === 'SDK') {
		if (connection.role === 'Source' && (actionType.target === 'User' || actionType.target === 'Group')) {
			fields.push('Transformation');
		}
	}

	if (
		type === 'App' &&
		connection.role === 'Destination' &&
		(actionType.target === 'User' || actionType.target === 'Group')
	) {
		fields.push('Matching');
		fields.push('UpdateOnDuplicates');
		fields.push('ExportMode');
	}

	if (type === 'Database' && connection.role === 'Source') {
		fields.push('Query');
	}

	if (type === 'FileStorage') {
		if (connection.role === 'Destination') {
			if (actionType.target === 'User') {
				fields.push('OrderBy');
			}
		}
		fields.push('File');
	}

	if (type === 'Database' && connection.role === 'Destination') {
		fields.push('TableName');
	}

	if (
		(type === 'App' || type === 'Database' || type === 'FileStorage') &&
		connection.role === 'Source' &&
		(actionType.target === 'User' || actionType.target === 'Group')
	) {
		fields.push('Incremental');
	}

	return fields;
};

const addPropertyToSchema = (
	path: string,
	property: Property,
	schema: ObjectType,
	fullSchema: TransformedMapping,
	isFirstLevel: boolean,
) => {
	if (isFirstLevel) {
		const isAlreadyInSchema = schema.properties!.find((p) => p.name === property!.name);
		if (!isAlreadyInSchema) {
			// Push the property in the schema.
			schema.properties!.push(property);
		}
	} else {
		const flat = flattenSchema(schema);
		const isAlreadyInSchema = flat[path] != null;

		if (!isAlreadyInSchema) {
			// Push the property's hierachy in the schema.
			const { ancestors } = getHierarchicalPaths(path, fullSchema);

			let insertedAncestors: string[] = [];
			let closestInsertedAncestor: string | null;
			let missingAncestors: string[] = [];
			for (const a of [...ancestors].reverse()) {
				const p = flat[a];
				const isClosestAlreadyFound = closestInsertedAncestor != null;
				if (p != null && !isClosestAlreadyFound) {
					closestInsertedAncestor = a;
				} else {
					if (isClosestAlreadyFound) {
						insertedAncestors.unshift(a);
					} else {
						missingAncestors.unshift(a);
					}
				}
			}

			if (closestInsertedAncestor != null) {
				const missingHierarchy = buildHierarchy([...missingAncestors, path], fullSchema);

				let hierarchy: Property = missingHierarchy;
				for (const a of [...insertedAncestors, closestInsertedAncestor].reverse()) {
					const p = flat[a].full;
					const typ = p.type as ObjectType;
					if (a === closestInsertedAncestor) {
						// Push the hierarchy of the missing ancestors
						// inside the closest inserted ancestor.
						typ.properties.push(hierarchy);
					} else {
						// Replace the property with the updated one
						// containing the updated hierarchy.
						const i = typ.properties.findIndex((p) => p.name === hierarchy.name);
						typ.properties.splice(i, 1, hierarchy);
					}
					hierarchy = p;
				}

				// Replace the first level property.
				const i = schema.properties.findIndex((p) => p.name === hierarchy.name);
				schema.properties.splice(i, 1, hierarchy);
			} else {
				// Push the entire hierarchy.
				const hierarchy = buildHierarchy([...ancestors, path], fullSchema);
				schema.properties.push(hierarchy);
			}
		}
	}
};

interface hierarchicalPaths {
	ancestors: string[];
	descendants: string[];
}

// getHierarchicalPaths returns the ancestors and descendants paths of
// the property with the given path.
const getHierarchicalPaths = (path: string, mapping: TransformedMapping): hierarchicalPaths => {
	const indentation = mapping[path].indentation;
	const ancestors: string[] = [];
	const descendants: string[] = [];
	for (const p in mapping) {
		if (mapping[p].indentation! < indentation! && path.startsWith(p)) {
			ancestors.push(p);
			continue;
		}
		if (mapping[p].indentation! > indentation! && p.startsWith(path)) {
			descendants.push(p);
			continue;
		}
	}
	return {
		ancestors,
		descendants,
	};
};

// getSiblingPaths returns the sibling paths of the property with
// the given path.
const getSiblingPaths = (path: string, mapping: TransformedMapping): string[] => {
	const { root, indentation } = mapping[path];
	const siblings: string[] = [];
	for (const p in mapping) {
		if (p !== path && mapping[p].root === root && mapping[p].indentation === indentation) {
			siblings.push(p);
		}
	}
	return siblings;
};

const buildHierarchy = (paths: string[], flatSchema: TransformedMapping): Property => {
	let hierarchy: Property;
	let i = 0;
	for (const p of [...paths].reverse()) {
		const full = flatSchema[p].full;
		if (full.type.kind === 'object') {
			if (i !== 0) {
				// empty the properties.
				const typ = full.type as ObjectType;
				typ.properties = [hierarchy];
				full.type = typ;
			}
		}
		hierarchy = full;
		i++;
	}
	return hierarchy;
};

const doesLastChangeTimeColumnNeedFormat = (lastChangeTimeColumn: string, schema: ObjectType): boolean => {
	if (lastChangeTimeColumn == null || lastChangeTimeColumn === '') {
		return false;
	}
	const flatInputSchema = flattenSchema(schema);
	const p = flatInputSchema[lastChangeTimeColumn];
	if (p == null) {
		return false;
	}
	const type = p.type;
	return type === 'json' || type === 'text';
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
	if (isSourceEventConnection(connection.role, connection.connector.type) && actionType.target === 'User') {
		return 'event';
	}
	if (actionType.target === 'User') {
		return 'user';
	} else if (actionType.target === 'Event') {
		return 'event';
	} else if (actionType.target === 'Group') {
		return 'group';
	}
};

const validateFilterConditionValues = (type: Type, values: string[] | null, propertyName: string) => {
	const throwIfInvalid = (isValid: boolean, typeKind: string) => {
		if (!isValid) {
			throw new Error(`The filter value on the property "${propertyName}" is not a valid ${typeKind}`);
		}
	};

	if (values == null) {
		return;
	}

	for (const v of values) {
		if (type.kind === 'int') {
			throwIfInvalid(isInt(v), type.kind);
		} else if (type.kind === 'uint') {
			throwIfInvalid(isUint(v), type.kind);
		} else if (type.kind === 'float') {
			throwIfInvalid(isFloat(v, type.bitSize), type.kind);
		} else if (type.kind === 'decimal') {
			throwIfInvalid(isDecimal(v), type.kind);
		} else if (type.kind === 'datetime') {
			throwIfInvalid(isDateTime(v), type.kind);
		} else if (type.kind === 'date') {
			throwIfInvalid(isDate(v), type.kind);
		} else if (type.kind === 'year') {
			throwIfInvalid(isYear(v), type.kind);
		} else if (type.kind === 'uuid') {
			throwIfInvalid(isUUID(v), type.kind);
		} else if (type.kind === 'inet') {
			throwIfInvalid(isInet(v), type.kind);
		} else if (type.kind === 'array') {
			if (type.elementType.kind !== 'json' && type.elementType.kind !== 'text') {
				validateFilterConditionValues(type.elementType, [v], propertyName);
			}
		}
	}
};

const validateMatching = (inMatching: Property, outMatching: Property) => {
	const inTyp = inMatching.type.kind;
	if (inTyp !== 'int' && inTyp !== 'uint' && inTyp !== 'text' && inTyp !== 'uuid') {
		throw new Error(`Matching property cannot be of type "${inTyp}"`);
	}

	// Check that the in property can be converted to the type of the
	// out property.
	const exTyp = outMatching.type.kind;
	const conversionError = new Error(`Matching property of type "${inTyp}" cannot be converted to type "${exTyp}"`);

	if (inTyp === 'int') {
		if (exTyp !== 'int' && exTyp !== 'uint' && exTyp !== 'text') {
			throw conversionError;
		}
	} else if (inTyp === 'uint') {
		if (exTyp !== 'int' && exTyp !== 'uint' && exTyp !== 'text') {
			throw conversionError;
		}
	} else if (inTyp === 'text') {
		if (exTyp !== 'int' && exTyp !== 'uint' && exTyp !== 'uuid' && exTyp !== 'text') {
			throw conversionError;
		}
	} else if (inTyp === 'uuid') {
		if (exTyp !== 'uuid' && exTyp !== 'text') {
			throw conversionError;
		}
	}
};

const propertyTypesAreEqual = (aType: Type, bType: Type): boolean => {
	if (aType.kind !== bType.kind) {
		return false;
	}

	if (aType.kind === 'int' || aType.kind === 'uint') {
		const t = bType as IntType | UintType;
		return aType.bitSize === t.bitSize && aType.minimum === t.minimum && aType.maximum === t.maximum;
	} else if (aType.kind === 'text') {
		const t = bType as TextType;
		return (
			aType.byteLen === t.byteLen &&
			aType.charLen === t.charLen &&
			aType.regexp === t.regexp &&
			JSON.stringify(aType.values) === JSON.stringify(t.values)
		);
	}

	return true;
};

export {
	SCHEDULE_PERIODS,
	FILTER_OPERATORS,
	EXPORT_MODE_OPTIONS,
	flattenSchema,
	isRecursiveType,
	computeDefaultAction,
	hasFilters,
	computeActionTypeFields,
	transformActionType,
	transformAction,
	transformInActionToSet,
	getCompatibleFilterOperators,
	isUnaryOperator,
	isBetweenOperator,
	isOneOfOperator,
	splitPropertyAndPath,
	getHierarchicalPaths,
	getSiblingPaths,
	doesLastChangeTimeColumnNeedFormat,
	getTransformationFunctionParameterName,
	validateMatching,
	propertyTypesAreEqual,
};

export type { TransformedMapping, TransformedProperty, TransformedActionType, TransformedAction, ActionTypeField };
