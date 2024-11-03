import { ConnectorValues } from './responses';
import { Compression } from './connection';
import Type, { ObjectType, Property } from './types';

type ActionTarget = 'Events' | 'Users' | 'Groups';

type ActionStep = 'Receive' | 'InputValidation' | 'Filter' | 'Transformation' | 'OutputValidation' | 'Finalize';

type SchedulePeriod = '5m' | '15m' | '30m' | '1h' | '2h' | '3h' | '6h' | '8h' | '12h' | '24h';

type ExportMode = 'CreateOnly' | 'UpdateOnly' | 'CreateOrUpdate';

type Mapping = Record<string, string>;

type TransformationPurpose = 'Create' | 'Update';

interface Transformation {
	Mapping: Mapping | null;
	Function: TransformationFunction | null;
}

interface TransformationFunction {
	Source: string;
	Language: string;
	PreserveJSON: boolean;
	InProperties: string[];
	OutProperties: string[];
}

interface ExpressionToBeExtracted {
	value: string;
	type: Type;
}

interface MatchingProperties {
	Internal: string;
	External: Property | null;
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
	| 'is null'
	| 'is not null'
	| 'exists'
	| 'does not exist';

interface FilterCondition {
	Property: string;
	Operator: FilterOperator | '';
	Values: string[] | null;
}

interface Filter {
	Logical: FilterLogical;
	Conditions: FilterCondition[];
}

interface Action {
	ID: number;
	Connection: number;
	Target: ActionTarget;
	Name: string;
	Enabled: boolean;
	EventType: string | null;
	Running: boolean;
	ScheduleStart: number | null;
	SchedulePeriod: SchedulePeriod | null;
	InSchema: ObjectType | null;
	OutSchema: ObjectType | null;
	Filter: Filter | null;
	Transformation: Transformation | null;
	Query: string | null;
	Path: string | null;
	Table: string | null;
	TableKeyProperty: string | null;
	Sheet: string | null;
	IdentityProperty: string | null;
	LastChangeTimeProperty: string | null;
	LastChangeTimeFormat: string | null;
	FileOrderingPropertyPath: string | null;
	ExportMode: ExportMode | null;
	MatchingProperties: MatchingProperties | null;
	ExportOnDuplicatedUsers: boolean | null;
	Compression: Compression;
	Connector: string;
}

interface ActionType {
	Name: string;
	Description: string;
	Target: ActionTarget;
	EventType: string;
}

interface ActionToSet {
	name: string;
	enabled?: boolean;
	filter?: Filter | null;
	inSchema?: ObjectType;
	outSchema?: ObjectType;
	transformation?: Transformation;
	query?: string;
	path?: string | null;
	tableName?: string | null;
	tableKeyProperty?: string | null;
	sheet?: string | null;
	IdentityProperty?: string | null;
	LastChangeTimeProperty?: string | null;
	LastChangeTimeFormat?: string | null;
	FileOrderingPropertyPath?: string | null;
	exportMode?: ExportMode | null;
	matchingProperties?: MatchingProperties | null;
	exportOnDuplicatedUsers?: boolean | null;
	Compression: Compression;
	Connector: string;
	UIValues?: ConnectorValues;
}

interface ActionError {
	Action: number;
	Step: ActionStep;
	Count: number;
	Message: string;
	LastOccurred: Date;
}

interface ActionMetrics {
	start: Date;
	end: Date;
	passed: [number, number, number, number, number, number][];
	failed: [number, number, number, number, number, number][];
}

export type {
	ActionTarget,
	Transformation,
	TransformationFunction,
	ExportMode,
	MatchingProperties,
	SchedulePeriod,
	Filter,
	FilterOperator,
	FilterLogical,
	FilterCondition,
	Action,
	ActionType,
	ActionToSet,
	ExpressionToBeExtracted,
	Mapping,
	TransformationPurpose,
	ActionStep,
	ActionError,
	ActionMetrics,
};
