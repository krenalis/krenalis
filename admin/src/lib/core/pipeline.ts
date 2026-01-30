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
	StringType,
} from '../api/types/types';
import API from '../api/api';
import TransformedConnection, { isSourceEventConnection } from './connection';
import { filterOrderingPropertySchema } from '../../components/helpers/getSchemaComboboxItems';
import {
	formatString,
	isDate,
	isDateTime,
	isDecimal,
	isFloat,
	isIP,
	isInt,
	isUnsigned,
	isUUID,
	isValidPropertyPath,
	isYear,
	parseText,
} from '../../utils/filters';
import { RAW_TRANSFORMATION_FUNCTIONS } from '../../components/routes/PipelineWrapper/Pipeline.constants';

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
	['int', 'float', 'decimal', 'datetime', 'date', 'time', 'year', 'uuid', 'json', 'ip', 'string'], // is
	['int', 'float', 'decimal', 'datetime', 'date', 'time', 'year', 'uuid', 'json', 'ip', 'string'], // is not
	['int', 'float', 'decimal', 'json', 'string'], // is less than
	['int', 'float', 'decimal', 'json', 'string'], // is less than or equal to
	['int', 'float', 'decimal', 'json', 'string'], // is greater than
	['int', 'float', 'decimal', 'json', 'string'], // is greater than or equal to
	['int', 'float', 'decimal', 'year', 'datetime', 'date', 'time', 'json', 'string'], // is between
	['int', 'float', 'decimal', 'year', 'datetime', 'date', 'time', 'json', 'string'], // is not between
	['json', 'string', 'array'], // contains
	['json', 'string', 'array'], // does not contain
	['int', 'float', 'decimal', 'year', 'datetime', 'date', 'time', 'json', 'string'], // is one of
	['int', 'float', 'decimal', 'year', 'datetime', 'date', 'time', 'json', 'string'], // is not one of
	['json', 'string'], // starts with
	['json', 'string'], // ends with
	['datetime', 'date', 'time', 'year'], // is before
	['datetime', 'date', 'time', 'year'], // is on or before
	['datetime', 'date', 'time', 'year'], // is after
	['datetime', 'date', 'time', 'year'], // is on or after
	['boolean', 'json'], // is true
	['boolean', 'json'], // is false
	['json', 'string', 'object', 'array', 'map'], // is empty
	['json', 'string', 'object', 'array', 'map'], // is not empty
	[
		'boolean',
		'int',
		'float',
		'decimal',
		'datetime',
		'date',
		'year',
		'time',
		'uuid',
		'json',
		'ip',
		'string',
		'array',
		'object',
		'map',
	], // is null
	[
		'boolean',
		'int',
		'float',
		'decimal',
		'datetime',
		'date',
		'year',
		'time',
		'uuid',
		'json',
		'ip',
		'string',
		'array',
		'object',
		'map',
	], // is not null
	[
		'boolean',
		'int',
		'float',
		'decimal',
		'datetime',
		'date',
		'year',
		'time',
		'uuid',
		'json',
		'ip',
		'string',
		'array',
		'object',
		'map',
	], // exists
	[
		'boolean',
		'int',
		'float',
		'decimal',
		'datetime',
		'date',
		'year',
		'time',
		'uuid',
		'json',
		'ip',
		'string',
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

type FlatSchema = Record<string, TransformedProperty>;

interface TransformedTransformation {
	mapping: FlatSchema | null;
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
	updatedAtColumn?: string | null;
	updatedAtFormat?: string | null;
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
		if (property.type === 'string' && (property.full.type as StringType).values != null) {
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

const splitPropertyAndPath = (propertyName: string, flatSchema: FlatSchema): [string, string] => {
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
		if (connection.isApplication) {
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
		if (connection.isApplication) {
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
const flattenSchema = (typ: ObjectType | ArrayType | MapType, insertPrefilled?: boolean): FlatSchema | null => {
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
		if (flat.type === 'int' || flat.type === 'float') {
			const prop = property.type as IntType | FloatType;
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

const transformPipelineMapping = (mapping: Mapping, outputSchema: ObjectType): FlatSchema => {
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

const transformPipeline = (
	pipeline: Pipeline,
	outputSchema: ObjectType,
	supportsTransformation: boolean,
): TransformedPipeline => {
	let pipelineMapping = pipeline.transformation?.mapping;
	if (
		supportsTransformation &&
		(pipeline.transformation == null ||
			(pipeline.transformation.mapping == null && pipeline.transformation.function == null))
	) {
		// Mappings are selected but empty because the transformation is
		// optional in this pipeline.
		pipelineMapping = {};
	}

	if (
		pipeline.updatedAtFormat != null &&
		pipeline.updatedAtFormat != '' &&
		pipeline.updatedAtFormat.startsWith("'") &&
		pipeline.updatedAtFormat.endsWith("'")
	) {
		pipeline.updatedAtFormat = pipeline.updatedAtFormat.substring(1, pipeline.updatedAtFormat.length - 1);
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
					const formatted = formatString(v);
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
			function: pipeline.transformation?.function,
		},
		query: pipeline.query,
		path: pipeline.path,
		tableName: pipeline.tableName,
		tableKey: pipeline.tableKey,
		sheet: pipeline.sheet,
		identityColumn: pipeline.identityColumn,
		updatedAtColumn: pipeline.updatedAtColumn,
		updatedAtFormat: pipeline.updatedAtFormat,
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
		(connection.isDestination && connection.isApplication && pipelineType.target === 'Event');

	if (pipeline.transformation.mapping != null) {
		const inputSchema: ObjectType = { kind: 'object', properties: [] };
		const outputSchema: ObjectType = { kind: 'object', properties: [] };
		const mappingToSave = {};
		const expressions: ExpressionToBeExtracted[] = [];
		const paths = Object.keys(pipeline.transformation.mapping);
		for (const p of paths) {
			const property = pipeline.transformation.mapping[p];
			const isFirstLevel = property.indentation === 0;
			if (property.value === '') {
				const { isRequired, isSelected } = checkMapping(p, pipeline, pipelineType);
				if (isRequired && !isSelected) {
					throw new Error(`Property "${p}" is required. Indicate an expression for this property.`);
				}
				continue;
			}

			// Check if there are UI errors in the mapping.
			if (property.error && property.error !== '') {
				throw new Error(`Please fix the errors in the mapping`);
			}

			const mapped = flattenedOutputSchema![p].full;
			expressions.push({
				value: property.value,
				type: mapped!.type,
			});

			mappingToSave[p] = property.value;

			addPropertyToSchema(p, mapped, outputSchema, flattenedOutputSchema, isFirstLevel);
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
			const isParentSelected = selectedInPaths.findIndex((pa) => p.startsWith(`${pa}.`)) !== -1;
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
			const isParentSelected = selectedOutPaths.findIndex((pa) => p.startsWith(`${pa}.`)) !== -1;
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

		const outputSchemaPaths = Object.keys(flattenedOutputSchema);
		for (const p of outputSchemaPaths) {
			const { isRequired, isSelected } = checkFunctionPath(p, pipeline, pipelineType, 'output', outPaths);
			if (isRequired && !isSelected) {
				throw new Error(`Property "${p}" is required and you must pass it in the transformation function`);
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

		if (pipeline.updatedAtColumn) {
			const isAlreadyInSchema = inSchema.properties!.findIndex((p) => p.name === pipeline.updatedAtColumn) !== -1;
			if (!isAlreadyInSchema) {
				const updatedAtColumn = flattenedInputSchema[pipeline.updatedAtColumn];
				if (updatedAtColumn == null) {
					throw new Error('Update time must be a valid column');
				}
				inSchema.properties.push(updatedAtColumn.full);
			}
			if (doesUpdatedAtColumnNeedFormat(pipeline.updatedAtColumn, pipelineType.inputSchema)) {
				if (pipeline.updatedAtFormat !== 'ISO8601' && pipeline.updatedAtFormat !== 'Excel') {
					// the format is custom.
					try {
						validateCustomUpdatedAtFormat(pipeline.updatedAtFormat);
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
		const isAppEventsExport =
			connection.isApplication && connection.isDestination && pipelineType.target === 'Event';

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
		connection.isDestination && connection.connector.type == 'Application' && pipelineType.target == 'Event';
	let importIdentitiesFromEvents = connection.isSource && connection.isEventBased && pipelineType.target == 'User';
	if (importIdentitiesFromEvents || importEventsIntoWarehouse || dispatchEventsToApps) {
		inSchema = null;
	}

	let incremental = pipeline.incremental;
	if (connection.isSource && (connection.isDatabase || connection.isFileStorage)) {
		// If update time is not set the import cannot be
		// incremental.
		if (pipeline.updatedAtColumn === '') {
			incremental = false;
		}
	}

	const pipelineToSet: PipelineToSet = {
		name: pipeline.name,
		enabled: pipeline.enabled,
		filter: filter,
		inSchema: inSchema && inSchema.properties.length > 0 ? inSchema : null,
		outSchema: outSchema && outSchema.properties.length > 0 ? outSchema : null,
		transformation: mapping == null && func == null ? null : { mapping: mapping, function: func },
		query: query!,
		path: pipeline.path,
		tableName: pipeline.tableName,
		tableKey: pipeline.tableKey,
		sheet: pipeline.sheet,
		identityColumn: pipeline.identityColumn,
		updatedAtColumn: pipeline.updatedAtColumn,
		updatedAtFormat: pipeline.updatedAtFormat,
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
			pipelineType.target == 'User' &&
			(connection.isApplication || connection.isDatabase || connection.isFileStorage),
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
		pipeline.updatedAtColumn = '';
		pipeline.updatedAtFormat = '';
	}
	if (fields.includes('File')) {
		pipeline.path = '';
		pipeline.identityColumn = '';
		pipeline.updatedAtColumn = '';
		pipeline.updatedAtFormat = '';
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

	if (type === 'Application') {
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
		type === 'Application' &&
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
		(type === 'Application' || type === 'Database' || type === 'FileStorage') &&
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
	fullSchema: FlatSchema,
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

interface TransformationPropertyInfo {
	isRequired: boolean;
	isSelected: boolean;
}

// checkMapping checks whether the user must select a value for a specific
// property in the mappings (the mapping is required). Additionally it returns
// whether the property is selected.
//
// A property is required if it is the table key, or if it is required for
// create/update (based on export mode and pipeline target) and one of the
// following conditions applies:
// - it is a first-level property;
// - its parent is selected;
// - at least one of its ancestors is selected, and all the intermediate
//   properties between the property and that ancestor are create/update
//   required;
//
// This algorithm is based on the principle, imposed by Meergo, that if you pass
// a value for an object property, then you must also pass a value for each of
// its sub-properties that are created/update required.
const checkMapping = (
	path: string,
	pipeline: TransformedPipeline,
	pipelineType: TransformedPipelineType,
): TransformationPropertyInfo => {
	const property = pipeline.transformation?.mapping?.[path];
	if (property == null) {
		return {
			isRequired: false,
			isSelected: false,
		};
	}

	const paths = Object.keys(pipeline.transformation.mapping);
	const parents: string[] = [];
	for (const p of paths) {
		if (path.startsWith(`${p}.`)) {
			parents.push(p);
		}
	}
	const reversedParents = structuredClone(parents).reverse();

	const hasRequired =
		(pipelineType.target === 'Event' && (property.createRequired || property.updateRequired)) ||
		(pipeline.exportMode != null &&
			((property.createRequired && pipeline.exportMode.includes('Create')) ||
				(property.updateRequired && pipeline.exportMode.includes('Update'))));

	const isFirstLevel = property.indentation === 0;
	const isTableKey = !!pipeline.tableKey && pipeline.tableKey === path;

	let isRequired = false;
	if (!hasRequired) {
		if (isTableKey) {
			isRequired = true;
		}
	} else {
		if (isFirstLevel) {
			isRequired = true;
		} else {
			// A selected property is a property that is mapped, either explicitly (e.g.
			// when the user has set a value in its mapping) or implicitly (e.g. when
			// one of its descendants is explicitly mapped).
			let closestSelectedParent: string;
			for (const parent of reversedParents) {
				const isMapped = pipeline.transformation.mapping[parent].value !== '';
				if (isMapped) {
					closestSelectedParent = parent;
					break;
				}
				const children = paths.filter((key) => key.startsWith(`${parent}.`));
				const hasMappedChild =
					children.findIndex((pa) => pipeline.transformation.mapping[pa].value !== '') !== -1;
				if (hasMappedChild) {
					closestSelectedParent = parent;
					break;
				}
			}

			if (closestSelectedParent != null) {
				const i = reversedParents.findIndex((parent) => parent === closestSelectedParent);
				const intermediateParents = reversedParents.slice(0, i);
				let areIntermediateRequired = true;
				for (const parent of intermediateParents) {
					const p = pipeline.transformation.mapping[parent].full;
					const hasRequired =
						(pipelineType.target === 'Event' && (p.createRequired || p.updateRequired)) ||
						(pipeline.exportMode != null &&
							((p.createRequired && pipeline.exportMode.includes('Create')) ||
								(p.updateRequired && pipeline.exportMode.includes('Update'))));
					if (!hasRequired) {
						areIntermediateRequired = false;
						break;
					}
				}

				isRequired = areIntermediateRequired;
			}
		}
	}

	const isMapped = pipeline.transformation.mapping[path].value !== '';

	// The property is already automatically selected if it is used as the out
	// matching property or if one of its children or parents are mapped.
	const isMatching = pipeline.matching != null && pipeline.matching.out === path;
	const children = paths.filter((pa) => pa.startsWith(`${path}.`));
	const hasMappedChild = children.findIndex((pa) => pipeline.transformation.mapping[pa].value !== '') !== -1;
	const hasMappedParent = parents.findIndex((pa) => pipeline.transformation.mapping[pa].value !== '') !== -1;

	const isSelected = isMapped || isMatching || hasMappedChild || hasMappedParent;

	return {
		isRequired,
		isSelected,
	};
};

// isPathRequired returns whether the user must select a specific path for the
// transformation function (the path is required). Additionally it returns
// whether the property is selected.
//
// A path is required if it is the path of the table key, or if it is the path
// of a property that is required for create/update (based on export mode and
// pipeline target) and for which one of the following conditions applies:
// - it is a first-level property;
// - its parent is selected;
// - at least one of its ancestors is selected, and all the intermediate
//   properties between the property and that ancestor are create/update
//   required;
//
// This algorithm is based on the principle, imposed by Meergo, that if you pass
// a value for an object property, then you must also pass a value for each of
// its sub-properties that are created/update required.
const checkFunctionPath = (
	path: string,
	pipeline: TransformedPipeline,
	pipelineType: TransformedPipelineType,
	role: 'input' | 'output',
	selectedPaths: string[],
): TransformationPropertyInfo => {
	const flatSchema =
		role === 'input' ? flattenSchema(pipelineType.inputSchema) : flattenSchema(pipelineType.outputSchema);

	const property = flatSchema[path];
	if (property == null) {
		return {
			isRequired: false,
			isSelected: false,
		};
	}

	const paths = Object.keys(flatSchema);
	const parents: string[] = [];
	for (const p of paths) {
		if (path.startsWith(`${p}.`)) {
			parents.push(p);
		}
	}
	const reversedParents = structuredClone(parents).reverse();

	const hasRequired =
		(pipelineType.target === 'Event' && (property.createRequired || property.updateRequired)) ||
		(pipeline.exportMode != null &&
			((property.createRequired && pipeline.exportMode.includes('Create')) ||
				(property.updateRequired && pipeline.exportMode.includes('Update'))));

	const isFirstLevel = property.indentation === 0;
	const isTableKey = !!pipeline.tableKey && pipeline.tableKey === path;

	let isRequired = false;
	if (!hasRequired) {
		if (isTableKey) {
			isRequired = true;
		}
	} else {
		if (isFirstLevel) {
			isRequired = true;
		} else {
			// A selected property is a property that is selected, either explicitly
			// (e.g. when the user has flagged its checkbox in the full mode of the
			// transformation) or implicitly (e.g. when one of its descendants is
			// flagged).
			let closestSelectedParent: string;
			for (const parent of reversedParents) {
				const isSelected = selectedPaths.includes(parent);
				if (isSelected) {
					closestSelectedParent = parent;
					break;
				}
				const children = paths.filter((key) => key.startsWith(`${parent}.`));
				const hasSelectedChild = children.findIndex((pa) => selectedPaths.includes(pa)) !== -1;
				if (hasSelectedChild) {
					closestSelectedParent = parent;
					break;
				}
			}

			if (closestSelectedParent != null) {
				const i = reversedParents.findIndex((parent) => parent === closestSelectedParent);
				const intermediateParents = reversedParents.slice(0, i);
				let areIntermediateRequired = true;
				for (const parent of intermediateParents) {
					const p = flatSchema[parent].full;
					const hasRequired =
						(pipelineType.target === 'Event' && (p.createRequired || p.updateRequired)) ||
						(pipeline.exportMode != null &&
							((p.createRequired && pipeline.exportMode.includes('Create')) ||
								(p.updateRequired && pipeline.exportMode.includes('Update'))));
					if (!hasRequired) {
						areIntermediateRequired = false;
						break;
					}
				}

				isRequired = areIntermediateRequired;
			}
		}
	}

	const isFlagged = selectedPaths.includes(path);

	// The property is already automatically selected if it is used as
	// the out matching property or if one of its children or parents
	// are flagged.
	const isMatching = pipeline.matching != null && pipeline.matching.out === path;
	const children = paths.filter((pa: string) => pa.startsWith(`${path}.`));
	const hasFlaggedChild = children.findIndex((pa: string) => selectedPaths.includes(pa)) !== -1;
	const hasFlaggedParent = selectedPaths.findIndex((pa) => path.startsWith(`${pa}.`)) !== -1;

	const isSelected = isFlagged || isMatching || hasFlaggedChild || hasFlaggedParent;

	return {
		isRequired,
		isSelected,
	};
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
const getHierarchicalPaths = (path: string, schema: FlatSchema): hierarchicalPaths => {
	const indentation = schema[path].indentation;
	const ancestors: string[] = [];
	const descendants: string[] = [];
	for (const p in schema) {
		if (schema[p].indentation! < indentation! && path.startsWith(`${p}.`)) {
			ancestors.push(p);
			continue;
		}
		if (schema[p].indentation! > indentation! && p.startsWith(`${path}.`)) {
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
const getSiblingPaths = (path: string, schema: FlatSchema): string[] => {
	const { root, indentation } = schema[path];
	const siblings: string[] = [];
	for (const p in schema) {
		if (p !== path && schema[p].root === root && schema[p].indentation === indentation) {
			siblings.push(p);
		}
	}
	return siblings;
};

const buildHierarchy = (paths: string[], schema: FlatSchema, property: Property): Property => {
	let hierarchy: Property;
	let i = 0;
	for (const p of [...paths].reverse()) {
		let full = schema[p].full;
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

const doesUpdatedAtColumnNeedFormat = (updatedAtColumn: string, schema: ObjectType): boolean => {
	if (updatedAtColumn == null || updatedAtColumn === '') {
		return false;
	}
	const flatInputSchema = flattenSchema(schema);
	const p = flatInputSchema[updatedAtColumn];
	if (p == null) {
		return false;
	}
	const type = p.type;
	return type === 'json' || type === 'string';
};

const validateCustomUpdatedAtFormat = (format: string) => {
	if (format === '') {
		throw new Error('Update time format cannot be empty');
	}
	if (Array.from(format).length > 64) {
		throw new Error('Update time format is longer than 64 characters');
	}
	if (!format.includes('%')) {
		throw new Error(`Update time format "${format}" is not a valid format`);
	}
};

const computeDefaultTransformationFunction = (
	outputSchema: ObjectType,
	language: string,
	connection: TransformedConnection,
	pipelineType: TransformedPipelineType,
): string => {
	if (language !== 'JavaScript' && language !== 'Python') {
		return '';
	}

	let func = RAW_TRANSFORMATION_FUNCTIONS[language].replace(
		'$parameter',
		getTransformationFunctionParameter(connection, pipelineType),
	);

	const isEventSend = connection.isDestination && pipelineType.target.includes('Event');
	if (isEventSend) {
		// Pre-populate the function.
		const requiredProperties: Property[] = [];
		for (const property of outputSchema.properties) {
			if (property.createRequired) {
				requiredProperties.push(property);
			}
		}
		let r = '';
		let imports = new Set();
		for (let i = 0; i < requiredProperties.length; i++) {
			r += '\n\t\t';
			const property = requiredProperties[i];
			r += getTransformationFunctionReturnField(property, 0, language);
			if (language === 'Python') {
				switch (property.type.kind) {
					case 'decimal':
						imports.add('decimal');
						break;
					case 'datetime':
					case 'date':
					case 'time':
						imports.add('datetime');
						break;
					case 'uuid':
						imports.add('uuid');
						break;
					default:
						break;
				}
			}
			if (i === requiredProperties.length - 1) {
				r += '\n\t';
			}
		}
		func = func.replace('$return', r);
		if (imports.size > 0) {
			let f = '';
			for (const im of imports) {
				f += `import ${im}\n`;
			}
			func = f + '\n' + func;
		}
	} else {
		func = func.replace('$return', '');
	}

	return func;
};

const BASE_FUNCTION_RETURN_INDENTATION = 2;

const getTransformationFunctionReturnField = (
	property: Property,
	indentation: number,
	language: 'JavaScript' | 'Python',
): string => {
	let f = '';

	const totalIndentation = BASE_FUNCTION_RETURN_INDENTATION + indentation;

	if (indentation > 0) {
		f += `\n${'\t'.repeat(totalIndentation)}`;
	}

	const isPython = language === 'Python';

	if (isPython) {
		f += `"${property.name}": `;
	} else {
		f += `${property.name}: `;
	}

	switch (property.type.kind) {
		case 'string':
			f += isPython ? '""' : "''";
			break;
		case 'boolean':
			f += isPython ? 'False' : 'false';
			break;
		case 'int':
			if (isPython) {
				f += '0';
			} else {
				if (property.type.bitSize === 64) {
					f += '0n'; // bigint
				} else {
					f += '0';
				}
			}
			break;
		case 'float':
			f += isPython ? '0.0' : '0';
			break;
		case 'decimal':
			f += isPython ? 'decimal.Decimal(0)' : "'0.0'";
			break;
		case 'datetime':
			f += isPython ? 'datetime.datetime(2000, 1, 1, 0, 0, 0)' : "'2000-01-01T00:00:00.000000000Z'";
			break;
		case 'date':
			f += isPython ? 'datetime.date(2000, 1, 1)' : "'2000-01-01'";
			break;
		case 'time':
			f += isPython ? 'datetime.time(0, 0, 0)' : "'00:00:00.000000000'";
			break;
		case 'year':
			f += '2000';
			break;
		case 'uuid':
			f += isPython
				? 'uuid.UUID("00000000-0000-0000-0000-000000000000")'
				: "'00000000-0000-0000-0000-000000000000'";
			break;
		case 'json':
			f += isPython ? 'None' : 'null';
			break;
		case 'ip':
			f += isPython ? '"0.0.0.0"' : "'0.0.0.0'";
			break;
		case 'array':
			f += '[]';
			break;
		case 'object':
			const childProperties = property.type.properties;
			let fragments = [];
			for (const p of childProperties) {
				if (p.createRequired) {
					fragments.push(getTransformationFunctionReturnField(p, indentation + 1, language));
				}
			}
			f += '{';
			f += fragments.join('');
			// Closing bracket
			if (fragments.length > 0) {
				f += '\n' + '\t'.repeat(totalIndentation) + '}';
			} else {
				f += '}';
			}

			break;
		case 'map':
			f += '{}';
			break;
		default:
			break;
	}

	f += ',';

	return f;
};

const getTransformationFunctionParameter = (
	connection: TransformedConnection,
	pipelineType: TransformedPipelineType,
): string => {
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

	let isJsonOrText = property.type === 'json' || property.type === 'string';
	if (property.type === 'array') {
		const typ = property.full.type as ArrayType;
		if (typ.elementType.kind === 'json' || typ.elementType.kind === 'string') {
			isJsonOrText = true;
		}
	}

	let values: string[] | null = [];
	if (isJsonOrText && condition.values != null) {
		const stringType = property.type === 'string' ? (property.full.type as StringType) : null;
		const propertyValues = stringType?.values ?? null;
		const allowsEmptySelection = propertyValues != null && propertyValues.includes('');

		for (const [i, v] of condition.values.entries()) {
			const isFirstValueEmpty = i === 0 && v === '';
			const isSecondBetweenEmpty = i === 1 && v === '' && isBetweenOperator(condition.operator);
			if ((isFirstValueEmpty || isSecondBetweenEmpty) && !allowsEmptySelection) {
				throw new Error(`The filter value on the property "${propertyName}" cannot be empty`);
			}
			if (v === '') {
				if (allowsEmptySelection) {
					values.push('');
				}
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
	const throwIfInvalid = (isValid: boolean, typeKind: string, unsigned?: boolean) => {
		if (!isValid) {
			throw new Error(
				`The filter value on the property "${propertyName}" is not a valid${unsigned === true ? 'unsigned ' : ''} ${typeKind}`,
			);
		}
	};

	if (values == null) {
		return;
	}

	for (const v of values) {
		if (type.kind === 'int') {
			if (type.unsigned) {
				throwIfInvalid(isUnsigned(v), type.kind, true);
			} else {
				throwIfInvalid(isInt(v), type.kind);
			}
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
		} else if (type.kind === 'ip') {
			throwIfInvalid(isIP(v), type.kind);
		} else if (type.kind === 'array') {
			if (type.elementType.kind !== 'json' && type.elementType.kind !== 'string') {
				validateFilterConditionValues(type.elementType, [v], propertyName);
			}
		}
	}
};

const validateMatching = (inMatching: Property, outMatching: Property) => {
	const inTyp = inMatching.type.kind;
	if (inTyp !== 'int' && inTyp !== 'string' && inTyp !== 'uuid') {
		throw new Error(`Matching property cannot be of type "${inTyp}"`);
	}

	// Check that the in property can be converted to the type of the
	// out property.
	const exTyp = outMatching.type.kind;
	const conversionError = new Error(`Matching property of type "${inTyp}" cannot be converted to type "${exTyp}"`);

	if (inTyp === 'int') {
		if (exTyp !== 'int' && exTyp !== 'string') {
			throw conversionError;
		}
	} else if (inTyp === 'string') {
		if (exTyp !== 'int' && exTyp !== 'uuid' && exTyp !== 'string') {
			throw conversionError;
		}
	} else if (inTyp === 'uuid') {
		if (exTyp !== 'uuid' && exTyp !== 'string') {
			throw conversionError;
		}
	}
};

const propertyTypesAreEqual = (aType: Type, bType: Type): boolean => {
	if (aType.kind !== bType.kind) {
		return false;
	}

	if (aType.kind === 'int') {
		const t = bType as IntType;
		return aType.bitSize === t.bitSize && aType.minimum === t.minimum && aType.maximum === t.maximum;
	} else if (aType.kind === 'string') {
		const t = bType as StringType;
		return (
			aType.maxBytes === t.maxBytes &&
			aType.maxLength === t.maxLength &&
			aType.pattern === t.pattern &&
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
	checkMapping,
	checkFunctionPath,
	getCompatibleFilterOperators,
	isUnaryOperator,
	isBetweenOperator,
	isOneOfOperator,
	splitPropertyAndPath,
	getHierarchicalPaths,
	getSiblingPaths,
	doesUpdatedAtColumnNeedFormat,
	computeDefaultTransformationFunction,
	validateAndNormalizeFilterCondition,
	validateMatching,
	propertyTypesAreEqual,
};

export type {
	FlatSchema,
	TransformedProperty,
	TransformedPipelineType,
	TransformedEventType,
	TransformedPipeline,
	PipelineTypeField,
};
