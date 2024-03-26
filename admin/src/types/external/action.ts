import { Filter, UIValues } from './api';
import { Compression } from './connection';
import Type, { ObjectType, Property } from './types';

type ActionTarget = 'Events' | 'Users' | 'Groups';

type SchedulePeriod = '5m' | '15m' | '30m' | '1h' | '2h' | '3h' | '6h' | '8h' | '12h' | '24h';

type ExportMode = 'CreateOnly' | 'UpdateOnly' | 'CreateOrUpdate';

type Mapping = Record<string, string>;

interface Transformation {
	Mapping: Mapping | null;
	Function: TransformationFunction | null;
}

interface TransformationFunction {
	Source: string;
	Language: string;
}

interface ExpressionToBeExtracted {
	value: string;
	type: Type;
}

interface MatchingProperties {
	Internal: string;
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
	Transformation: Transformation | null;
	Query: string | null;
	Path: string | null;
	Table: string | null;
	Sheet: string | null;
	IdentityColumn: string | null;
	TimestampColumn: string | null;
	TimestampFormat: string | null;
	BusinessID: string | null;
	ExportMode: ExportMode | null;
	MatchingProperties: MatchingProperties | null;
	ExportOnDuplicatedUsers: boolean | null;
	Compression: Compression;
	Connector: number;
	Settings: UIValues;
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
	sheet?: string | null;
	IdentityColumn?: string | null;
	TimestampColumn?: string | null;
	TimestampFormat?: string | null;
	BusinessID?: string | null;
	exportMode?: ExportMode | null;
	matchingProperties?: MatchingProperties | null;
	exportOnDuplicatedUsers?: boolean | null;
	Compression: Compression;
	Connector: number;
	Settings?: UIValues;
}

export type {
	ActionTarget,
	Transformation,
	TransformationFunction,
	ExportMode,
	MatchingProperties,
	SchedulePeriod,
	Action,
	ActionType,
	ActionToSet,
	ExpressionToBeExtracted,
	Mapping,
};
