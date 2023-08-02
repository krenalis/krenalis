import call from './call';
import * as http from './http';
import Type from '../../types/types';
import {
	ActionTarget,
	SchedulePeriod,
	ConnectionRole,
	ExpressionToBeExtracted,
	Compression,
	ActionToSet,
	ConnectionOptions,
	AnonymousIdentifiers,
} from '../../types/connection';
import { adminBasePath } from '../../constants/path';

class API {
	apiURL: string;
	connections: Connections;
	eventlisteners: Eventlisteners;
	users: Users;
	workspace: Workspace;
	connectors: Connectors;

	constructor(baseURL: string) {
		const apiURL = baseURL + '/api';
		this.apiURL = apiURL;
		this.connections = new Connections(apiURL);
		this.eventlisteners = new Eventlisteners(apiURL);
		this.users = new Users(apiURL);
		this.workspace = new Workspace(baseURL, apiURL);
		this.connectors = new Connectors(baseURL, apiURL);
	}

	login = async (email: string, password: string) => {
		return await call(`${adminBasePath}`, http.POST, { email, password });
	};

	eventsSchema = async () => {
		return await call(`${this.apiURL}/events-schema`, http.GET);
	};

	validateExpression = async (
		expression: string,
		schema: Type,
		destinationPropertyType: Type,
		destinationPropertyNullable: boolean
	) => {
		return await call(`${this.apiURL}/validate-expression`, http.POST, {
			expression,
			schema,
			destinationPropertyType,
			destinationPropertyNullable,
		});
	};

	expressionsProperties = async (expressions: ExpressionToBeExtracted[], schema: Type) => {
		return await call(`${this.apiURL}/expressions-properties`, http.POST, {
			expressions,
			schema,
		});
	};
}

class Connections {
	apiURL: string;

	constructor(url: string) {
		this.apiURL = url;
	}

	find = async () => {
		return await call(`${this.apiURL}/connections`, http.GET);
	};

	get = async (connection: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.GET);
	};

	delete = async (connection: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.DELETE);
	};

	stats = async (connection: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/stats`, http.GET);
	};

	imports = async (connection: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/imports`, http.GET);
	};

	query = async (connection: number, query: string, limit: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/exec-query`, http.POST, {
			query: query,
			limit: limit,
		});
	};

	records = async (connection: number, path: string, sheet: string, limit: number) => {
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

	sheets = async (connection: number, path: string) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/sheets?path=${encodeURIComponent(path)}`
		);
	};

	setStorage = async (connection: number, storage: number, compression: Compression) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/storage`, http.POST, {
			storage: storage,
			compression: compression,
		});
	};

	setStatus = async (connection: number, enabled: boolean) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/status`, http.POST, {
			enabled: enabled,
		});
	};

	ui = async (connection: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui`, http.GET);
	};

	uiEvent = async (connection: number, event: string, values: Map<string, any>) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui-event`, http.POST, {
			event: event,
			values: values,
		});
	};

	keys = async (connection: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys`, http.GET);
	};

	generateKey = async (connection: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys`, http.POST);
	};

	revokeKey = async (connection: number, key: string) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys/${encodeURIComponent(key)}`,
			http.DELETE
		);
	};

	actionSchemas = async (connection: number, target: ActionTarget, eventType: string) => {
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

	addAction = async (connection: number, target: ActionTarget, eventType: string, actionToSet: ActionToSet) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions`, http.POST, {
			target: target,
			eventType: eventType,
			action: actionToSet,
		});
	};

	setAction = async (connection: number, action: number, actionToSet: ActionToSet) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}`,
			http.PUT,
			actionToSet
		);
	};

	deleteAction = async (connection: number, action: number) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}`,
			http.DELETE
		);
	};

	setActionStatus = async (connection: number, action: number, enabled: boolean) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}/status`,
			http.POST,
			{ Enabled: enabled }
		);
	};

	setActionSchedulePeriod = async (connection: number, action: number, schedulePeriod: SchedulePeriod) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action
			)}/schedule-period`,
			http.POST,
			{ SchedulePeriod: schedulePeriod }
		);
	};

	executeAction = async (connection: number, action: number, reimport: boolean) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action
			)}/execute`,
			http.POST,
			{ Reimport: reimport }
		);
	};

	completePath = async (storageConnection: number, path: string) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(storageConnection)}/complete-path/${encodeURIComponent(
				path
			)}`,
			http.GET
		);
	};
}

class Eventlisteners {
	apiURL: string;

	constructor(url: string) {
		this.apiURL = url;
	}

	add = async (size: number, source: number, server: number, stream: number) => {
		return await call(`${this.apiURL}/event-listeners`, http.PUT, {
			size: size,
			source: source,
			server: server,
			stream: stream,
		});
	};

	remove = async (eventListener: string) => {
		return await call(`${this.apiURL}/event-listeners/${encodeURIComponent(eventListener)}`, http.DELETE);
	};

	events = async (eventListener: string) => {
		return await call(`${this.apiURL}/event-listeners/${encodeURIComponent(eventListener)}/events`, http.GET);
	};
}

class Users {
	apiURL: string;

	constructor(url: string) {
		this.apiURL = url;
	}

	find = async (properties: string[], start: number, end: number) => {
		return await call(`${this.apiURL}/users`, http.POST, {
			properties: properties,
			start: start,
			end: end,
		});
	};

	events = async (user: number) => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/events`, http.GET);
	};

	traits = async (user: number) => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/traits`, http.GET);
	};
}

class Workspace {
	baseURL: string;
	apiURL: string;

	constructor(baseURL: string, apiURL: string) {
		this.baseURL = baseURL;
		this.apiURL = apiURL;
	}

	get = async () => {
		return await call(`${this.apiURL}/workspace`, http.GET);
	};

	userSchema = async () => {
		return await call(`${this.apiURL}/workspace/user-schema`, http.GET);
	};

	reloadSchemas = async () => {
		return await call(`${this.apiURL}/workspace/reload-schemas`, http.POST);
	};

	addConnection = async (
		connector: number,
		role: ConnectionRole,
		settings: Map<string, any>,
		options: ConnectionOptions
	) => {
		return await call(`${this.apiURL}/workspace/add-connection`, http.POST, {
			connector: connector,
			role: role,
			settings: settings,
			options: options,
		});
	};

	oauthToken = async (connector: number, oauthCode: string) => {
		const redirectURI = `${this.baseURL}${adminBasePath}oauth/authorize`;
		return await call(`${this.apiURL}/workspace/oauth-token`, http.POST, {
			connector: connector,
			oauthCode: oauthCode,
			redirectURI: redirectURI,
		});
	};

	anonymousIdentifiers = async (identifiers: AnonymousIdentifiers) => {
		return await call(`${this.apiURL}/workspace/anonymous-identifiers`, http.POST, {
			AnonymousIdentifiers: identifiers,
		});
	};
}

class Connectors {
	baseURL: string;
	apiURL: string;

	constructor(baseURL: string, apiURL: string) {
		this.baseURL = baseURL;
		this.apiURL = apiURL;
	}

	authCodeURL = async (connector: number) => {
		const redirectURI = `${this.baseURL}${adminBasePath}oauth/authorize`;
		return await call(
			`${this.apiURL}/connectors/${connector}/auth-code-url?redirecturi=${encodeURIComponent(redirectURI)}`,
			http.GET
		);
	};

	find = async () => {
		return await call(`${this.apiURL}/connectors`, http.GET);
	};

	get = async (connector: number) => {
		return await call(`${this.apiURL}/connectors/${connector}`, http.GET);
	};

	ui = async (connector: number, role: ConnectionRole, oauthToken: string) => {
		return await call(`${this.apiURL}/connectors/${connector}/ui`, http.POST, {
			role: role,
			oauthToken: oauthToken,
		});
	};

	uiEvent = async (
		connector: number,
		event: string,
		values: Map<string, any>,
		role: ConnectionRole,
		oauthToken: string
	) => {
		return await call(`${this.apiURL}/connectors/${connector}/ui-event`, http.POST, {
			event: event,
			values: values,
			role: role,
			oauthToken: oauthToken,
		});
	};
}

export default API;
