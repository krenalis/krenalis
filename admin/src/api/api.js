import call from './call';
import * as http from './http';

class API {
	constructor(baseURL) {
		let apiURL = baseURL + '/api';
		this.apiURL = apiURL;
		this.connections = new Connections(apiURL);
		this.eventlisteners = new Eventlisteners(apiURL);
		this.users = new Users(apiURL);
		this.workspace = new Workspace(baseURL, apiURL);
		this.connectors = new Connectors(baseURL, apiURL);
	}

	predefinedMappings = async () => {
		return await call(`${this.apiURL}/predefined-mappings`, http.GET);
	};

	eventsSchema = async () => {
		return await call(`${this.apiURL}/events-schema`, http.GET);
	};
}

class Connections {
	constructor(url) {
		this.apiURL = url;
	}

	find = async () => {
		return await call(`${this.apiURL}/connections`, http.GET);
	};

	get = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.GET);
	};

	delete = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.DELETE);
	};

	reload = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/reload`, http.POST);
	};

	stats = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/stats`, http.GET);
	};

	schema = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/schema`, http.GET);
	};

	imports = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/imports`, http.GET);
	};

	import = async (connection, reimport) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/import`, http.POST, {
			reimport: reimport,
		});
	};

	query = async (connection, query, limit) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/exec-query`, http.POST, {
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
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/records${queryString}`,
			http.GET
		);
	};

	sheets = async (connection, path) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/sheets?path=${encodeURIComponent(path)}`
		);
	};

	setStorage = async (connection, storage) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/storage/${storage}`, http.PUT);
	};

	setStatus = async (connection, enabled) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/status`, http.POST, {
			enabled: enabled,
		});
	};

	mappings = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/mappings`, http.GET);
	};

	setMappings = async (connection, mappings) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/mappings`, http.PUT, mappings);
	};

	transformation = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/transformation`, http.GET);
	};

	setTransformation = async (connection, transformation) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/transformation`,
			http.PUT,
			transformation
		);
	};

	ui = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui`, http.GET);
	};

	uiEvent = async (connection, event, values) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui-event`, http.POST, {
			event: event,
			values: values,
		});
	};

	keys = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys`, http.GET);
	};

	generateKey = async (connection) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys`, http.POST);
	};

	revokeKey = async (connection, key) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys/${encodeURIComponent(key)}`,
			http.DELETE
		);
	};

	actionSchemas = async (connection, target, eventType) => {
		if (eventType != null) {
			return await call(
				`${this.apiURL}/connections/${encodeURIComponent(
					connection
				)}/action-schemas/Events/${encodeURIComponent(eventType)}`,
				http.GET
			);
		} else {
			return await call(
				`${this.apiURL}/connections/${encodeURIComponent(connection)}/action-schemas/${encodeURIComponent(
					target
				)}`,
				http.GET
			);
		}
	};

	addAction = async (connection, actionObject) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions`,
			http.POST,
			actionObject
		);
	};

	setAction = async (connection, action, actionObject) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}`,
			http.PUT,
			actionObject
		);
	};

	deleteAction = async (connection, action) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}`,
			http.DELETE
		);
	};

	setActionStatus = async (connection, action, enabled) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}/status`,
			http.POST,
			{ Enabled: enabled }
		);
	};

	setActionSchedulePeriod = async (connection, action, schedulePeriod) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action
			)}/schedule-period`,
			http.POST,
			{ SchedulePeriod: schedulePeriod }
		);
	};

	executeAction = async (connection, action, reimport) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action
			)}/execute`,
			http.POST,
			{ Reimport: reimport }
		);
	};
}

class Eventlisteners {
	constructor(url) {
		this.apiURL = url;
	}

	add = async (size, source, server, stream) => {
		return await call(`${this.apiURL}/event-listeners`, http.PUT, {
			size: size,
			source: source,
			server: server,
			stream: stream,
		});
	};

	remove = async (eventListener) => {
		return await call(`${this.apiURL}/event-listeners/${encodeURIComponent(eventListener)}`, http.DELETE);
	};

	events = async (eventListener) => {
		return await call(`${this.apiURL}/event-listeners/${encodeURIComponent(eventListener)}/events`, http.GET);
	};
}

class Users {
	constructor(url) {
		this.apiURL = url;
	}

	find = async (properties, start, end) => {
		return await call(`${this.apiURL}/users`, http.POST, { properties: properties, start: start, end: end });
	};

	events = async (user) => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/events`, http.GET);
	};

	traits = async (user) => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/traits`, http.GET);
	};
}

class Workspace {
	constructor(baseURL, apiURL) {
		this.baseURL = baseURL;
		this.apiURL = apiURL;
	}

	userSchema = async () => {
		return await call(`${this.apiURL}/workspace/user-schema`, http.GET);
	};

	reloadSchemas = async () => {
		return await call(`${this.apiURL}/workspace/reload-schemas`, http.POST);
	};

	addConnection = async (connector, role, settings, options) => {
		return await call(`${this.apiURL}/workspace/add-connection`, http.POST, {
			connector: connector,
			role: role,
			settings: settings,
			options: options,
		});
	};

	oauthToken = async (connector, oauthCode) => {
		const redirectURI = `${this.baseURL}/admin/oauth/authorize`;
		return await call(`${this.apiURL}/workspace/oauth-token`, http.POST, {
			connector: connector,
			oauthCode: oauthCode,
			redirectURI: redirectURI,
		});
	};
}

class Connectors {
	constructor(baseURL, apiURL) {
		this.baseURL = baseURL;
		this.apiURL = apiURL;
	}

	authCodeURL = async (connector) => {
		const redirectURI = `${this.baseURL}/admin/oauth/authorize`;
		return await call(
			`${this.apiURL}/connectors/${connector}/auth-code-url?redirecturi=${encodeURIComponent(redirectURI)}`,
			http.GET
		);
	};

	find = async () => {
		return await call(`${this.apiURL}/connectors`, http.GET);
	};

	get = async (connector) => {
		return await call(`${this.apiURL}/connectors/${connector}`, http.GET);
	};

	ui = async (connector, role, oauthToken) => {
		return await call(`${this.apiURL}/connectors/${connector}/ui`, http.POST, {
			role: role,
			oauthToken: oauthToken,
		});
	};

	uiEvent = async (connector, event, values, role, oauthToken) => {
		return await call(`${this.apiURL}/connectors/${connector}/ui-event`, http.POST, {
			event: event,
			values: values,
			role: role,
			oauthToken: oauthToken,
		});
	};
}

export default API;
