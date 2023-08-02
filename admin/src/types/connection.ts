import Type from './types';

type ConnectorType = 'App' | 'Database' | 'File' | 'Mobile' | 'Server' | 'Storage' | 'Stream' | 'Website';

type ConnectionRole = 'Source' | 'Destination';

type Health = 'Healthy' | 'NoRecentData' | 'RecentError' | 'AccessDenied';

type ActionTarget = 'Events' | 'Users' | 'Groups';

type SchedulePeriod = '5m' | '15m' | '30m' | '1h' | '2h' | '3h' | '6h' | '8h' | '12h' | '24h';

type ActionFilterLogical = 'all' | 'any';

type ExportMode = 'CreateOnly' | 'UpdateOnly' | 'CreateOrUpdate';

interface ActionFilterCondition {
	Property: string;
	Operator: string;
	Value: string;
}

interface ActionFilter {
	Logical: ActionFilterLogical;
	Conditions: ActionFilterCondition[];
}

interface ActionType {
	Name: string;
	Description: string;
	Target: ActionTarget;
	EventType: string;
	MissingSchema: boolean;
}

interface Transformation {
	Func: string;
	In: string[];
	Out: string[];
}

interface MatchingProperties {
	Internal: string;
	External: string;
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
	Mapping: Map<string, string> | null;
	Transformation: Transformation | null;
	Identifiers: string[] | null;
	Query: string | null;
	Path: string | null;
	Table: string | null;
	Sheet: string | null;
	ExportMode: ExportMode | null;
	MatchingProperties: MatchingProperties | null;
}

interface Connection {
	ID: number;
	Name: string;
	Type: ConnectorType;
	Role: ConnectionRole;
	Connector: number;
	Storage: number;
	HasSettings: boolean;
	Enabled: boolean;
	ActionsCount: number;
	Health: Health;
	ActionTypes?: ActionType[];
	Actions?: Action[];
}

type Compression = '' | 'Zip' | 'Gzip' | 'Snappy';

interface ExpressionToBeExtracted {
	value: string;
	type: Type;
	nullable: boolean;
}

interface ActionToSet {
	name: string;
	enabled?: boolean;
	filter?: ActionFilter | null;
	inSchema?: Type;
	outSchema?: Type;
	identifiers?: string[];
	mapping?: Map<string, string> | null;
	transformation?: Transformation | null;
	query?: string;
	path?: string;
	tableName?: string;
	sheet?: string;
	exportMode?: ExportMode | null;
	matchingProperties?: MatchingProperties | null;
}

interface ConnectionOptions {
	name: string;
	enabled: boolean;
	storage: number;
	compression: Compression;
	websiteHost: string;
	oAuth: string;
}

interface AnonymousIdentifiers {
	priority: string[];
	mapping: Map<string, string>;
}

export default Connection;
export {
	ActionTarget,
	ActionFilter,
	Transformation,
	ExportMode,
	MatchingProperties,
	SchedulePeriod,
	ConnectionRole,
	Compression,
	ExpressionToBeExtracted,
	ActionToSet,
	ConnectionOptions,
	AnonymousIdentifiers,
};
