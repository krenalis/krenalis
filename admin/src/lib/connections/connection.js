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

	actionTypeByAction(action) {
		if (action.Target === 'Events') {
			return this.actionTypes.find((t) => t.EventType === action.EventType);
		} else {
			return this.actionTypes.find((t) => t.Target === action.Target);
		}
	}
}

export default Connection;
