import call from './call';
import * as http from './http';
import Type, { Property, ObjectType } from '../../types/external/types';
import {
	Connection,
	ConnectionRole,
	ConnectionToAdd,
	ConnectionStats,
	ConnectionToSet,
} from '../../types/external/connection';
import { Identifiers } from '../../types/external/identifiers';
import {
	ActionTarget,
	SchedulePeriod,
	ActionToSet,
	ExpressionToBeExtracted,
	Transformation,
} from '../../types/external/action';
import { uiBasePath } from '../../constants/path';
import { Connector } from '../../types/external/connector';
import { WarehouseResponse, WarehouseType } from '../../types/external/warehouse';
import Workspace, { AddWorkspaceResponse, PrivacyRegion, DisplayedProperties } from '../../types/external/workspace';
import {
	UIResponse,
	UIValues,
	authCodeURLResponse,
	EventListenerEventsResponse,
	AddEventListenerResponse,
	ActionSchemasResponse,
	ExecQueryResponse,
	CompletePathResponse,
	SheetsResponse,
	RecordsResponse,
	TransformationLanguagesResponse,
	TransformDataResponse,
	Filter,
	FindUsersResponse,
	AppUsersResponse,
	EventPreviewResponse,
	ObservedEvent,
	UserEventsResponse,
	userTraitsResponse,
	Execution,
	MemberToSet,
	Member,
	MemberInvitationResponse,
	UserIdentitiesResponse,
	ConnectionIdentitiesResponse,
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
		return await call(`${this.apiURL}/members/login`, http.POST, { email, password });
	};

	logout = async (): Promise<void> => {
		return await call(`${this.apiURL}/members/logout`, http.POST);
	};

	eventsSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/events-schema`, http.GET);
	};

	validateExpression = async (
		expression: string,
		properties: Property[],
		type: Type,
		required: boolean,
		nullable: boolean,
		signal?: AbortSignal,
	): Promise<string> => {
		return await call(
			`${this.apiURL}/validate-expression`,
			http.POST,
			{
				expression,
				properties,
				type: type,
				required: required,
				nullable: nullable,
			},
			{ signal },
		);
	};

	expressionsProperties = async (expressions: ExpressionToBeExtracted[], schema: Type): Promise<string[]> => {
		return await call(`${this.apiURL}/expressions-properties`, http.POST, {
			expressions,
			schema,
		});
	};

	transformationLanguages = async (): Promise<TransformationLanguagesResponse> => {
		return await call(`${this.apiURL}/transformation-languages`, http.GET);
	};

	transformData = async (
		data: Record<string, any>,
		inSchema: ObjectType,
		outSchema: ObjectType,
		transformation: Transformation,
	): Promise<TransformDataResponse> => {
		return await call(`${this.apiURL}/transform-data`, http.POST, {
			data,
			inSchema,
			outSchema,
			transformation,
		});
	};

	members = async (): Promise<Member[]> => {
		return await call(`${this.apiURL}/members`, http.GET);
	};

	inviteMember = async (email: string): Promise<void> => {
		return await call(`${this.apiURL}/members/invitations`, http.POST, {
			email,
		});
	};

	memberInvitation = async (token: string): Promise<MemberInvitationResponse> => {
		return await call(`${this.apiURL}/members/invitations/${token}`, http.GET);
	};

	acceptInvitation = async (token: string, name: string, password: string): Promise<void> => {
		return await call(`${this.apiURL}/members/invitations/${token}`, http.PUT, {
			name: name,
			password: password,
		});
	};

	member = async (): Promise<Member> => {
		return await call(`${this.apiURL}/members/current`, http.GET);
	};

	updateMember = async (memberToSet: MemberToSet): Promise<void> => {
		return await call(`${this.apiURL}/members/current`, http.PUT, {
			memberToSet: memberToSet,
		});
	};

	deleteMember = async (member: number): Promise<void> => {
		return await call(`${this.apiURL}/members/${member}`, http.DELETE);
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

	executions = async (connection: number): Promise<Execution[]> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/executions`, http.GET);
	};

	identities = async (connection: number, first: number, limit: number): Promise<ConnectionIdentitiesResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/identities`, http.POST, {
			first,
			limit,
		});
	};

	query = async (connection: number, query: string, limit: number): Promise<ExecQueryResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/exec-query`, http.POST, {
			query: query,
			limit: limit,
		});
	};

	records = async (
		connection: number,
		fileConnector: number,
		path: string,
		sheet: string | null,
		compression: string,
		settings: UIValues,
		limit: number,
	): Promise<RecordsResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/records`, http.POST, {
			FileConnector: fileConnector,
			Path: path,
			Sheet: sheet,
			Compression: compression,
			Settings: settings,
			Limit: limit,
		});
	};

	sheets = async (
		connection: number,
		fileConnector: number,
		path: string,
		compression: string,
		settings: UIValues,
	): Promise<SheetsResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/sheets`, http.POST, {
			FileConnector: fileConnector,
			Path: path,
			Compression: compression,
			Settings: settings,
		});
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

	actionUiEvent = async (
		connection: number,
		action: number,
		event: string,
		values: UIValues,
	): Promise<UIResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action,
			)}/ui-event`,
			http.POST,
			{
				event: event,
				values: values,
			},
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
		outSchema: ObjectType,
		transformation?: Transformation,
	): Promise<EventPreviewResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/event-preview`, http.POST, {
			eventType: eventType,
			event: event,
			outSchema: outSchema,
			transformation: transformation,
		});
	};

	removeEventConnection = async (connection: number, eventConnection: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/event-connections/${encodeURIComponent(
				eventConnection,
			)}`,
			http.DELETE,
		);
	};

	addEventConnection = async (connection: number, eventConnection: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/event-connections/${encodeURIComponent(
				eventConnection,
			)}`,
			http.POST,
		);
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

	find = async (properties: string[], filter: Filter, first: number, limit: number): Promise<FindUsersResponse> => {
		return await call(`${this.apiURL}/users`, http.POST, {
			properties: properties,
			filter: filter,
			first: first,
			limit: limit,
		});
	};

	events = async (user: number): Promise<UserEventsResponse> => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/events`, http.GET);
	};

	traits = async (user: number): Promise<userTraitsResponse> => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/traits`, http.GET);
	};

	identities = async (user: number, first: number, limit: number): Promise<UserIdentitiesResponse> => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/identities`, http.POST, {
			first: first,
			limit: limit,
		});
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

	update = async (
		name: string,
		privacyRegion: PrivacyRegion,
		displayedProperties: DisplayedProperties,
	): Promise<void> => {
		return await call(`${this.apiURL}`, http.PUT, {
			name,
			privacyRegion,
			displayedProperties,
		});
	};

	delete = async (): Promise<void> => {
		return await call(`${this.apiURL}`, http.DELETE);
	};

	userSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/user-schema`, http.GET);
	};

	identifiersSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/identifiers-schema`, http.GET);
	};

	addConnection = async (connection: ConnectionToAdd, oAuthToken: string): Promise<number> => {
		return await call(`${this.apiURL}/add-connection`, http.POST, {
			connection: connection,
			oAuthToken: oAuthToken,
		});
	};

	oauthToken = async (connector: number, oauthCode: string): Promise<string> => {
		const redirectURI = `${this.origin}${uiBasePath}oauth/authorize`;
		return await call(`${this.apiURL}/oauth-token`, http.POST, {
			connector: connector,
			oauthCode: oauthCode,
			redirectURI: redirectURI,
		});
	};

	setIdentifiers = async (identifiers: Identifiers): Promise<void> => {
		return await call(`${this.apiURL}/identifiers`, http.POST, {
			identifiers: identifiers,
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

	runIdentityResolution = async (): Promise<void> => {
		return await call(`${this.apiURL}/run-identity-resolution`, http.POST);
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
		const redirectURI = `${this.origin}${uiBasePath}oauth/authorize`;
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
