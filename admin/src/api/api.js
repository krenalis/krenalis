import call from './call';
import * as http from './http';

class API {
	constructor(url) {
		this.baseURL = url;
		this.connections = new Connections(url);
		this.eventlisteners = new Eventlisteners(url);
		this.users = new Users(url);
		this.workspace = new Workspace(url);
		this.connectors = new Connectors(url);
	}

	predefinedMappings = async () => {
		return await call(`${this.baseURL}/api/predefined-mappings`, http.GET);
	};

	eventsSchema = async () => {
		return await call(`${this.baseURL}/api/events-schema`, http.GET);
	};
}

class Connections {
	constructor(url) {
		this.baseURL = url;
	}

	find = async () => {
		return await call(`${this.baseURL}/api/connections`, http.GET);
	};

	get = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}`, http.GET);
	};

	delete = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}`, http.DELETE);
	};

	reload = async (connection) => {
		return await call(`/api/connections/${encodeURIComponent(connection)}/reload`, http.POST);
	};

	stats = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/stats`, http.GET);
	};

	schema = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/schema`, http.GET);
	};

	imports = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/imports`, http.GET);
	};

	import = async (connection, reimport) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/import`, http.POST, {
			reimport: reimport,
		});
	};

	query = async (connection, query, limit) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/exec-query`, http.POST, {
			query: query,
			limit: limit,
		});
	};

	records = async (connection, path, sheet, limit) => {
		let queryString = `?limit=${limit}`;
		if (path != null) {
			queryString += `&path=${encodeURIComponent(path)}`;
		}
		if (sheet != null) {
			queryString += `&sheet=${encodeURIComponent(sheet)}`;
		}
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/records${queryString}`,
			http.GET
		);
	};

	setStorage = async (connection, storage) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/storage/${storage}`,
			http.PUT
		);
	};

	setStatus = async (connection, enabled) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/status`, http.POST, {
			enabled: enabled,
		});
	};

	mappings = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/mappings`, http.GET);
	};

	setMappings = async (connection, mappings) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/mappings`,
			http.PUT,
			mappings
		);
	};

	transformation = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/transformation`, http.GET);
	};

	setTransformation = async (connection, transformation) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/transformation`,
			http.PUT,
			transformation
		);
	};

	ui = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/ui`, http.GET);
	};

	uiEvent = async (connection, event, values) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/ui-event`, http.POST, {
			event: event,
			values: values,
		});
	};

	keys = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/keys`, http.GET);
	};

	generateKey = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/keys`, http.POST);
	};

	revokeKey = async (connection, key) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/keys/${encodeURIComponent(key)}`,
			http.DELETE
		);
	};

	actionTypes = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/action-types`, http.GET);
	};

	usersAction = async (connection) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/action-types/Users`,
			http.GET
		);
	};

	groupsAction = async (connection) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/action-types/Groups`,
			http.GET
		);
	};

	eventsAction = async (connection) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/action-types/Events`,
			http.GET
		);
	};

	eventAction = async (connection, eventType) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/action-types/Events/${encodeURIComponent(
				eventType
			)}`,
			http.GET
		);
	};

	actions = async (connection) => {
		return await call(`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/actions`, http.GET);
	};

	addAction = async (connection, actionObject) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/actions`,
			http.POST,
			actionObject
		);
	};

	setAction = async (connection, action, actionObject) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}`,
			http.PUT,
			actionObject
		);
	};

	deleteAction = async (connection, action) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}`,
			http.DELETE
		);
	};

	setActionStatus = async (connection, action, enabled) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action
			)}/status`,
			http.POST,
			{ Enabled: enabled }
		);
	};

	setActionSchedulePeriod = async (connection, action, schedulePeriod) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action
			)}/schedule-period`,
			http.POST,
			{ SchedulePeriod: schedulePeriod }
		);
	};

	executeAction = async (connection, action, reimport) => {
		return await call(
			`${this.baseURL}/api/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action
			)}/execute`,
			http.POST,
			{ Reimport: reimport }
		);
	};
}

class Eventlisteners {
	constructor(url) {
		this.baseURL = url;
	}

	add = async (size, source, server, stream) => {
		return await call(`${this.baseURL}/api/event-listeners`, http.PUT, {
			size: size,
			source: source,
			server: server,
			stream: stream,
		});
	};

	remove = async (eventListener) => {
		return await call(`${this.baseURL}/api/event-listeners/${encodeURIComponent(eventListener)}`, http.DELETE);
	};

	events = async (eventListener) => {
		return await call(`${this.baseURL}/api/event-listeners/${encodeURIComponent(eventListener)}/events`, http.GET);
	};
}

class Users {
	constructor(url) {
		this.baseURL = url;
	}

	find = async (properties, start, end) => {
		return await call(`${this.baseURL}/api/users`, http.POST, { properties: properties, start: start, end: end });
	};
}

class Workspace {
	constructor(url) {
		this.baseURL = url;
	}

	userSchema = async () => {
		return await call(`/api/workspace/user-schema`, http.GET);
	};

	reloadSchemas = async () => {
		return await call(`/api/workspace/reload-schemas`, http.POST);
	};

	addConnection = async (connector, role, settings, options) => {
		return await call(`/api/workspace/add-connection`, http.POST, {
			connector: connector,
			role: role,
			settings: settings,
			options: options,
		});
	};

	oauthToken = async (connector, oauthCode) => {
		return await call(`/api/workspace/oauth-token`, http.POST, {
			connector: connector,
			oauthCode: oauthCode,
		});
	};
}

class Connectors {
	constructor(url) {
		this.baseURL = url;
	}

	find = async () => {
		return await call(`${this.baseURL}/api/connectors`, http.GET);
	};

	get = async (id) => {
		return await call(`${this.baseURL}/api/connectors/${id}`, http.GET);
	};

	ui = async (id, role, oauthToken) => {
		return await call(`${this.baseURL}/api/connectors/${id}/ui`, http.POST, { role: role, oauthToken: oauthToken });
	};

	uiEvent = async (id, event, values, role, oauthToken) => {
		return await call(`${this.baseURL}/api/connectors/${id}/ui-event`, http.POST, {
			event: event,
			values: values,
			role: role,
			oauthToken: oauthToken,
		});
	};
}

export default API;
