import { Action, ActionType } from './action';
import { ConnectorValues } from './responses';

type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'Mobile' | 'Server' | 'Stream' | 'Website';

type ConnectionRole = 'Source' | 'Destination';

type Health = 'Healthy' | 'NoRecentData' | 'RecentError' | 'AccessDenied';

interface Connection {
	ID: number;
	Name: string;
	Type: ConnectorType;
	Role: ConnectionRole;
	Connector: string;
	Storage: number;
	Compression: Compression;
	Strategy?: Strategy | null;
	WebsiteHost: string;
	SendingMode: SendingMode | null;
	HasUI: boolean;
	Enabled: boolean;
	ActionsCount: number;
	Health: Health;
	ActionTypes?: ActionType[];
	Actions?: Action[];
	EventConnections?: number[];
}

type Compression = '' | 'Zip' | 'Gzip' | 'Snappy';

type Strategy = 'AB-C' | 'ABC' | 'A-B-C' | 'AC-B';

type SendingMode = 'Cloud' | 'Device' | 'Combined';

interface ConnectionToAdd {
	name: string;
	role: string;
	enabled: boolean;
	connector: string;
	strategy?: Strategy | null;
	websiteHost: string;
	SendingMode?: SendingMode | null;
	uiValues: ConnectorValues;
	eventConnections: Number[] | null;
}

interface ConnectionToSet {
	name: string;
	enabled: boolean;
	strategy?: Strategy | null;
	websiteHost: string;
	SendingMode?: SendingMode | null;
}

export type {
	Connection,
	ConnectionRole,
	Compression,
	Strategy,
	SendingMode,
	ConnectionToAdd,
	ConnectionToSet,
	ConnectorType,
	Health,
};
