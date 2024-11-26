import { Action, ActionType } from './action';
import { ConnectorValues } from './responses';

type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'Mobile' | 'Server' | 'Stream' | 'Website';

type ConnectionRole = 'Source' | 'Destination';

type Health = 'Healthy' | 'NoRecentData' | 'RecentError';

interface Connection {
	id: number;
	name: string;
	type: ConnectorType;
	role: ConnectionRole;
	connector: string;
	storage: number;
	compression: Compression;
	strategy?: Strategy | null;
	websiteHost: string;
	sendingMode: SendingMode | null;
	hasUI: boolean;
	enabled: boolean;
	actionsCount: number;
	health: Health;
	actionTypes?: ActionType[];
	actions?: Action[];
	linkedConnections?: number[];
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
	sendingMode?: SendingMode | null;
	uiValues: ConnectorValues;
	linkedConnections: Number[] | null;
}

interface ConnectionToSet {
	name: string;
	enabled: boolean;
	strategy?: Strategy | null;
	websiteHost: string;
	sendingMode?: SendingMode | null;
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
