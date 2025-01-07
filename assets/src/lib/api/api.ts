import call from './call';
import * as http from './http';
import Type, { Property, ObjectType, Role } from './types/types';
import { Connection, ConnectionRole, ConnectionToAdd, ConnectionToSet } from './types/connection';
import { Identifiers } from './types/identifiers';
import {
	ActionTarget,
	SchedulePeriod,
	ActionToSet,
	ExpressionToBeExtracted,
	Transformation,
	TransformationPurpose,
	ActionStep,
	ActionMetrics,
	Filter,
} from './types/action';
import { UI_BASE_PATH } from '../../constants/paths';
import { Connector } from './types/connector';
import { WarehouseMode, WarehouseResponse, WarehouseSettings } from './types/warehouse';
import Workspace, {
	CreateWorkspaceResponse,
	DisplayedProperties,
	LastIdentityResolution,
	PrimarySources,
	PrivacyRegion,
} from './types/workspace';
import {
	ActionErrorsResponse,
	ActionSchemasResponse,
	AppUsersResponse,
	CompletePathResponse,
	ConnectionIdentitiesResponse,
	ConnectorSettings,
	ConnectorUIResponse,
	CreateEventListenerResponse,
	Event,
	EventListenerEventsResponse,
	ExecQueryResponse,
	Execution,
	FindUsersResponse,
	Member,
	MemberInvitationResponse,
	MemberToSet,
	PreviewSendEventResponse,
	PreviewUserSchemaUpdateResponse,
	RePaths,
	RecordsResponse,
	SheetsResponse,
	TransformDataResponse,
	TransformationLanguagesResponse,
	UserEventsResponse,
	UserIdentitiesResponse,
	authCodeURLResponse,
	userTraitsResponse,
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
		signal?: AbortSignal,
	): Promise<string> => {
		return await call(
			`${this.apiURL}/validate-expression`,
			http.POST,
			{
				expression,
				properties,
				type,
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
		purpose: TransformationPurpose,
	): Promise<TransformDataResponse> => {
		return await call(`${this.apiURL}/transformations`, http.POST, {
			data,
			inSchema,
			outSchema,
			transformation,
			purpose,
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
			name,
			password,
		});
	};

	member = async (): Promise<Member> => {
		return await call(`${this.apiURL}/members/current`, http.GET);
	};

	updateMember = async (memberToSet: MemberToSet): Promise<void> => {
		return await call(`${this.apiURL}/members/current`, http.PUT, {
			memberToSet,
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

	update = async (id: number, connection: ConnectionToSet) => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(id)}`, http.PUT, {
			connection,
		});
	};

	delete = async (connection: number): Promise<void> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}`, http.DELETE);
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

	execQuery = async (connection: number, query: string, limit: number): Promise<ExecQueryResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/query/executions`, http.POST, {
			query: query,
			limit: limit,
		});
	};

	records = async (
		connection: number,
		format: string,
		path: string,
		sheet: string | null,
		compression: string,
		formatSettings: ConnectorSettings,
		limit: number,
	): Promise<RecordsResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/records`, http.POST, {
			format,
			path,
			sheet,
			compression,
			formatSettings,
			limit,
		});
	};

	sheets = async (
		connection: number,
		format: string,
		path: string,
		compression: string,
		formatSettings: ConnectorSettings,
	): Promise<SheetsResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/sheets`, http.POST, {
			format,
			path,
			compression,
			formatSettings,
		});
	};

	ui = async (connection: number): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui`, http.GET);
	};

	uiEvent = async (connection: number, event: string, settings: ConnectorSettings): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui-event`, http.POST, {
			event,
			settings,
		});
	};

	writeKeys = async (connection: number): Promise<string[]> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys`, http.GET);
	};

	createWriteKey = async (connection: number): Promise<string> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/keys`, http.POST);
	};

	deleteWriteKey = async (connection: number, key: string): Promise<void> => {
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

	createAction = async (
		connection: number,
		target: ActionTarget,
		eventType: string,
		action: ActionToSet,
	): Promise<number> => {
		return await call(`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions`, http.POST, {
			target,
			eventType,
			action,
		});
	};

	updateAction = async (connection: number, id: number, action: ActionToSet): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(id)}`,
			http.PUT,
			action,
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
			{ enabled },
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
			{ schedulePeriod },
		);
	};

	executeAction = async (connection: number, action: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action,
			)}/executions`,
			http.POST,
			{ reload: false },
		);
	};

	actionUiEvent = async (
		connection: number,
		action: number,
		event: string,
		formatSettings: ConnectorSettings,
	): Promise<ConnectorUIResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/${encodeURIComponent(
				action,
			)}/ui-event`,
			http.POST,
			{
				event,
				formatSettings,
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
			schema,
			cursor,
		});
	};

	previewSendEvent = async (
		connection: number,
		eventType: string,
		event: Event,
		outSchema: ObjectType,
		transformation?: Transformation,
	): Promise<PreviewSendEventResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/events/send-previews`,
			http.POST,
			{
				eventType,
				event,
				outSchema,
				transformation,
			},
		);
	};

	unlinkConnection = async (connection: number, connection2: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/linked-connections/${encodeURIComponent(
				connection2,
			)}`,
			http.DELETE,
		);
	};

	linkConnection = async (connection: number, connection2: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/linked-connections/${encodeURIComponent(
				connection2,
			)}`,
			http.POST,
		);
	};
}

class EventListeners {
	apiURL: string;

	constructor(url: string) {
		this.apiURL = url;
	}

	create = async (size: number, filter: Filter): Promise<CreateEventListenerResponse> => {
		return await call(`${this.apiURL}/event-listeners`, http.POST, {
			size,
			filter,
		});
	};

	delete = async (eventListener: string): Promise<void> => {
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

	find = async (
		properties: string[],
		filter: Filter,
		order: string,
		orderDesc: boolean,
		first: number,
		limit: number,
	): Promise<FindUsersResponse> => {
		return await call(`${this.apiURL}/users`, http.POST, {
			properties,
			filter,
			order,
			orderDesc,
			first,
			limit,
		});
	};

	events = async (user: string): Promise<UserEventsResponse> => {
		return await call(`${this.apiURL}/events`, http.POST, {
			properties: [
				'id',
				'user',
				'connection',
				'anonymousId',
				'category',
				'context',
				'event',
				'groupId',
				'messageId',
				'name',
				'properties',
				'receivedAt',
				'sentAt',
				'timestamp',
				'traits',
				'type',
				'userId',
			],
			filter: {
				logical: 'and',
				conditions: [
					{
						property: 'user',
						operator: 'is',
						values: [user],
					},
				],
			},
			order: 'timestamp',
			orderDesc: true,
			first: 0,
			limit: 10,
		});
	};

	traits = async (user: string): Promise<userTraitsResponse> => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/traits`, http.GET);
	};

	identities = async (user: string, first: number, limit: number): Promise<UserIdentitiesResponse> => {
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
	eventListeners: EventListeners;
	users: Users;

	constructor(origin: string, apiURL: string, workspaceID: number) {
		this.origin = origin;
		this.baseAPIURL = apiURL + '/workspaces';
		const url = this.baseAPIURL + `/${workspaceID}`;
		this.apiURL = url;
		this.connections = new Connections(url);
		this.eventListeners = new EventListeners(url);
		this.users = new Users(url);
	}

	list = async (): Promise<Workspace[]> => {
		return await call(`${this.baseAPIURL}`, http.GET);
	};

	create = async (
		name: string,
		privacyRegion: PrivacyRegion,
		userSchema: ObjectType,
		displayedProperties: DisplayedProperties,
		warehouseType: string,
		warehouseMode: WarehouseMode,
		warehouseSettings: WarehouseSettings,
	): Promise<CreateWorkspaceResponse> => {
		return await call(`${this.baseAPIURL}`, http.POST, {
			name: name,
			privacyRegion: privacyRegion,
			userSchema: userSchema,
			displayedProperties: displayedProperties,
			warehouse: {
				type: warehouseType,
				mode: warehouseMode,
				settings: warehouseSettings,
			},
		});
	};

	testCreation = async (
		name: string,
		privacyRegion: PrivacyRegion,
		userSchema: ObjectType,
		displayedProperties: DisplayedProperties,
		warehouseType: string,
		warehouseMode: WarehouseMode,
		warehouseSettings: WarehouseSettings,
	): Promise<void> => {
		return await call(`${this.baseAPIURL}/test`, http.POST, {
			name: name,
			privacyRegion: privacyRegion,
			userSchema: userSchema,
			displayedProperties: displayedProperties,
			warehouse: {
				type: warehouseType,
				mode: warehouseMode,
				settings: warehouseSettings,
			},
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

	createConnection = async (connection: ConnectionToAdd, oauthToken: string): Promise<number> => {
		return await call(`${this.apiURL}/connections`, http.POST, {
			connection,
			oauthToken,
		});
	};

	oauthToken = async (connector: string, oauthCode: string): Promise<string> => {
		const redirectURI = `${this.origin}${UI_BASE_PATH}oauth/authorize`;
		return await call(`${this.apiURL}/oauth-token`, http.POST, {
			connector,
			oauthCode,
			redirectURI,
		});
	};

	updateIdentityResolution = async (runOnBatchImport: boolean, identifiers: Identifiers): Promise<void> => {
		return await call(`${this.apiURL}/identity-resolution/settings`, http.PUT, {
			runOnBatchImport,
			identifiers,
		});
	};

	warehouse = async (): Promise<WarehouseResponse> => {
		return await call(`${this.apiURL}/warehouse/settings`, http.GET);
	};

	updateWarehouseMode = async (mode: WarehouseMode, cancelIncompatibleOperations: boolean): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/mode`, http.PUT, {
			mode,
			cancelIncompatibleOperations,
		});
	};

	testWarehouseUpdate = async (settings: any): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/can-change-settings`, http.POST, {
			settings,
		});
	};

	updateWarehouse = async (
		name: string,
		mode: WarehouseMode,
		settings: any,
		cancelIncompatibleOperations: boolean,
	): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/settings`, http.PUT, {
			name,
			mode,
			settings,
			cancelIncompatibleOperations,
		});
	};

	startIdentityResolution = async (): Promise<void> => {
		return await call(`${this.apiURL}/identity-resolutions`, http.POST);
	};

	updateUserSchema = async (schema: ObjectType, primarySources: PrimarySources, rePaths: RePaths): Promise<void> => {
		return await call(`${this.apiURL}/user-schema`, http.PUT, {
			schema,
			primarySources,
			rePaths,
		});
	};

	previewUserSchemaUpdate = async (
		schema: ObjectType,
		rePaths: RePaths,
	): Promise<PreviewUserSchemaUpdateResponse> => {
		return await call(`${this.apiURL}/change-user-schema-queries`, http.POST, {
			schema,
			rePaths,
		});
	};

	actionErrors = async (
		start: Date,
		end: Date | null,
		actions: number[],
		first: number,
		limit: number,
		step?: ActionStep,
	): Promise<ActionErrorsResponse> => {
		let actionsQueryString = '';
		for (let i = 0; i < actions.length; i++) {
			if (i > 0) {
				actionsQueryString += '&';
			}
			actionsQueryString += `actions=${encodeURIComponent(actions[i])}`;
		}
		const r: ActionErrorsResponse = await call(
			`${this.apiURL}/action-errors?start=${encodeURIComponent(start.toISOString())}${end ? `&end=${encodeURIComponent(end.toISOString())}` : ''}&${actionsQueryString}&first=${encodeURIComponent(first)}&limit=${encodeURIComponent(limit)}${step ? `&step=${encodeURIComponent(step)}` : ''}`,
			http.GET,
		);
		for (let i = 0; i < r.errors.length; i++) {
			r.errors[i].lastOccurred = new Date(r.errors[i].lastOccurred);
		}
		return r;
	};

	actionMetricsPerDate = async (start: Date, end: Date, actions: number[]): Promise<ActionMetrics> => {
		let actionsQueryString = '';
		for (let i = 0; i < actions.length; i++) {
			if (i > 0) {
				actionsQueryString += '&';
			}
			actionsQueryString += `actions=${encodeURIComponent(actions[i])}`;
		}
		const sd = start.toISOString().split('T')[0];
		const ed = end.toISOString().split('T')[0];
		const r = await call(
			`${this.apiURL}/action-metrics/dates?start=${encodeURIComponent(sd)}&end=${encodeURIComponent(ed)}&${actionsQueryString}`,
			http.GET,
		);
		r.start = new Date(r.start);
		r.end = new Date(r.end);
		return r;
	};

	actionMetricsPerDay = async (days: number, actions: number[]): Promise<ActionMetrics> => {
		let actionsQueryString = '';
		for (let i = 0; i < actions.length; i++) {
			if (i > 0) {
				actionsQueryString += '&';
			}
			actionsQueryString += `actions=${encodeURIComponent(actions[i])}`;
		}
		const r = await call(
			`${this.apiURL}/action-metrics/days?days=${encodeURIComponent(days)}&${actionsQueryString}`,
			http.GET,
		);
		r.start = new Date(r.start);
		r.end = new Date(r.end);
		return r;
	};

	actionMetricsPerHour = async (hours: number, actions: number[]): Promise<ActionMetrics> => {
		let actionsQueryString = '';
		for (let i = 0; i < actions.length; i++) {
			if (i > 0) {
				actionsQueryString += '&';
			}
			actionsQueryString += `actions=${encodeURIComponent(actions[i])}`;
		}
		const r = await call(
			`${this.apiURL}/action-metrics/hours?hours=${encodeURIComponent(hours)}&${actionsQueryString}`,
			http.GET,
		);
		r.start = new Date(r.start);
		r.end = new Date(r.end);
		return r;
	};

	actionMetricsPerMinute = async (minutes: number, actions: number[]): Promise<ActionMetrics> => {
		let actionsQueryString = '';
		for (let i = 0; i < actions.length; i++) {
			if (i > 0) {
				actionsQueryString += '&';
			}
			actionsQueryString += `actions=${encodeURIComponent(actions[i])}`;
		}
		const r = await call(
			`${this.apiURL}/action-metrics/minutes?minutes=${encodeURIComponent(minutes)}&${actionsQueryString}`,
			http.GET,
		);
		r.start = new Date(r.start);
		r.end = new Date(r.end);
		return r;
	};

	lastIdentityResolution = async (): Promise<LastIdentityResolution> => {
		return await call(`${this.apiURL}/identity-resolution/execution`, http.GET);
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
			connector,
			role,
			oauthToken,
		});
	};

	uiEvent = async (
		workspace: number,
		connector: string,
		event: string,
		settings: ConnectorSettings,
		role: ConnectionRole,
		oauthToken: string,
	): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/workspaces/${workspace}/ui-event`, http.POST, {
			connector,
			event,
			settings,
			role,
			oauthToken,
		});
	};
}

// TODO: review this for production.
if (typeof window !== 'undefined') {
	(window as any).API = API;
}

export default API;
