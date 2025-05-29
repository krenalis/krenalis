import {
	Connection,
	Health,
	ConnectionRole,
	ConnectorType,
	Compression,
	Strategy,
	SendingMode,
} from '../api/types/connection';
import { Action, ActionTarget, ActionType } from '../api/types/action';
import TransformedConnector from './connector';
import { Variant } from '../../components/routes/App/App.types';
import { ConnectorTarget } from '../api/types/connector';

interface ConnectionStatus {
	text: string;
	variant: Variant | '';
}

class TransformedConnection {
	id: number;
	name: string;
	connector: TransformedConnector;
	role: ConnectionRole;
	actionsCount: number;
	health: Health;
	storage: number;
	compression: Compression;
	strategy?: Strategy | null;
	sendingMode: SendingMode | null;
	status: ConnectionStatus;
	description: string;
	linkedFiles?: TransformedConnection[];
	actionTypes?: ActionType[];
	actions?: Action[];
	linkedConnections?: number[];

	constructor(
		id: number,
		name: string,
		connector: TransformedConnector,
		role: ConnectionRole,
		actionsCount: number,
		health: Health,
		storage: number,
		compression: Compression,
		strategy: Strategy | null,
		sendingMode: SendingMode | null,
		status: ConnectionStatus,
		description: string,
		linkedFiles?: TransformedConnection[],
		actionTypes?: ActionType[],
		actions?: Action[],
		linkedConnections?: number[],
	) {
		this.id = id;
		this.name = name;
		this.connector = connector;
		this.role = role;
		this.actionsCount = actionsCount;
		this.health = health;
		this.storage = storage == null ? 0 : storage;
		this.compression = compression;
		this.strategy = strategy;
		this.sendingMode = sendingMode;
		this.status = status;
		this.description = description;
		this.linkedFiles = linkedFiles;
		this.actionTypes = actionTypes == null ? [] : actionTypes;
		this.actions = actions == null ? [] : actions;
		if (linkedConnections) {
			this.linkedConnections = linkedConnections;
		}
	}

	get isApp() {
		return this.connector.type === 'App';
	}

	get isDatabase() {
		return this.connector.type === 'Database';
	}

	get isFile() {
		return this.connector.type === 'File' && this.storage !== 0;
	}

	get isFileStorage() {
		return this.connector.type === 'FileStorage';
	}

	get isSDK() {
		return this.connector.type === 'SDK';
	}

	get isStream() {
		return this.connector.type === 'Stream';
	}

	get isSource() {
		return this.role === 'Source';
	}

	get isDestination() {
		return this.role === 'Destination';
	}

	get isEventBased() {
		return this.connector.type === 'SDK';
	}

	get hasIdentities() {
		return this.role === 'Source' && this.connector.type !== 'Stream';
	}

	get hasAnonymousIdentifiers() {
		return this.connector.type === 'SDK';
	}

	get hasSettings(): boolean {
		return this.connector.hasSettings(this.role);
	}
}

const getActionTypeFromConnection = (
	connection: TransformedConnection,
	target: ActionTarget,
	eventType: string | null,
): ActionType | undefined => {
	let actionType: ActionType | undefined;
	if (target === 'Events') {
		if (eventType == null) {
			actionType = connection.actionTypes!.find((t) => t.target === 'Events' && t.eventType === null);
		} else {
			actionType = connection.actionTypes!.find((t) => t.eventType === eventType);
		}
	} else {
		actionType = connection.actionTypes!.find((t) => t.target === target);
	}
	return actionType;
};

const getConnectionDescription = (connection: Connection, connector: TransformedConnector): string => {
	let description: string;
	if (connection.role === 'Source') {
		description = connector.asSource.summary;
	} else {
		description = connector.asDestination.summary;
	}
	return description;
};

const getConnectionFullConnector = (
	connectorName: string,
	connectors: TransformedConnector[],
): TransformedConnector => {
	return connectors.find((c) => c.name === connectorName)!;
};

const getConnectionStatus = (connection: Connection): ConnectionStatus => {
	switch (connection.health) {
		case 'Healthy':
			return { text: 'Working properly', variant: 'success' };
		case 'NoRecentData':
			return { text: 'No recent Data', variant: 'danger' };
		case 'RecentError':
			return { text: 'Recent error', variant: 'danger' };
		default:
			return { text: '', variant: '' };
	}
};

const isSourceEventConnection = (role: ConnectionRole, type: ConnectorType): boolean => {
	return role === 'Source' && type === 'SDK';
};

const isEventConnection = (role: ConnectionRole, type: ConnectorType, targets: ConnectorTarget[]): boolean => {
	return (
		(role === 'Source' && type === 'SDK') ||
		(role === 'Destination' && type === 'App' && targets.includes('Events'))
	);
};

const getFileStorageConnections = (
	storageID: number,
	connections: TransformedConnection[],
): TransformedConnection[] => {
	return connections.filter((c) => c.storage === storageID);
};

export default TransformedConnection;
export {
	getActionTypeFromConnection,
	getConnectionDescription,
	getConnectionFullConnector,
	getConnectionStatus,
	getFileStorageConnections,
	isEventConnection,
	isSourceEventConnection,
};
export type { ConnectionStatus };
