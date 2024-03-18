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
	Strategy?: Strategy | null;
	WebsiteHost: string;
	BusinessID: BusinessID;
	HasSettings: boolean;
	Enabled: boolean;
	ActionsCount: number;
	Health: Health;
	ActionTypes?: ActionType[];
	Actions?: Action[];
}

type Compression = '' | 'Zip' | 'Gzip' | 'Snappy';

type Strategy = 'AB-C' | 'ABC' | 'A-B-C' | 'AC-B';

interface BusinessID {
	Name: string;
	Label: string;
}

interface ConnectionToAdd {
	name: string;
	role: string;
	enabled: boolean;
	connector: number;
	strategy?: Strategy | null;
	websiteHost: string;
	businessID: BusinessID;
	settings: UIValues;
}

interface ConnectionToSet {
	name: string;
	enabled: boolean;
	strategy?: Strategy | null;
	websiteHost: string;
	businessID: BusinessID;
}

interface ConnectionStats {
	UserIdentities: number[];
}

export type {
	BusinessID,
	Connection,
	ConnectionRole,
	Compression,
	Strategy,
	ConnectionToAdd,
	ConnectionToSet,
	ConnectorType,
	Health,
	ConnectionStats,
};
