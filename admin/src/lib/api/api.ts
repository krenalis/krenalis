import call from './call';
import * as http from './http';
import Type, { ObjectType } from '../../types/external/types';
import {
	Connection,
	ConnectionRole,
	ConnectionToAdd,
	ConnectionStats,
	ConnectionToSet,
} from '../../types/external/connection';
import { AnonymousIdentifiers, Identifiers } from '../../types/external/identifiers';
import {
	ActionTarget,
	SchedulePeriod,
	ActionToSet,
	MappingExpression,
	Mapping,
	Transformation,
} from '../../types/external/action';
import { adminBasePath } from '../../constants/path';
import { Connector } from '../../types/external/connector';
import { WarehouseResponse, WarehouseType } from '../../types/external/warehouse';
import Workspace, { AddWorkspaceResponse, PrivacyRegion } from '../../types/external/workspace';
import {
	UIResponse,
	UIValues,
	authCodeURLResponse,
	Import,
	EventListenerEventsResponse,
	AddEventListenerResponse,
	ActionSchemasResponse,
	ExecQueryResponse,
	CompletePathResponse,
	SheetsResponse,
	RecordsResponse,
	TransformationLanguagesResponse,
	TransformationPreviewResponse,
	Filter,
	FindUsersResponse,
	AppUsersResponse,
	EventPreviewResponse,
	ObservedEvent,
	UserEventsResponse,
	userTraitsResponse,
} from '../../types/external/api';

class API {
	apiURL: string;
	workspaces: Workspaces;
	connectors: Connectors;

	constructor(origin: string, workspaceID: number) {
		const apiURL = origin + '/api';
		this.apiURL = apiURL;
		this.workspaces = new Workspaces(origin, apiURL, workspaceID);
		this.connectors = new Connectors(origin, apiURL);
	}

	login = async (email: string, password: string): Promise<[number, string]> => {
		return await call(`${adminBasePath}`, http.POST, { email, password });
	};

	eventsSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/events-schema`, http.GET);
	};

	validateExpression = async (
		expression: string,
		schema: Type,
		destinationPropertyType: Type,
		destinationPropertyNullable: boolean,
	): Promise<string> => {
		return await call(`${this.apiURL}/validate-expression`, http.POST, {
			expression,
			schema,
			destinationPropertyType,
			destinationPropertyNullable,
		});
	};

	expressionsProperties = async (expressions: MappingExpression[], schema: Type): Promise<string[]> => {
		return await call(`${this.apiURL}/expressions-properties`, http.POST, {
			expressions,
			schema,
		});
	};

	transformationLanguages = async (): Promise<TransformationLanguagesResponse> => {
		return await call(`${this.apiURL}/transformation-languages`, http.GET);
	};

	transformationPreview = async (
		data: Record<string, any>,
		inSchema: ObjectType,
		outSchema: ObjectType,
		mapping: Mapping,
		transformation: Transformation,
	): Promise<TransformationPreviewResponse> => {
		return await call(`${this.apiURL}/transformation-preview`, http.POST, {
			data,
			inSchema,
			outSchema,
			mapping,
			transformation,
		});
	};
}

class Connections {
	apiURL: string;

	constructor(url: string) {
		this.apiURL = url;
	}

	find = async (): Promise<Connection[]> => {
		const connections = await call(`${this.apiURL}/connections`, http.GET);
		return connections as Connection[];
	};

	get = async (connection: number): Promise<Connection> => {
		const c = await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.GET);
		return c as Connection;
	};

	set = async (connection: number, connectionToSet: ConnectionToSet) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.POST, {
			connection: connectionToSet,
		});
	};

	delete = async (connection: number): Promise<void> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.DELETE);
	};

	stats = async (connection: number): Promise<ConnectionStats> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/stats`, http.GET);
	};

	imports = async (connection: number): Promise<Import[]> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/imports`, http.GET);
	};

	query = async (connection: number, query: string, limit: number): Promise<ExecQueryResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/exec-query`, http.POST, {
			query: query,
			limit: limit,
		});
	};

	records = async (
		connection: number,
		path: string,
		sheet: string | null,
		limit: number,
	): Promise<RecordsResponse> => {
		let queryString = `?limit=${limit}`;
		if (path != null) {
			queryString += `&path=${encodeURIComponent(path)}`;
		}
		if (sheet != null) {
			queryString += `&sheet=${encodeURIComponent(sheet)}`;
		}
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/records${queryString}`,
			http.GET,
		);
	};

	sheets = async (connection: number, path: string): Promise<SheetsResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/sheets?path=${encodeURIComponent(path)}`,
			http.GET,
		);
	};

	ui = async (connection: number): Promise<UIResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui`, http.GET);
	};

	uiEvent = async (connection: number, event: string, values: UIValues): Promise<UIResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui-event`, http.POST, {
			event: event,
			values: values,
		});
	};

	keys = async (connection: number): Promise<string[]> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys`, http.GET);
	};

	generateKey = async (connection: number): Promise<string> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys`, http.POST);
	};

	revokeKey = async (connection: number, key: string): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys/${encodeURIComponent(key)}`,
			http.DELETE,
		);
	};

	actionSchemas = async (
		connection: number,
		target: ActionTarget,
		eventType: string,
	): Promise<ActionSchemasResponse> => {
		if (eventType != null) {
			return await call(
				`${this.apiURL}/connections/${encodeURIComponent(
					connection,
				)}/action-schemas/Events/${encodeURIComponent(eventType)}`,
				http.GET,
			);
		} else {
			return await call(
				`${this.apiURL}/connections/${encodeURIComponent(connection)}/action-schemas/${encodeURIComponent(
					target,
				)}`,
				http.GET,
			);
		}
	};

	addAction = async (
		connection: number,
		target: ActionTarget,
		eventType: string,
		actionToSet: ActionToSet,
	): Promise<number> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions`, http.POST, {
			target: target,
			eventType: eventType,
			action: actionToSet,
		});
	};

	setAction = async (connection: number, action: number, actionToSet: ActionToSet): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}`,
			http.PUT,
			actionToSet,
		);
	};

	deleteAction = async (connection: number, action: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}`,
			http.DELETE,
		);
	};

	setActionStatus = async (connection: number, action: number, enabled: boolean): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(action)}/status`,
			http.POST,
			{ Enabled: enabled },
		);
	};

	setActionSchedulePeriod = async (
		connection: number,
		action: number,
		schedulePeriod: SchedulePeriod,
	): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action,
			)}/schedule-period`,
			http.POST,
			{ SchedulePeriod: schedulePeriod },
		);
	};

	executeAction = async (connection: number, action: number, reimport: boolean): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action,
			)}/execute`,
			http.POST,
			{ Reimport: reimport },
		);
	};

	completePath = async (storageConnection: number, path: string): Promise<CompletePathResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(storageConnection)}/complete-path/${encodeURIComponent(
				path,
			)}`,
			http.GET,
		);
	};

	tableSchema = async (connection: number, tableName: string): Promise<ObjectType> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/tables/${encodeURIComponent(
				tableName,
			)}/schema`,
			http.GET,
		);
	};

	appUsers = async (connection: number, schema: ObjectType, cursor?: string): Promise<AppUsersResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/app-users`, http.POST, {
			Schema: schema,
			Cursor: cursor,
		});
	};

	eventPreview = async (
		connection: number,
		eventType: string,
		event: ObservedEvent,
		mapping?: Mapping,
		transformation?: Transformation,
	): Promise<EventPreviewResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/event-preview`, http.POST, {
			eventType: eventType,
			event: event,
			mapping: mapping,
			transformation: transformation,
		});
	};
}

class Eventlisteners {
	apiURL: string;

	constructor(url: string) {
		this.apiURL = url;
	}

	add = async (size: number, source: number, onlyValid: boolean): Promise<AddEventListenerResponse> => {
		return await call(`${this.apiURL}/event-listeners`, http.PUT, {
			size: size,
			source: source,
			onlyValid: onlyValid,
		});
	};

	remove = async (eventListener: string): Promise<void> => {
		return await call(`${this.apiURL}/event-listeners/${encodeURIComponent(eventListener)}`, http.DELETE);
	};

	events = async (eventListener: string): Promise<EventListenerEventsResponse> => {
		return await call(`${this.apiURL}/event-listeners/${encodeURIComponent(eventListener)}/events`, http.GET);
	};
}

class Users {
	apiURL: string;

	constructor(url: string) {
		this.apiURL = url;
	}

	find = async (filter: Filter, properties: string[], start: number, end: number): Promise<FindUsersResponse> => {
		return await call(`${this.apiURL}/users`, http.POST, {
			filter: filter,
			properties: properties,
			start: start,
			end: end,
		});
	};

	events = async (user: number): Promise<UserEventsResponse> => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/events`, http.GET);
	};

	traits = async (user: number): Promise<userTraitsResponse> => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/traits`, http.GET);
	};
}

class Workspaces {
	origin: string;
	baseAPIURL: string;
	apiURL: string;
	connections: Connections;
	eventlisteners: Eventlisteners;
	users: Users;

	constructor(origin: string, apiURL: string, workspaceID: number) {
		this.origin = origin;
		this.baseAPIURL = apiURL + '/workspaces';
		const url = this.baseAPIURL + `/${workspaceID}`;
		this.apiURL = url;
		this.connections = new Connections(url);
		this.eventlisteners = new Eventlisteners(url);
		this.users = new Users(url);
	}

	list = async (): Promise<Workspace[]> => {
		return await call(`${this.baseAPIURL}`, http.GET);
	};

	add = async (name: string, privacyRegion: PrivacyRegion): Promise<AddWorkspaceResponse> => {
		return await call(`${this.baseAPIURL}`, http.POST, {
			name,
			privacyRegion,
		});
	};

	get = async (): Promise<Workspace> => {
		return await call(`${this.apiURL}`, http.GET);
	};

	update = async (name: string, privacyRegion: PrivacyRegion): Promise<void> => {
		return await call(`${this.apiURL}`, http.PUT, {
			name,
			privacyRegion,
		});
	};

	delete = async (): Promise<void> => {
		return await call(`${this.apiURL}`, http.DELETE);
	};

	userSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/user-schema`, http.GET);
	};

	reloadSchemas = async (): Promise<void> => {
		return await call(`${this.apiURL}/reload-schemas`, http.POST);
	};

	addConnection = async (connection: ConnectionToAdd, oAuthToken: string): Promise<number> => {
		return await call(`${this.apiURL}/add-connection`, http.POST, {
			connection: connection,
			oAuthToken: oAuthToken,
		});
	};

	oauthToken = async (connector: number, oauthCode: string): Promise<string> => {
		const redirectURI = `${this.origin}${adminBasePath}oauth/authorize`;
		return await call(`${this.apiURL}/oauth-token`, http.POST, {
			connector: connector,
			oauthCode: oauthCode,
			redirectURI: redirectURI,
		});
	};

	setIdentifiers = async (identifiers: Identifiers, anonymousIdentifiers: AnonymousIdentifiers): Promise<void> => {
		return await call(`${this.apiURL}/identifiers`, http.POST, {
			identifiers: identifiers,
			anonymousIdentifiers: anonymousIdentifiers,
		});
	};

	warehouseSettings = async (): Promise<WarehouseResponse> => {
		return await call(`${this.apiURL}/warehouse-settings`, http.GET);
	};

	changeWarehouseSettings = async (type: WarehouseType, settings: any): Promise<void> => {
		return await call(`${this.apiURL}/warehouse-settings`, http.PUT, {
			Type: type,
			Settings: settings,
		});
	};

	pingWarehouse = async (warehouseType: WarehouseType, settings: any): Promise<void> => {
		return await call(`${this.apiURL}/ping-warehouse`, http.POST, {
			Type: warehouseType,
			Settings: settings,
		});
	};

	connectWarehouse = async (warehouseType: WarehouseType, settings: any): Promise<void> => {
		return await call(`${this.apiURL}/connect-warehouse`, http.POST, {
			Type: warehouseType,
			Settings: settings,
		});
	};

	disconnectWarehouse = async (): Promise<void> => {
		return await call(`${this.apiURL}/disconnect-warehouse`, http.POST);
	};
}

class Connectors {
	origin: string;
	apiURL: string;

	constructor(origin: string, apiURL: string) {
		this.origin = origin;
		this.apiURL = apiURL;
	}

	authCodeURL = async (connector: number): Promise<authCodeURLResponse> => {
		const redirectURI = `${this.origin}${adminBasePath}oauth/authorize`;
		return await call(
			`${this.apiURL}/connectors/${connector}/auth-code-url?redirecturi=${encodeURIComponent(redirectURI)}`,
			http.GET,
		);
	};

	find = async (): Promise<Connector[]> => {
		return await call(`${this.apiURL}/connectors`, http.GET);
	};

	get = async (connector: number): Promise<Connector> => {
		return await call(`${this.apiURL}/connectors/${connector}`, http.GET);
	};

	ui = async (
		workspace: number,
		connector: number,
		role: ConnectionRole,
		oauthToken: string,
	): Promise<UIResponse> => {
		return await call(`${this.apiURL}/workspaces/${workspace}/ui`, http.POST, {
			connector: connector,
			role: role,
			oauthToken: oauthToken,
		});
	};

	uiEvent = async (
		workspace: number,
		connector: number,
		event: string,
		values: UIValues,
		role: ConnectionRole,
		oauthToken: string,
	): Promise<UIResponse> => {
		return await call(`${this.apiURL}/workspaces/${workspace}/ui-event`, http.POST, {
			connector: connector,
			event: event,
			values: values,
			role: role,
			oauthToken: oauthToken,
		});
	};
}

export default API;
