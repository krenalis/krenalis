import {
	Pipeline,
	PipelineTarget,
	PipelineToSet,
	PipelineType,
	ExportMode,
	ExpressionToBeExtracted,
	Filter,
	FilterCondition,
	FilterOperator,
	Mapping,
	Matching,
	SchedulePeriod,
	TransformationFunction,
} from '../api/types/pipeline';
import { ConnectorSettings } from '../api/types/responses';
import { Compression } from '../api/types/connection';
import Type, {
	ArrayType,
	FloatType,
	IntType,
	MapType,
	ObjectType,
	Property,
	Role,
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
	'is empty',
	'is not empty',
	'is null',
	'is not null',
	'exists',
	'does not exist',
];

const UNARY_OPERATORS = new Set<FilterOperator>([
	'is true',
	'is false',
	'is empty',
	'is not empty',
	'is null',
	'is not null',
	'exists',
	'does not exist',
]);

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
	['json', 'text', 'object', 'array', 'map'], // is empty
	['json', 'text', 'object', 'array', 'map'], // is not empty
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
	], // exists
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
	], // does not exist
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

type PipelineTypeField =
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

interface TransformedPipelineType {
	name: string;
	description: string;
	target: PipelineTarget;
	eventType: string;
	inputSchema: ObjectType;
	outputSchema: ObjectType;
	inputMatchingSchema: ObjectType | null;
	outputMatchingSchema: ObjectType | null;
	fields: PipelineTypeField[];
}

interface TransformedEventType {
	id: string;
	name: string;
	description: string;
	filter: Filter | null;
}

interface TransformedPipeline {
	id?: number;
	connection?: number;
	target: PipelineTarget;
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

const getCompatibleFilterOperators = (
	property: TransformedProperty,
	hasPath: boolean,
	role: Role,
	target: PipelineTarget,
): number[] => {
	if (property == null) {
		return [];
	}
	const operators: number[] = [];
	for (const i of Object.keys(FILTER_OPERATORS)) {
		let op = FILTER_OPERATORS[i];

		// 'is null' and 'is not null' are compatible only with nullable
		// properties or json type properties.
		if (op === 'is null' || op === 'is not null') {
			const isNullable = property.full.nullable;
			const isJSON = property.type === 'json';
			if (!isNullable && !isJSON) {
				continue;
			}
		}

		// 'contains' and 'does not contain' should only be shown if the type of
		// the array element is supported by the 'is' operator.
		if ((op === 'contains' || op === 'does not contain') && property.type === 'array') {
			const elementType = (property.full.type as ArrayType).elementType;
			const isOperatorIndex = FILTER_OPERATORS.findIndex((op) => op === 'is');
			if (!typesByFilterOperator[isOperatorIndex].includes(elementType.kind)) {
				continue;
			}
		}

		// text property with values can only be used with the operators
		// 'is', 'is not', 'is one of', and 'is not one of'.
		if (property.type === 'text' && (property.full.type as TextType).values != null) {
			switch (op) {
				case 'is':
				case 'is not':
				case 'is one of':
				case 'is not one of':
					break;
				default:
					continue;
			}
		}

		// 'is empty' and 'is not empty' cannot be used on object properties for users with role 'Destination' or for events.
		if ((op === 'is empty' || op === 'is not empty') && property.type === 'object') {
			const disallowEmptyOnObject = (role === 'Destination' && target == 'User') || target == 'Event';
			if (disallowEmptyOnObject) {
				continue;
			}
		}

		if (op === 'exists' || op === 'does not exist') {
			if (!(property.readOptional || (property.type === 'json' && hasPath))) {
				continue;
			}
		}

		if (typesByFilterOperator[i].includes(property.type)) {
			operators.push(Number(i));
		}
	}
	return operators;
};

const isUnaryOperator = (op: FilterOperator | ''): boolean => (op === '' ? false : UNARY_OPERATORS.has(op));

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
		if (flatSchema?.[b] != null) {
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

	const property = flatSchema?.[base];
	const isJSON = property?.type === 'json';
	if (!isJSON && path !== '') {
		// handle cases where the user has typed an invalid subproperty and for
		// this reason the subproperty name was incorrectly considered as a
		// path.
		return ['', ''];
	}

	return [base, path];
};

const hasValidTransformation = (pipeline: PipelineToSet) => {
	return pipeline.transformation?.function != null || pipeline.transformation?.mapping != null;
};

const errInvalidTransformation = new Error('Pipeline must have a valid transformation');
const errTransformationNotSupported = new Error('Pipeline does not support transformations');

const validateTransformation = (
	connection: TransformedConnection,
	pipelineType: TransformedPipelineType,
	pipeline: PipelineToSet,
) => {
	if (connection.isSource) {
		if (connection.isAPI) {
			if (pipelineType.target === 'User' || pipelineType.target === 'Group') {
				if (!hasValidTransformation(pipeline)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isDatabase) {
			if (pipelineType.target === 'User' || pipelineType.target === 'Group') {
				if (!hasValidTransformation(pipeline)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isFileStorage) {
			if (pipelineType.target === 'User' || pipelineType.target === 'Group') {
				if (!hasValidTransformation(pipeline)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isSDK || connection.isWebhook) {
			if (pipelineType.target === 'Event') {
				if (hasValidTransformation(pipeline)) {
					throw errTransformationNotSupported;
				}
			}
		}
	} else {
		if (connection.isAPI) {
			if (pipelineType.target === 'User' || pipelineType.target === 'Group') {
				if (!hasValidTransformation(pipeline)) {
					throw errInvalidTransformation;
				}
			}
			if (pipelineType.target === 'Event' && pipelineType.outputSchema == null) {
				if (hasValidTransformation(pipeline)) {
					throw errTransformationNotSupported;
				}
			}
		} else if (connection.isDatabase) {
			if (pipelineType.target === 'User' || pipelineType.target === 'Group') {
				if (!hasValidTransformation(pipeline)) {
					throw errInvalidTransformation;
				}
			}
		} else if (connection.isFileStorage) {
			if (pipelineType.target === 'User' || pipelineType.target === 'Group') {
				if (hasValidTransformation(pipeline)) {
					throw errTransformationNotSupported;
				}
			}
		}
	}
};

// TODO: do not set the value and the required values here (this should only
// return the flattened schema). Add a new 'getDefaultMapping' function that
// takes the flatten schema and add values, and disableds. In
// 'transformPipelineMapping' set the values or set ''. This should only return
// the list of flattened keys mapping to the full property object.
const flattenSchema = (typ: ObjectType | ArrayType | MapType, insertPrefilled?: boolean): TransformedMapping | null => {
	if (typ == null || !isRecursiveType(typ)) {
		return null;
	}

	const flattenProperty = (property: Property): TransformedProperty => {
		let val = '';
		if (insertPrefilled != null && insertPrefilled && property.prefilled) {
			val = property.prefilled;
		}
		const flat = {
			value: val,
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

const transformPipelineType = (
	pipelineType: PipelineType,
	fields: PipelineTypeField[],
	inputSchema: ObjectType,
	outputSchema: ObjectType,
	inputMatchingSchema: ObjectType,
	outputMatchingSchema: ObjectType,
): TransformedPipelineType => {
	return {
		name: pipelineType.name,
		description: pipelineType.description,
		target: pipelineType.target,
		eventType: pipelineType.eventType,
		inputSchema: inputSchema,
		outputSchema: outputSchema,
		inputMatchingSchema: inputMatchingSchema,
		outputMatchingSchema: outputMatchingSchema,
		fields: fields,
	};
};

const transformPipelineMapping = (mapping: Mapping, outputSchema: ObjectType): TransformedMapping => {
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

const transformPipeline = (pipeline: Pipeline, outputSchema: ObjectType): TransformedPipeline => {
	let pipelineMapping = pipeline.transformation.mapping;
	if (pipeline.transformation.function == null && pipelineMapping == null) {
		// Mappings are selected but there is nothing mapped.
		pipelineMapping = {};
	}

	if (
		pipeline.lastChangeTimeFormat != null &&
		pipeline.lastChangeTimeFormat != '' &&
		pipeline.lastChangeTimeFormat.startsWith("'") &&
		pipeline.lastChangeTimeFormat.endsWith("'")
	) {
		pipeline.lastChangeTimeFormat = pipeline.lastChangeTimeFormat.substring(
			1,
			pipeline.lastChangeTimeFormat.length - 1,
		);
	}

	if (pipeline.filter) {
		const conditions: FilterCondition[] = [];
		for (const condition of pipeline.filter.conditions) {
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
		pipeline.filter.conditions = conditions;
	}

	return {
		id: pipeline.id,
		connection: pipeline.connection,
		target: pipeline.target,
		name: pipeline.name,
		enabled: pipeline.enabled,
		eventType: pipeline.eventType,
		running: pipeline.running,
		scheduleStart: pipeline.scheduleStart,
		schedulePeriod: pipeline.schedulePeriod,
		inSchema: pipeline.inSchema,
		outSchema: pipeline.outSchema,
		filter: pipeline.filter,
		transformation: {
			mapping: pipelineMapping != null ? transformPipelineMapping(pipelineMapping, outputSchema) : null,
			function: pipeline.transformation.function,
		},
		query: pipeline.query,
		path: pipeline.path,
		tableName: pipeline.tableName,
		tableKey: pipeline.tableKey,
		sheet: pipeline.sheet,
		identityColumn: pipeline.identityColumn,
		lastChangeTimeColumn: pipeline.lastChangeTimeColumn,
		lastChangeTimeFormat: pipeline.lastChangeTimeFormat,
		incremental: pipeline.incremental,
		exportMode: pipeline.exportMode,
		matching: pipeline.matching,
		updateOnDuplicates: pipeline.updateOnDuplicates,
		format: pipeline.format,
		compression: pipeline.compression,
		orderBy: pipeline.orderBy,
	};
};

const transformInPipelineToSet = async (
	pipeline: TransformedPipeline,
	formatSettings: ConnectorSettings,
	pipelineType: TransformedPipelineType,
	api: API,
	connection: TransformedConnection,
	trimFunction: boolean,
	selectedInPaths: string[],
	selectedOutPaths: string[],
): Promise<PipelineToSet> => {
	let mapping: Mapping;
	let inSchema: ObjectType;
	let outSchema: ObjectType;
	let func: TransformationFunction;
	let query: string;

	const isDestinationFileOnUsers =
		connection.isDestination && connection.isFileStorage && pipelineType.target === 'User';

	const flattenedInputSchema = flattenSchema(pipelineType.inputSchema);
	const flattenedOutputSchema = flattenSchema(pipelineType.outputSchema);

	// Remove the prefilled values from the output schema.
	for (const p in flattenedOutputSchema) {
		if (flattenedOutputSchema[p].full.prefilled) {
			delete flattenedOutputSchema[p].full.prefilled;
		}
	}

	const allowsConstantTransformation =
		(connection.isSource && connection.isEventBased && pipelineType.target === 'User') ||
		(connection.isDestination && connection.isAPI && pipelineType.target === 'Event');

	if (pipeline.transformation.mapping != null) {
		const inputSchema: ObjectType = { kind: 'object', properties: [] };
		const outputSchema: ObjectType = { kind: 'object', properties: [] };
		const mappingToSave = {};
		const expressions: ExpressionToBeExtracted[] = [];

		const keys = Object.keys(pipeline.transformation.mapping);
		for (const k of keys) {
			// The property must be mapped if it is required and it is a
			// first-level property, or if one of its siblings has been
			// mapped.
			const property = pipeline.transformation.mapping[k];
			const isFirstLevel = property.indentation === 0;
			if (property.value === '') {
				const hasRequired =
					(pipelineType.target === 'Event' && (property.createRequired || property.updateRequired)) ||
					(pipeline.exportMode != null &&
						((property.createRequired && pipeline.exportMode.includes('Create')) ||
							(property.updateRequired && pipeline.exportMode.includes('Update'))));

				const siblings: string[] = [];
				for (const key of keys) {
					const p = pipeline.transformation.mapping[key];
					if (p.root === property.root && p.indentation === property.indentation && key !== k) {
						siblings.push(key);
					}
				}
				const hasMappedSiblings =
					siblings.findIndex((k) => pipeline.transformation.mapping[k].value !== '') !== -1;

				const isRequired = hasRequired && (isFirstLevel || hasMappedSiblings);
				const isInMatching = pipeline.matching != null && pipeline.matching.out === k; // Check if is used in the matching properties.
				if (isRequired && !isInMatching) {
					throw new Error(`Property "${k}" is required. Indicate an expression for this property.`);
				}
				continue;
			}

			// Check if there are UI errors in the mapping.
			if (property.error && property.error !== '') {
				throw new Error(`Please fix the errors in the mapping`);
			}

			const mapped = flattenedOutputSchema![k].full;
			expressions.push({
				value: property.value,
				type: mapped!.type,
			});

			mappingToSave[k] = property.value;

			addPropertyToSchema(k, mapped, outputSchema, flattenedOutputSchema, isFirstLevel);
		}

		let inputPaths: string[];
		try {
			inputPaths = await api.expressionsProperties(expressions, pipelineType.inputSchema);
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

		inSchema = sortPropertiesByOriginalSchema(inputSchema, pipelineType.inputSchema);
		outSchema = sortPropertiesByOriginalSchema(outputSchema, pipelineType.outputSchema);
	} else if (pipeline.transformation.function != null) {
		const inputSchema: ObjectType = { kind: 'object', properties: [] };
		const outputSchema: ObjectType = { kind: 'object', properties: [] };

		const inPaths: string[] = [];
		for (const p of selectedInPaths) {
			// Add the property to the input schema of the pipeline.
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

		const outPaths: string[] = [];
		for (const p of selectedOutPaths) {
			// Add the property to the output schema of the pipeline.
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

		if ((inPaths.length === 0 && !allowsConstantTransformation) || outPaths.length === 0) {
			throw new Error(
				'After selecting a transformation function, select at least one input and one output property in Full mode before saving',
			);
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
				pipeline.exportMode != null &&
				((p.createRequired && pipeline.exportMode.includes('Create')) ||
					(p.updateRequired && pipeline.exportMode.includes('Update')));

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
			const isInMatching = pipeline.matching != null && pipeline.matching.out === k;
			if (isRequired && !isSelected && !isParentSelected && !isInMatching) {
				throw new Error(`Property "${k}" is required and you must pass it in the transformation function`);
			}
			continue;
		}

		let source = pipeline.transformation.function.source;
		if (trimFunction) {
			source = source.trim();
		}

		func = {
			source: source,
			language: pipeline.transformation.function.language,
			preserveJSON: pipeline.transformation.function.preserveJSON,
			inPaths: inPaths,
			outPaths: outPaths,
		};

		inSchema = sortPropertiesByOriginalSchema(inputSchema, pipelineType.inputSchema);
		outSchema = sortPropertiesByOriginalSchema(outputSchema, pipelineType.outputSchema);
	} else if (isDestinationFileOnUsers) {
		inSchema = pipelineType.inputSchema;
		outSchema = null; // TODO(Gianluca): it this necessary?
	}

	if (pipeline.matching != null) {
		const inMatching = pipeline.matching.in;
		const outMatching = pipeline.matching.out;
		if (inMatching === '' || outMatching === '') {
			throw new Error('Matching properties cannot be empty');
		}

		const flattenedInputMatchingSchema = flattenSchema(pipelineType.inputMatchingSchema);
		const flattenedOutputMatchingSchema = flattenSchema(pipelineType.outputMatchingSchema);

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

		// Add the in matching property to the input schema of the pipeline, if it
		// isn't already inserted.
		const isAlreadyInSchema = flattenSchema(inSchema)[inMatching] != null;
		if (!isAlreadyInSchema) {
			addPropertyToSchema(
				inMatching,
				inMatchingProperty.full,
				inSchema,
				flattenedInputSchema,
				inMatchingProperty.indentation === 0,
			);
		}

		// Add the out matching property to the output schema of the pipeline.
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
				if (pipeline.exportMode === 'CreateOnly' || pipeline.exportMode === 'CreateOrUpdate') {
					throw new Error(
						`Since "${pipeline.matching.out}" is set as the ${connection.connector.label}'s matching property and it is read-only, users can only be updated, not created. Change the matching property accordingly or select 'Update only'.`,
					);
				} else {
					p = a;
				}
			}
			const isAlreadyInSchema = flattenSchema(outSchema)[outMatching] != null;
			if (isAlreadyInSchema) {
				throw new Error(`External matching property cannot be used in the transformation`);
			}
			addPropertyToSchema(
				outMatching,
				p,
				outSchema,
				flattenedOutputSchema,
				outMatchingProperty.indentation === 0,
			);
		}
	}

	if (connection.isSource && (connection.isDatabase || connection.isFileStorage)) {
		if (pipeline.identityColumn == null || pipeline.identityColumn === '') {
			throw new Error('User identifier cannot be empty');
		}
		const isAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === pipeline.identityColumn) !== -1;
		if (!isAlreadyInSchema) {
			const identityColumn = flattenedInputSchema[pipeline.identityColumn];
			if (identityColumn == null) {
				throw new Error('User identifier must be a valid property');
			}
			inSchema.properties.push(identityColumn.full);
		}

		if (pipeline.lastChangeTimeColumn) {
			const isAlreadyInSchema =
				inSchema.properties!.findIndex((p) => p.name === pipeline.lastChangeTimeColumn) !== -1;
			if (!isAlreadyInSchema) {
				const lastChangeTimeColumn = flattenedInputSchema[pipeline.lastChangeTimeColumn];
				if (lastChangeTimeColumn == null) {
					throw new Error('Last change time must be a valid column');
				}
				inSchema.properties.push(lastChangeTimeColumn.full);
			}
			if (doesLastChangeTimeColumnNeedFormat(pipeline.lastChangeTimeColumn, pipelineType.inputSchema)) {
				if (pipeline.lastChangeTimeFormat !== 'ISO8601' && pipeline.lastChangeTimeFormat !== 'Excel') {
					// the format is custom.
					try {
						validateCustomLastChangeTimeFormat(pipeline.lastChangeTimeFormat);
					} catch (err) {
						throw err;
					}
				}
			}
		}
	}

	let filter: Filter = null;
	if (pipeline.filter != null) {
		if (inSchema == null) {
			inSchema = { kind: 'object', properties: [] };
		}

		let f = { logical: pipeline.filter.logical, conditions: [] };

		// Exclude conditions that have empty properties.
		let conditions = pipeline.filter.conditions.filter((condition) => condition.property !== '');

		const isEventImport = connection.isSource && pipelineType.target === 'Event';
		const isEventBasedUserImport = connection.isEventBased && connection.isSource && pipelineType.target === 'User';
		const isAppEventsExport = connection.isAPI && connection.isDestination && pipelineType.target === 'Event';

		for (const condition of conditions) {
			const propertyName = condition.property;
			const [base, path] = splitPropertyAndPath(propertyName, flattenedInputSchema);
			const property = flattenedInputSchema[base];
			let c: FilterCondition;
			try {
				c = validateAndNormalizeFilterCondition(
					condition,
					property,
					path,
					propertyName,
					isEventBasedUserImport || isAppEventsExport || isEventImport ? ['mpid'] : null,
				);
			} catch (err) {
				throw err;
			}
			addPropertyToSchema(base, property.full, inSchema, flattenedInputSchema, property.indentation === 0);
			f.conditions.push(c);
		}

		if (f.conditions.length > 0) {
			filter = f;
		}
	}

	if (pipeline.query != null) {
		query = pipeline.query.trim();
	}

	if (pipeline.sheet != null) {
		const s = pipeline.sheet;
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

	if (pipeline.orderBy != null) {
		const p = pipeline.orderBy;
		if (p === '') {
			throw new Error('File ordering property cannot be empty');
		}
		const filteredSchema = filterOrderingPropertySchema(pipelineType.inputSchema);
		if (filteredSchema != null) {
			if (filteredSchema[p] == null) {
				throw new Error(`File ordering property "${p}" does not exist in schema`);
			}
		}
	}

	const isDatabaseExportOnUsers =
		connection.connector.type === 'Database' && connection.role === 'Destination' && pipelineType.target === 'User';

	if (isDatabaseExportOnUsers) {
		// the table key must be defined for database type pipelines that
		// export users.
		if (pipeline.tableKey == null || pipeline.tableKey === '') {
			throw new Error('Table key cannot be empty');
		}

		// the table key must be a valid property.
		const property = flattenedOutputSchema[pipeline.tableKey];
		if (property == null) {
			throw new Error('Table key must be a valid property');
		}

		// the table key must necessarily be transformed and there must
		// be another transformed property in addition to it.
		if (mapping == null && func == null) {
			throw new Error('Table key must be transformed');
		} else if (mapping != null) {
			const mappedPaths = Object.keys(mapping);
			if (!mappedPaths.includes(pipeline.tableKey)) {
				throw new Error('Table key must be transformed');
			}
			if (mappedPaths.length === 1) {
				throw new Error('Another property must be transformed in addition to the table key property');
			}
		} else if (func != null) {
			if (!func.outPaths.includes(pipeline.tableKey)) {
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
			if (p.name === pipeline.tableKey) {
				p.createRequired = true;
				p.updateRequired = false;
				p.nullable = false;
				break;
			}
		}
	} else {
		// the table key must be empty for pipelines that are not
		// database type pipelines that export users.
		if (pipeline.tableKey != null && pipeline.tableKey !== '') {
			throw new Error('Table key must be empty for this kind of pipeline');
		}
	}

	// In cases where the input schema refers to events, that is when:
	//
	//  - identities are imported from events
	//  - events are imported into the data warehouse
	//  - events are dispatched to apps
	//
	// the input schema must be nil, which means the schema of the events.
	let importEventsIntoWarehouse = connection.isSource && connection.isEventBased && pipelineType.target == 'Event';
	let dispatchEventsToApps =
		connection.isDestination && connection.connector.type == 'API' && pipelineType.target == 'Event';
	let importIdentitiesFromEvents = connection.isSource && connection.isEventBased && pipelineType.target == 'User';
	if (importIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps) {
		inSchema = null;
	}

	let incremental = pipeline.incremental;
	if (connection.isSource && (connection.isDatabase || connection.isFileStorage)) {
		// If last change time is not set the import cannot be
		// incremental.
		if (pipeline.lastChangeTimeColumn === '') {
			incremental = false;
		}
	}

	const pipelineToSet: PipelineToSet = {
		name: pipeline.name,
		enabled: pipeline.enabled,
		filter: filter,
		inSchema: inSchema && inSchema.properties.length > 0 ? inSchema : null,
		outSchema: outSchema && outSchema.properties.length > 0 ? outSchema : null,
		transformation: {
			mapping: mapping,
			function: func,
		},
		query: query!,
		path: pipeline.path,
		tableName: pipeline.tableName,
		tableKey: pipeline.tableKey,
		sheet: pipeline.sheet,
		identityColumn: pipeline.identityColumn,
		lastChangeTimeColumn: pipeline.lastChangeTimeColumn,
		lastChangeTimeFormat: pipeline.lastChangeTimeFormat,
		incremental: incremental,
		compression: pipeline.compression,
		orderBy: pipeline.orderBy,
		format: pipeline.format,
		formatSettings: formatSettings,
	};

	if (pipeline.matching != null) {
		pipelineToSet.matching = pipeline.matching;
	}

	if (pipeline.updateOnDuplicates != null) {
		let updateOnDuplicates = pipeline.updateOnDuplicates;
		if (!pipeline.exportMode.includes('Update')) {
			// If export mode is "CreateOnly", `updateOnDuplicates` is
			// not taken into consideration.
			updateOnDuplicates = false;
		}
		pipelineToSet.updateOnDuplicates = updateOnDuplicates;
	}

	if (pipeline.exportMode != null) {
		pipelineToSet.exportMode = pipeline.exportMode;
	}

	try {
		validateTransformation(connection, pipelineType, pipelineToSet);
	} catch (err) {
		throw err;
	}

	return pipelineToSet;
};

const computeDefaultPipeline = (
	pipelineType: PipelineType | TransformedPipelineType,
	connection: TransformedConnection,
	outputSchema: ObjectType,
	fields: PipelineTypeField[],
): TransformedPipeline => {
	const pipeline: TransformedPipeline = {
		target: pipelineType.target,
		name: pipelineType.name,
		// The pipeline is enabled by default only for batch operations importing or exporting users.
		enabled:
			pipelineType.target == 'User' && (connection.isAPI || connection.isDatabase || connection.isFileStorage),
		filter: null,
		transformation: {
			mapping: flattenSchema(outputSchema, true),
			function: null,
		},
		inSchema: null,
		outSchema: null,
	};
	if (fields.includes('Filter')) {
		const eventType = connection.eventTypes.find((t) => t.id === pipelineType.eventType);
		if (eventType != null && eventType.filter != null) {
			pipeline.filter = eventType.filter;
		} else if ((connection.isSDK || connection.isWebhook) && pipelineType.target === 'User') {
			pipeline.filter = {
				logical: 'or',
				conditions: [
					{ property: 'type', operator: 'is', values: ['identify'] },
					{ property: 'traits', operator: 'is not empty', values: null },
				],
			};
		}
	}
	if (fields.includes('Query')) {
		pipeline.query = connection.connector.asSource.sampleQuery;
		pipeline.identityColumn = '';
		pipeline.lastChangeTimeColumn = '';
		pipeline.lastChangeTimeFormat = '';
	}
	if (fields.includes('File')) {
		pipeline.path = '';
		pipeline.identityColumn = '';
		pipeline.lastChangeTimeColumn = '';
		pipeline.lastChangeTimeFormat = '';
		pipeline.sheet = null;
		pipeline.compression = '';
		pipeline.format = '';
	}
	if (fields.includes('OrderBy')) {
		pipeline.orderBy = '';
	}
	if (fields.includes('TableName')) {
		pipeline.tableName = '';
		pipeline.tableKey = '';
	}
	if (fields.includes('ExportMode')) {
		pipeline.exportMode = Object.keys(EXPORT_MODE_OPTIONS)[0] as ExportMode;
	}
	if (fields.includes('Matching')) {
		pipeline.matching = { in: '', out: '' };
	}
	if (fields.includes('UpdateOnDuplicates')) {
		pipeline.updateOnDuplicates = false;
	}
	if (fields.includes('Incremental')) {
		pipeline.incremental = false;
	}
	return pipeline;
};

const hasFilters = (connection: TransformedConnection, target: PipelineTarget) => {
	// Filters are always allowed except for pipelines that import users
	// from databases.
	return !(connection.role === 'Source' && connection.connector.type === 'Database' && target === 'User');
};

const computePipelineTypeFields = (connection: TransformedConnection, pipelineType: PipelineType) => {
	const fields: PipelineTypeField[] = [];

	if (hasFilters(connection, pipelineType.target)) {
		fields.push('Filter');
	}

	const type = connection.connector.type;

	if (type === 'API') {
		fields.push('Transformation');
	} else if (type === 'Database') {
		fields.push('Transformation');
	} else if (type === 'FileStorage' && connection.role === 'Source') {
		fields.push('Transformation');
	} else if (type === 'SDK' || type == 'Webhook') {
		if (connection.role === 'Source' && (pipelineType.target === 'User' || pipelineType.target === 'Group')) {
			fields.push('Transformation');
		}
	}

	if (
		type === 'API' &&
		connection.role === 'Destination' &&
		(pipelineType.target === 'User' || pipelineType.target === 'Group')
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
			if (pipelineType.target === 'User') {
				fields.push('OrderBy');
			}
		}
		fields.push('File');
	}

	if (type === 'Database' && connection.role === 'Destination') {
		fields.push('TableName');
	}

	if (
		(type === 'API' || type === 'Database' || type === 'FileStorage') &&
		connection.role === 'Source' &&
		(pipelineType.target === 'User' || pipelineType.target === 'Group')
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
				const missingHierarchy = buildHierarchy([...missingAncestors, path], fullSchema, property);

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
				const hierarchy = buildHierarchy([...ancestors, path], fullSchema, property);
				schema.properties.push(hierarchy);
			}
		}
	}
};

function sortPropertiesByOriginalSchema(schema: ObjectType, original: ObjectType): ObjectType {
	const originalIndexMap = new Map(original.properties.map((prop, index) => [prop.name, index]));

	const sortedProps = [...schema.properties].sort(
		(a, b) => originalIndexMap.get(a.name)! - originalIndexMap.get(b.name)!,
	);

	const properties = sortedProps.map((p) => {
		if (p.type.kind === 'object' && Array.isArray(p.type.properties)) {
			// Sort the nested properties too.
			const prop = original.properties.find((op) => op.name === p.name);
			return {
				...p,
				type: sortPropertiesByOriginalSchema(p.type, prop.type as ObjectType),
			};
		}
		return p;
	});

	return {
		kind: 'object',
		properties: properties,
	};
}

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

const buildHierarchy = (paths: string[], flatSchema: TransformedMapping, property: Property): Property => {
	let hierarchy: Property;
	let i = 0;
	for (const p of [...paths].reverse()) {
		let full = flatSchema[p].full;
		if (i === 0) {
			full = property;
		}
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
	pipelineType: TransformedPipelineType,
): String => {
	if (isSourceEventConnection(connection.role, connection.connector.type) && pipelineType.target === 'User') {
		return 'event';
	}
	if (pipelineType.target === 'User') {
		return 'user';
	} else if (pipelineType.target === 'Event') {
		return 'event';
	} else if (pipelineType.target === 'Group') {
		return 'group';
	}
};

// validateAndNormalizeFilterCondition validates and normalizes a filter
// condition. It throws an error if the condition is invalid, otherwise it
// returns the condition with its values parsed.
const validateAndNormalizeFilterCondition = (
	condition: FilterCondition,
	property: TransformedProperty,
	propertyPath: string,
	propertyName: string,
	propertiesToHide?: string[] | null,
): FilterCondition => {
	if (property == null || propertiesToHide?.includes(propertyName)) {
		throw new Error(`Property "${propertyName}" of filter condition does not exist`);
	}

	if (property.type === 'json' && propertyPath.trim() !== '') {
		const isValid = isValidPropertyPath(propertyPath);
		if (!isValid) {
			throw new Error(`Property path "${propertyPath}" of filter condition is not valid`);
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

	const c: FilterCondition = { property: condition.property, operator: condition.operator, values: values };
	return c;
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
	computeDefaultPipeline,
	hasFilters,
	computePipelineTypeFields,
	transformPipelineType,
	transformPipeline,
	transformInPipelineToSet,
	getCompatibleFilterOperators,
	isUnaryOperator,
	isBetweenOperator,
	isOneOfOperator,
	splitPropertyAndPath,
	getHierarchicalPaths,
	getSiblingPaths,
	doesLastChangeTimeColumnNeedFormat,
	getTransformationFunctionParameterName,
	validateAndNormalizeFilterCondition,
	validateMatching,
	propertyTypesAreEqual,
};

export type {
	TransformedMapping,
	TransformedProperty,
	TransformedPipelineType,
	TransformedEventType,
	TransformedPipeline,
	PipelineTypeField,
};
