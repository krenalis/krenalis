import Type from './types';
import { Action, ActionType } from './action';

type ConnectorType = 'App' | 'Database' | 'File' | 'Mobile' | 'Server' | 'Storage' | 'Stream' | 'Website';

type ConnectionRole = 'Source' | 'Destination';

type Health = 'Healthy' | 'NoRecentData' | 'RecentError' | 'AccessDenied';

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

interface ConnectionOptions {
	name: string;
	enabled: boolean;
	storage: number;
	compression: Compression;
	websiteHost: string;
	oAuth: string;
}

interface ConnectionStats {
	UserIdentities: number[];
}

export type { Connection, ConnectionRole, Compression, ConnectionOptions, ConnectorType, Health, ConnectionStats };
