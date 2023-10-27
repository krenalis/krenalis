import { Action, ActionType } from './action';
import { UIValues } from './api';

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
	Compression: Compression;
	WebsiteHost: string;
	HasSettings: boolean;
	Enabled: boolean;
	ActionsCount: number;
	Health: Health;
	ActionTypes?: ActionType[];
	Actions?: Action[];
}

type Compression = '' | 'Zip' | 'Gzip' | 'Snappy';

interface ConnectionToAdd {
	name: string;
	role: string;
	enabled: boolean;
	connector: number;
	storage: number;
	compression: Compression;
	websiteHost: string;
	settings: UIValues;
}

interface ConnectionToSet {
	name: string;
	enabled: boolean;
	storage: number;
	compression: Compression;
	websiteHost: string;
}

interface ConnectionStats {
	UserIdentities: number[];
}

export type {
	Connection,
	ConnectionRole,
	Compression,
	ConnectionToAdd,
	ConnectionToSet,
	ConnectorType,
	Health,
	ConnectionStats,
};
