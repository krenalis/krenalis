import { Action, ActionType } from './action';
import { ConnectorSettings } from './responses';

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
	hasSettings: boolean;
	actionsCount: number;
	health: Health;
	actionTypes?: ActionType[];
	actions?: Action[];
	linkedConnections?: number[];
}

type Compression = '' | 'Zip' | 'Gzip' | 'Snappy';

type Strategy = 'Conversion' | 'Fusion' | 'Isolation' | 'Preservation';

type SendingMode = 'Cloud' | 'Device' | 'Combined';

interface ConnectionToAdd {
	name: string;
	role: string;
	connector: string;
	strategy?: Strategy | null;
	websiteHost: string;
	sendingMode?: SendingMode | null;
	settings?: ConnectorSettings | null;
	linkedConnections: Number[] | null;
}

interface ConnectionToSet {
	name: string;
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
