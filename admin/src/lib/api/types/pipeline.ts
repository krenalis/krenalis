import { ConnectorSettings } from './responses';
import { Compression } from './connection';
import Type, { ObjectType } from './types';

type PipelineTarget = 'Event' | 'User' | 'Group';

type PipelineStep = 'Receive' | 'InputValidation' | 'Filter' | 'Transformation' | 'OutputValidation' | 'Finalize';

type SchedulePeriod = 'Off' | '5m' | '15m' | '30m' | '1h' | '2h' | '3h' | '6h' | '8h' | '12h' | '24h';

type ExportMode = 'CreateOnly' | 'UpdateOnly' | 'CreateOrUpdate';

type Mapping = Record<string, string>;

type TransformationPurpose = 'Import' | 'Create' | 'Update';

interface Transformation {
	mapping: Mapping | null;
	function: TransformationFunction | null;
}

interface TransformationFunction {
	source: string;
	language: string;
	preserveJSON: boolean;
	inPaths: string[];
	outPaths: string[];
}

interface ExpressionToBeExtracted {
	value: string;
	type: Type;
}

interface Matching {
	in: string; // property path.
	out: string; // property path.
}

type FilterLogical = 'and' | 'or';

type FilterOperator =
	| 'is'
	| 'is not'
	| 'is less than'
	| 'is less than or equal to'
	| 'is greater than'
	| 'is greater than or equal to'
	| 'is between'
	| 'is not between'
	| 'contains'
	| 'does not contain'
	| 'is one of'
	| 'is not one of'
	| 'starts with'
	| 'ends with'
	| 'is before'
	| 'is on or before'
	| 'is after'
	| 'is on or after'
	| 'is true'
	| 'is false'
	| 'is empty'
	| 'is not empty'
	| 'is null'
	| 'is not null'
	| 'exists'
	| 'does not exist';

interface FilterCondition {
	property: string;
	operator: FilterOperator | '';
	values: string[] | null;
}

interface Filter {
	logical: FilterLogical;
	conditions: FilterCondition[];
}

interface Pipeline {
	id: number;
	connection: number;
	target: PipelineTarget;
	name: string;
	enabled: boolean;
	eventType: string | null;
	running: boolean;
	scheduleStart: number | null;
	schedulePeriod: SchedulePeriod | null;
	inSchema: ObjectType | null;
	outSchema: ObjectType | null;
	filter: Filter | null;
	transformation: Transformation | null;
	query: string | null;
	path: string | null;
	tableName: string | null;
	tableKey: string | null;
	sheet: string | null;
	identityColumn: string | null;
	updatedAtColumn: string | null;
	updatedAtFormat: string | null;
	incremental: boolean | null;
	exportMode: ExportMode | null;
	matching: Matching | null;
	updateOnDuplicates: boolean | null;
	compression: Compression;
	orderBy: string | null;
	format: string;
}

interface PipelineType {
	name: string;
	description: string;
	target: PipelineTarget;
	eventType: string;
}

interface PipelineToSet {
	name: string;
	enabled?: boolean;
	filter?: Filter | null;
	inSchema?: ObjectType;
	outSchema?: ObjectType;
	transformation?: Transformation;
	query?: string;
	path?: string | null;
	tableName?: string | null;
	tableKey?: string | null;
	sheet?: string | null;
	identityColumn?: string | null;
	updatedAtColumn?: string | null;
	updatedAtFormat?: string | null;
	incremental?: boolean;
	exportMode?: ExportMode;
	matching?: Matching;
	updateOnDuplicates?: boolean;
	compression: Compression;
	orderBy?: string | null;
	format: string;
	formatSettings?: ConnectorSettings | null;
}

interface PipelineError {
	pipeline: number;
	step: number;
	count: number;
	message: string;
	lastOccurred: Date;
}

interface PipelineMetrics {
	start: Date;
	end: Date;
	passed: [number, number, number, number, number, number][];
	failed: [number, number, number, number, number, number][];
}

export type {
	PipelineTarget,
	Transformation,
	TransformationFunction,
	ExportMode,
	Matching,
	SchedulePeriod,
	Filter,
	FilterOperator,
	FilterLogical,
	FilterCondition,
	Pipeline,
	PipelineType,
	PipelineToSet,
	ExpressionToBeExtracted,
	Mapping,
	TransformationPurpose,
	PipelineStep,
	PipelineError,
	PipelineMetrics,
};
