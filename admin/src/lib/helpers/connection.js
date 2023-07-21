import * as variants from '../../constants/variants';

class Connection {
	constructor(
		id,
		name,
		type,
		role,
		connector,
		hasSettings,
		enabled,
		actionsCount,
		health,
		action,
		storage,
		actionTypes,
		actions
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
		this.action = action;
		this.storage = storage == null ? 0 : storage;
		this.actionTypes = actionTypes == null ? [] : actionTypes;
		this.actions = actions == null ? [] : actions;
	}

	static toConnectionsArray(connections) {
		return connections.map(
			(connection) =>
				new Connection(
					connection.ID,
					connection.Name,
					connection.Type,
					connection.Role,
					connection.Connector,
					connection.HasSettings,
					connection.Enabled,
					connection.ActionsCount,
					connection.Health,
					connection.Action,
					connection.Storage,
					connection.ActionTypes,
					connection.Actions
				)
		);
	}

	static new(connection) {
		return new Connection(
			connection.ID,
			connection.Name,
			connection.Type,
			connection.Role,
			connection.Connector,
			connection.HasSettings,
			connection.Enabled,
			connection.ActionsCount,
			connection.Health,
			connection.Action,
			connection.Storage,
			connection.ActionTypes,
			connection.Actions
		);
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
}

const getActionTypeFromConnection = (connection, target, eventType) => {
	let actionType;
	if (target === 'Events') {
		if (eventType == null) {
			actionType = connection.actionTypes.find((t) => t.Target === 'Events' && t.EventType === null);
		} else {
			actionType = connection.actionTypes.find((t) => t.EventType === eventType);
		}
	} else {
		actionType = connection.actionTypes.find((t) => t.Target === target);
	}
	return actionType;
};

const getConnectionDescription = (connection) => {
	let description;
	if (connection.isSource) {
		description = connection.connector.sourceDescription;
	} else {
		description = connection.connector.destinationDescription;
	}
	return description;
};

const getConnectionFullConnector = (connectorID, connectors) => {
	return connectors.find((c) => c.id === connectorID);
};

const getConnectionStatus = (connection) => {
	if (!connection.enabled) {
		return { text: 'Disabled', variant: variants.NEUTRAL };
	} else {
		switch (connection.health) {
			case 'Healthy':
				return { text: 'Working properly', variant: variants.SUCCESS };
			case 'NoRecentData':
				return { text: 'No recent Data', variant: variants.DANGER };
			case 'RecentError':
				return { text: 'Recent error', variant: variants.DANGER };
			case 'AccessDenied':
				return { text: 'Access denied', variant: variants.DANGER };
			default:
				return { text: null, variant: null };
		}
	}
};

const getStorageFileConnections = (storageID, connections) => {
	return connections.filter((c) => c.storage === storageID);
};

export default Connection;
export {
	getActionTypeFromConnection,
	getConnectionDescription,
	getConnectionFullConnector,
	getConnectionStatus,
	getStorageFileConnections,
};
