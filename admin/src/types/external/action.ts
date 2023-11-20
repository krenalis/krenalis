import { Filter } from './api';
import Type, { ObjectType, Property } from './types';

type ActionTarget = 'Events' | 'Users' | 'Groups';

type SchedulePeriod = '5m' | '15m' | '30m' | '1h' | '2h' | '3h' | '6h' | '8h' | '12h' | '24h';

type ExportMode = 'CreateOnly' | 'UpdateOnly' | 'CreateOrUpdate';

type Mapping = Record<string, string>;

interface Transformation {
	Source: string;
	Language: string;
}

interface MappingExpression {
	value: string;
	type: Type;
	nullable: boolean;
}

interface MatchingProperties {
	Internal: Property | null;
	External: Property | null;
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
	Mapping: Mapping | null;
	Transformation: Transformation | null;
	Query: string | null;
	Path: string | null;
	Table: string | null;
	Sheet: string | null;
	IdentityColumn: string | null;
	TimestampColumn: string | null;
	TimestampFormat: string | null;
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
	filter?: Filter | null;
	inSchema?: ObjectType;
	outSchema?: ObjectType;
	mapping?: Mapping;
	transformation?: Transformation | null;
	query?: string;
	path?: string | null;
	tableName?: string | null;
	sheet?: string | null;
	IdentityColumn?: string | null;
	TimestampColumn?: string | null;
	TimestampFormat?: string | null;
	exportMode?: ExportMode | null;
	matchingProperties?: MatchingProperties | null;
}

export type {
	ActionTarget,
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
