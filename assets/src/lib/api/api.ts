import call from './call';
import * as http from './http';
import Type, { Property, ObjectType, Role } from './types/types';
import { Connection, ConnectionRole, ConnectionToAdd, ConnectionStats, ConnectionToSet } from './types/connection';
import { Identifiers } from './types/identifiers';
import { ActionTarget, SchedulePeriod, ActionToSet, ExpressionToBeExtracted, Transformation } from './types/action';
import { UI_BASE_PATH } from '../../constants/paths';
import { Connector } from './types/connector';
import { WarehouseMode, WarehouseResponse, WarehouseType } from './types/warehouse';
import Workspace, { AddWorkspaceResponse, PrivacyRegion, DisplayedProperties } from './types/workspace';
import {
	ConnectorUIResponse,
	ConnectorValues,
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
	ChangeUsersSchemaQueriesResponse,
	RePaths,
} from './types/responses';

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
		return await call(`${this.apiURL}/event-schema`, http.GET);
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
		return await call(`${this.apiURL}/transformations`, http.POST, {
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
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.PUT, {
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
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/query/executions`, http.POST, {
			query: query,
			limit: limit,
		});
	};

	records = async (
		connection: number,
		fileConnector: string,
		path: string,
		sheet: string | null,
		compression: string,
		uiValues: ConnectorValues,
		limit: number,
	): Promise<RecordsResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/records`, http.POST, {
			FileConnector: fileConnector,
			Path: path,
			Sheet: sheet,
			Compression: compression,
			UIValues: uiValues,
			Limit: limit,
		});
	};

	sheets = async (
		connection: number,
		fileConnector: string,
		path: string,
		compression: string,
		uiValues: ConnectorValues,
	): Promise<SheetsResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/sheets`, http.POST, {
			FileConnector: fileConnector,
			Path: path,
			Compression: compression,
			UIValues: uiValues,
		});
	};

	ui = async (connection: number): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui`, http.GET);
	};

	uiEvent = async (connection: number, event: string, values: ConnectorValues): Promise<ConnectorUIResponse> => {
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

	actionTypes = async (connection: number) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/action-types`, http.GET);
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
				)}/actions/schemas/Events/${encodeURIComponent(eventType)}`,
				http.GET,
			);
		} else {
			return await call(
				`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/schemas/${encodeURIComponent(
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
			http.PUT,
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
			http.PUT,
			{ SchedulePeriod: schedulePeriod },
		);
	};

	executeAction = async (connection: number, action: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action,
			)}/executions`,
			http.POST,
			{ Reimport: false },
		);
	};

	actionUiEvent = async (
		connection: number,
		action: number,
		event: string,
		values: ConnectorValues,
	): Promise<ConnectorUIResponse> => {
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
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/events/send-previews`,
			http.POST,
			{
				eventType: eventType,
				event: event,
				outSchema: outSchema,
				transformation: transformation,
			},
		);
	};

	removeEventConnection = async (connection: number, eventConnection: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/events/connections/${encodeURIComponent(
				eventConnection,
			)}`,
			http.DELETE,
		);
	};

	addEventConnection = async (connection: number, eventConnection: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/events/connections/${encodeURIComponent(
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
		return await call(`${this.apiURL}/events-listeners`, http.POST, {
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
		return await call(
			`${this.apiURL}/users/${encodeURIComponent(user)}/identities?first=${first}&limit=${limit}`,
			http.GET,
		);
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
		return await call(`${this.apiURL}/connections`, http.POST, {
			connection: connection,
			oAuthToken: oAuthToken,
		});
	};

	oauthToken = async (connector: string, oauthCode: string): Promise<string> => {
		const redirectURI = `${this.origin}${UI_BASE_PATH}oauth/authorize`;
		return await call(`${this.apiURL}/oauth-token`, http.POST, {
			connector: connector,
			oauthCode: oauthCode,
			redirectURI: redirectURI,
		});
	};

	setIdentifiers = async (identifiers: Identifiers): Promise<void> => {
		return await call(`${this.apiURL}/identifiers`, http.PUT, {
			identifiers: identifiers,
		});
	};

	warehouseSettings = async (): Promise<WarehouseResponse> => {
		return await call(`${this.apiURL}/warehouse/settings`, http.GET);
	};

	changeWarehouseMode = async (mode: WarehouseMode): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/mode`, http.PUT, {
			Mode: mode,
		});
	};

	changeWarehouseSettings = async (type: WarehouseType, mode: WarehouseMode, settings: any): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/settings`, http.PUT, {
			Type: type,
			Mode: mode,
			Settings: settings,
		});
	};

	pingWarehouse = async (warehouseType: WarehouseType, settings: any): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/pings`, http.POST, {
			Type: warehouseType,
			Settings: settings,
		});
	};

	connectWarehouse = async (warehouseType: WarehouseType, mode: WarehouseMode, settings: any): Promise<void> => {
		return await call(`${this.apiURL}/warehouse`, http.POST, {
			Type: warehouseType,
			Mode: mode,
			Settings: settings,
		});
	};

	initWarehouse = async (): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/initializations`, http.POST);
	};

	disconnectWarehouse = async (): Promise<void> => {
		return await call(`${this.apiURL}/warehouse`, http.DELETE);
	};

	runIdentityResolution = async (): Promise<void> => {
		return await call(`${this.apiURL}/identity-resolutions`, http.POST);
	};

	changeUsersSchema = async (schema: ObjectType, rePaths: RePaths): Promise<void> => {
		return await call(`${this.apiURL}/user-schema`, http.PUT, {
			schema,
			rePaths,
		});
	};

	changeUsersSchemaQueries = async (
		schema: ObjectType,
		rePaths: RePaths,
	): Promise<ChangeUsersSchemaQueriesResponse> => {
		return await call(`${this.apiURL}/change-users-schema-queries`, http.POST, {
			schema,
			rePaths,
		});
	};
}

class Connectors {
	origin: string;
	apiURL: string;

	constructor(origin: string, apiURL: string) {
		this.origin = origin;
		this.apiURL = apiURL;
	}

	authCodeURL = async (connector: string, role: Role): Promise<authCodeURLResponse> => {
		const redirectURI = `${this.origin}${UI_BASE_PATH}oauth/authorize`;
		return await call(
			`${this.apiURL}/connectors/${connector}/auth-code-url?role=${role}&redirecturi=${encodeURIComponent(redirectURI)}`,
			http.GET,
		);
	};

	find = async (): Promise<Connector[]> => {
		return await call(`${this.apiURL}/connectors`, http.GET);
	};

	get = async (connector: string): Promise<Connector> => {
		return await call(`${this.apiURL}/connectors/${connector}`, http.GET);
	};

	ui = async (
		workspace: number,
		connector: string,
		role: ConnectionRole,
		oauthToken: string,
	): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/workspaces/${workspace}/ui`, http.POST, {
			connector: connector,
			role: role,
			oauthToken: oauthToken,
		});
	};

	uiEvent = async (
		workspace: number,
		connector: string,
		event: string,
		values: ConnectorValues,
		role: ConnectionRole,
		oauthToken: string,
	): Promise<ConnectorUIResponse> => {
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
