import Type from './types';

type ActionTarget = 'Events' | 'Users' | 'Groups';

type SchedulePeriod = '5m' | '15m' | '30m' | '1h' | '2h' | '3h' | '6h' | '8h' | '12h' | '24h';

type ExportMode = 'CreateOnly' | 'UpdateOnly' | 'CreateOrUpdate';

type ActionFilterLogical = 'all' | 'any';

type Mapping = Record<string, string>;

interface Transformation {
	Func: string;
}

interface MappingExpression {
	value: string;
	type: Type;
	nullable: boolean;
}

interface MatchingProperties {
	Internal: string;
	External: string;
}

interface ActionFilterCondition {
	Property: string;
	Operator: string;
	Value: string;
}

interface ActionFilter {
	Logical: ActionFilterLogical;
	Conditions: ActionFilterCondition[];
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
	InSchema: Type | null;
	OutSchema: Type | null;
	Filter: ActionFilter | null;
	Mapping: Mapping | null;
	Transformation: Transformation | null;
	Identifiers: string[] | null;
	Query: string | null;
	Path: string | null;
	Table: string | null;
	Sheet: string | null;
	ExportMode: ExportMode | null;
	MatchingProperties: MatchingProperties | null;
}

interface ActionType {
	Name: string;
	Description: string;
	Target: ActionTarget;
	EventType: string;
	MissingSchema: boolean;
}

interface ActionToSet {
	name: string;
	enabled?: boolean;
	filter?: ActionFilter | null;
	inSchema?: Type;
	outSchema?: Type;
	identifiers?: string[];
	mapping?: Mapping;
	transformation?: Transformation | null;
	query?: string;
	path?: string | null;
	tableName?: string | null;
	sheet?: string | null;
	exportMode?: ExportMode | null;
	matchingProperties?: MatchingProperties | null;
}

export {
	ActionTarget,
	ActionFilter,
	Transformation,
	ExportMode,
	MatchingProperties,
	SchedulePeriod,
	Action,
	ActionType,
	ActionToSet,
	MappingExpression,
	Mapping,
};
