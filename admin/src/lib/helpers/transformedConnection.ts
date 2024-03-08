import * as variants from '../../constants/variants';
import {
	Connection,
	Health,
	ConnectionRole,
	ConnectorType,
	Compression,
	Strategy,
} from '../../types/external/connection';
import { Action, ActionTarget, ActionType } from '../../types/external/action';
import TransformedConnector from './transformedConnector';

interface ConnectionStatus {
	text: string;
	variant: string;
}

interface BusinessID {
	Name: string;
	Label: string;
}

class TransformedConnection {
	id: number;
	name: string;
	type: ConnectorType;
	role: ConnectionRole;
	connector: TransformedConnector;
	hasSettings: boolean;
	enabled: boolean;
	actionsCount: number;
	health: Health;
	storage: number;
	compression: Compression;
	strategy?: Strategy | null;
	websiteHost: string;
	businessID: BusinessID;
	status: ConnectionStatus;
	description: string;
	linkedFiles?: TransformedConnection[];
	actionTypes?: ActionType[];
	actions?: Action[];

	constructor(
		id: number,
		name: string,
		type: ConnectorType,
		role: ConnectionRole,
		connector: TransformedConnector,
		hasSettings: boolean,
		enabled: boolean,
		actionsCount: number,
		health: Health,
		storage: number,
		compression: Compression,
		strategy: Strategy | null,
		websiteHost: string,
		businessID: BusinessID,
		status: ConnectionStatus,
		description: string,
		linkedFiles?: TransformedConnection[],
		actionTypes?: ActionType[],
		actions?: Action[],
	) {
		this.id = id;
		this.name = name;
		this.type = type;
		this.role = role;
		this.connector = connector;
		this.hasSettings = hasSettings;
		this.enabled = enabled;
		this.actionsCount = actionsCount;
		this.health = health;
		this.storage = storage == null ? 0 : storage;
		this.compression = compression;
		this.strategy = strategy;
		this.websiteHost = websiteHost;
		this.businessID = businessID;
		this.status = status;
		this.description = description;
		this.linkedFiles = linkedFiles;
		this.actionTypes = actionTypes == null ? [] : actionTypes;
		this.actions = actions == null ? [] : actions;
	}

	get isApp() {
		return this.type === 'App';
	}

	get isDatabase() {
		return this.type === 'Database';
	}

	get isFile() {
		return this.type === 'File' && this.storage !== 0;
	}

	get isMobile() {
		return this.type === 'Mobile';
	}

	get isServer() {
		return this.type === 'Server';
	}

	get isStorage() {
		return this.type === 'Storage';
	}

	get isStream() {
		return this.type === 'Stream';
	}

	get isWebsite() {
		return this.type === 'Website';
	}

	get isSource() {
		return this.role === 'Source';
	}

	get isDestination() {
		return this.role === 'Destination';
	}

	get hasIdentities() {
		return this.role === 'Source' && this.type !== 'Storage' && this.type !== 'Stream';
	}

	get hasAnonymousIdentifiers() {
		return this.type === 'Mobile' || this.type === 'Server' || this.type === 'Website';
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
			actionType = connection.actionTypes!.find((t) => t.Target === 'Events' && t.EventType === null);
		} else {
			actionType = connection.actionTypes!.find((t) => t.EventType === eventType);
		}
	} else {
		actionType = connection.actionTypes!.find((t) => t.Target === target);
	}
	return actionType;
};

const getConnectionDescription = (connection: Connection, connector: TransformedConnector): string => {
	let description: string;
	if (connection.Role === 'Source') {
		description = connector.sourceDescription;
	} else {
		description = connector.destinationDescription;
	}
	return description;
};

const getConnectionFullConnector = (connectorID: number, connectors: TransformedConnector[]): TransformedConnector => {
	return connectors.find((c) => c.id === connectorID)!;
};

const getConnectionStatus = (connection: Connection): ConnectionStatus => {
	if (!connection.Enabled) {
		return { text: 'Disabled', variant: variants.NEUTRAL };
	} else {
		switch (connection.Health) {
			case 'Healthy':
				return { text: 'Working properly', variant: variants.SUCCESS };
			case 'NoRecentData':
				return { text: 'No recent Data', variant: variants.DANGER };
			case 'RecentError':
				return { text: 'Recent error', variant: variants.DANGER };
			case 'AccessDenied':
				return { text: 'Access denied', variant: variants.DANGER };
			default:
				return { text: '', variant: '' };
		}
	}
};

const getStorageFileConnections = (
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
	getStorageFileConnections,
};
export type { ConnectionStatus };
