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
import { Connector, ConnectorDocumentation } from './types/connector';
import { WarehouseMode, WarehouseResponse, WarehouseSettings } from './types/warehouse';
import Workspace, {
	CreateWorkspaceResponse,
	UIPreferences,
	LatestIdentityResolution,
	LatestAlterUserSchema,
	PrimarySources,
} from './types/workspace';
import {
	AccessKeyResponse,
	ActionErrorsResponse,
	ActionSchemasResponse,
	AppUsersResponse,
	AbsolutePathResponse,
	ConnectionIdentitiesResponse,
	ConnectorSettings,
	ConnectorUIResponse,
	CreateAccessKeyResponse,
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
	PreviewAlterUserSchemaResponse,
	RePaths,
	RecordsResponse,
	SheetsResponse,
	TableSchemaResponse,
	TelemetryLevel,
	TransformDataResponse,
	TransformationLanguagesResponse,
	UserEventsResponse,
	UserIdentitiesResponse,
	authCodeURLResponse,
	userTraitsResponse,
} from './types/responses';
import { AccessKeyType } from './types/organization';

const API_BASE_PATH = '/api/v1';

class API {
	apiURL: string;
	workspaceID: number;
	workspaces: Workspaces;
	connectors: Connectors;

	constructor(origin: string, workspaceID: number) {
		const apiURL = origin + API_BASE_PATH;
		this.apiURL = apiURL;
		this.workspaceID = workspaceID;
		this.workspaces = new Workspaces(origin, apiURL, workspaceID);
		this.connectors = new Connectors(origin, apiURL);
	}

	installationID = async (): Promise<string> => {
		return await call(`${this.apiURL}/installation-id`, http.GET);
	};

	javaScriptSDKURL = async (): Promise<string> => {
		return await call(`${this.apiURL}/javascript-sdk-url`, http.GET);
	};

	login = async (email: string, password: string, isUnique?: boolean): Promise<[number, string]> => {
		return await call(`${this.apiURL}/members/login`, http.POST, this.workspaceID, {
			email,
			password,
			isUnique: isUnique == null ? false : isUnique,
		});
	};

	logout = async (): Promise<void> => {
		return await call(`${this.apiURL}/members/logout`, http.POST, this.workspaceID);
	};

	sendMemberPasswordReset = async (email: string): Promise<void> => {
		return await call(`${this.apiURL}/members/reset-password`, http.PUT, this.workspaceID, { email });
	};

	validateMemberPasswordResetToken = async (token: string): Promise<void> => {
		return await call(`${this.apiURL}/members/reset-password/${token}`, http.GET, this.workspaceID);
	};

	changeMemberPasswordByToken = async (token: string, password: string): Promise<void> => {
		return await call(`${this.apiURL}/members/reset-password/${token}`, http.PUT, this.workspaceID, {
			password,
		});
	};

	skipMemberEmailVerification = async (): Promise<boolean> => {
		return await call(`${this.apiURL}/skip-member-email-verification`, http.GET);
	};

	telemetryLevel = async (): Promise<TelemetryLevel> => {
		return await call(`${this.apiURL}/telemetry/level`, http.GET);
	};

	eventsSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/events/schema`, http.GET, this.workspaceID);
	};

	eventURL = async (): Promise<string> => {
		return await call(`${this.apiURL}/event-url`, http.GET);
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
			this.workspaceID,
			{
				expression,
				properties,
				type,
			},
			{ signal },
		);
	};

	expressionsProperties = async (expressions: ExpressionToBeExtracted[], schema: Type): Promise<string[]> => {
		return await call(`${this.apiURL}/expressions-properties`, http.POST, this.workspaceID, {
			expressions,
			schema,
		});
	};

	transformationLanguages = async (): Promise<TransformationLanguagesResponse> => {
		return await call(`${this.apiURL}/system/transformations/languages`, http.GET, this.workspaceID);
	};

	transformData = async (
		data: Record<string, any>,
		inSchema: ObjectType,
		outSchema: ObjectType,
		transformation: Transformation,
		purpose: TransformationPurpose,
	): Promise<TransformDataResponse> => {
		return await call(`${this.apiURL}/transformations`, http.POST, this.workspaceID, {
			data,
			inSchema,
			outSchema,
			transformation,
			purpose,
		});
	};

	members = async (): Promise<Member[]> => {
		return await call(`${this.apiURL}/members`, http.GET, this.workspaceID);
	};

	inviteMember = async (email: string): Promise<void> => {
		return await call(`${this.apiURL}/members/invitations`, http.POST, this.workspaceID, {
			email,
		});
	};

	memberInvitation = async (token: string): Promise<MemberInvitationResponse> => {
		return await call(`${this.apiURL}/members/invitations/${token}`, http.GET, this.workspaceID);
	};

	acceptInvitation = async (token: string, name: string, password: string): Promise<void> => {
		return await call(`${this.apiURL}/members/invitations/${token}`, http.PUT, this.workspaceID, {
			name,
			password,
		});
	};

	member = async (): Promise<Member> => {
		return await call(`${this.apiURL}/members/current`, http.GET, this.workspaceID);
	};

	updateMember = async (memberToSet: MemberToSet): Promise<void> => {
		return await call(`${this.apiURL}/members/current`, http.PUT, this.workspaceID, {
			memberToSet,
		});
	};

	addMember = async (memberToSet: MemberToSet): Promise<void> => {
		return await call(`${this.apiURL}/members`, http.POST, this.workspaceID, {
			memberToSet,
		});
	};

	deleteMember = async (member: number): Promise<void> => {
		return await call(`${this.apiURL}/members/${member}`, http.DELETE, this.workspaceID);
	};

	keys = async (): Promise<AccessKeyResponse> => {
		return await call(`${this.apiURL}/keys`, http.GET, this.workspaceID);
	};

	createAccessKey = async (
		name: string,
		workspace: number | null,
		type: AccessKeyType,
	): Promise<CreateAccessKeyResponse> => {
		return await call(`${this.apiURL}/keys`, http.POST, this.workspaceID, {
			name,
			workspace,
			type,
		});
	};

	updateAccessKey = async (key: number, name: string): Promise<void> => {
		return await call(`${this.apiURL}/keys/${encodeURIComponent(key)}`, http.PUT, this.workspaceID, {
			name,
		});
	};

	deleteAccessKey = async (key: number): Promise<void> => {
		return await call(`${this.apiURL}/keys/${encodeURIComponent(key)}`, http.DELETE, this.workspaceID);
	};
}

class Connections {
	apiURL: string;
	workspaceID: number;

	constructor(url: string, workspaceID: number) {
		this.apiURL = url;
		this.workspaceID = workspaceID;
	}

	find = async (): Promise<Connection[]> => {
		const res = await call(`${this.apiURL}/connections`, http.GET, this.workspaceID);
		return res.connections as Connection[];
	};

	get = async (connection: number): Promise<Connection> => {
		const c = await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}`,
			http.GET,
			this.workspaceID,
		);
		// Transform the 'actions.transformation' field to match the expected format used throughout the rest of the codebase.
		for (let action of c.actions) {
			if (action.transformation == null) {
				action.transformation = {
					mapping: null,
					function: null,
				};
			}
		}
		return c as Connection;
	};

	update = async (id: number, connection: ConnectionToSet) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(id)}`,
			http.PUT,
			this.workspaceID,
			connection,
		);
	};

	delete = async (connection: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}`,
			http.DELETE,
			this.workspaceID,
		);
	};

	execution = async (id: number): Promise<Execution> => {
		return await call(`${this.apiURL}/actions/executions/${id}`, http.GET, this.workspaceID);
	};

	executions = async (): Promise<Execution[]> => {
		return await call(`${this.apiURL}/actions/executions`, http.GET, this.workspaceID);
	};

	identities = async (connection: number, first: number, limit: number): Promise<ConnectionIdentitiesResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/identities?first=${first}&limit=${limit}`,
			http.GET,
			this.workspaceID,
		);
	};

	execQuery = async (connection: number, query: string, limit: number): Promise<ExecQueryResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/query`,
			http.POST,
			this.workspaceID,
			{
				query: query,
				limit: limit,
			},
		);
	};

	records = async (
		connection: number,
		path: string,
		format: string,
		sheet: string | null,
		compression: string,
		formatSettings: ConnectorSettings | null,
		limit: number,
	): Promise<RecordsResponse> => {
		let params = [];
		params.push(['format', format]);
		if (sheet !== null) {
			params.push(['sheet', sheet]);
		}
		params.push(['compression', compression]);
		if (formatSettings != null) {
			params.push(['formatSettings', JSON.stringify(formatSettings)]);
		}
		params.push(['limit', limit]);
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/files/${encodeURIComponent(path)}` +
				queryString(params),
			http.GET,
			this.workspaceID,
		);
	};

	sheets = async (
		connection: number,
		path: string,
		format: string,
		compression: string,
		formatSettings: ConnectorSettings | null,
	): Promise<SheetsResponse> => {
		let params = [];
		params.push(['format', format]);
		params.push(['compression', compression]);
		if (formatSettings != null) {
			params.push(['formatSettings', JSON.stringify(formatSettings)]);
		}
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/files/${encodeURIComponent(path)}/sheets` +
				queryString(params),
			http.GET,
			this.workspaceID,
		);
	};

	ui = async (connection: number): Promise<ConnectorUIResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui`,
			http.GET,
			this.workspaceID,
		);
	};

	uiEvent = async (connection: number, event: string, settings: ConnectorSettings): Promise<ConnectorUIResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/ui-event`,
			http.POST,
			this.workspaceID,
			{
				event,
				settings,
			},
		);
	};

	eventWriteKeys = async (connection: number): Promise<string[]> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/event-write-keys`,
			http.GET,
			this.workspaceID,
		);
	};

	createEventWriteKey = async (connection: number): Promise<string> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/event-write-keys`,
			http.POST,
			this.workspaceID,
		);
	};

	deleteEventWriteKey = async (connection: number, key: string): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/event-write-keys/${encodeURIComponent(key)}`,
			http.DELETE,
			this.workspaceID,
		);
	};

	// TODO(Gianluca): this method is deprecated. See the issue
	// https://github.com/meergo/meergo/issues/1265.
	actionTypes = async (connection: number) => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/action-types`,
			http.GET,
			this.workspaceID,
		);
	};

	// TODO(Gianluca): this method is deprecated. See the issue
	// https://github.com/meergo/meergo/issues/1266.
	actionSchemas = async (
		connection: number,
		target: ActionTarget,
		eventType: string,
	): Promise<ActionSchemasResponse> => {
		if (eventType != null) {
			return await call(
				`${this.apiURL}/connections/${encodeURIComponent(
					connection,
				)}/actions/schemas/Events?type=${encodeURIComponent(eventType)}`,
				http.GET,
				this.workspaceID,
			);
		} else {
			return await call(
				`${this.apiURL}/connections/${encodeURIComponent(connection)}/actions/schemas/${encodeURIComponent(
					target,
				)}`,
				http.GET,
				this.workspaceID,
			);
		}
	};

	createAction = async (
		connection: number,
		target: ActionTarget,
		eventType: string,
		action: ActionToSet,
	): Promise<number> => {
		// Transform the 'actions.transformation' field to match the API expected format.
		if ('transformation' in action) {
			if (action.transformation.mapping == null && action.transformation.function == null) {
				action.transformation = null;
			}
		}
		return await call(`${this.apiURL}/actions`, http.POST, this.workspaceID, {
			connection,
			target,
			eventType,
			...action,
		});
	};

	updateAction = async (id: number, action: ActionToSet): Promise<void> => {
		// Transform the 'actions.transformation' field to match the API expected format.
		if ('transformation' in action) {
			if (action.transformation.mapping == null && action.transformation.function == null) {
				action.transformation = null;
			}
		}
		return await call(`${this.apiURL}/actions/${encodeURIComponent(id)}`, http.PUT, this.workspaceID, action);
	};

	deleteAction = async (action: number): Promise<void> => {
		return await call(`${this.apiURL}/actions/${encodeURIComponent(action)}`, http.DELETE, this.workspaceID);
	};

	setActionStatus = async (action: number, enabled: boolean): Promise<void> => {
		return await call(`${this.apiURL}/actions/${encodeURIComponent(action)}/status`, http.PUT, this.workspaceID, {
			enabled,
		});
	};

	setActionSchedulePeriod = async (action: number, period: SchedulePeriod | null): Promise<void> => {
		return await call(`${this.apiURL}/actions/${encodeURIComponent(action)}/schedule`, http.PUT, this.workspaceID, {
			period,
		});
	};

	executeAction = async (action: number): Promise<number> => {
		const res = await call(
			`${this.apiURL}/actions/${encodeURIComponent(action)}/exec`,
			http.POST,
			this.workspaceID,
			{
				incremental: null,
			},
		);
		return res.id as number;
	};

	actionUiEvent = async (
		action: number,
		event: string,
		formatSettings: ConnectorSettings,
	): Promise<ConnectorUIResponse> => {
		return await call(
			`${this.apiURL}/actions/${encodeURIComponent(action)}/ui-event`,
			http.POST,
			this.workspaceID,
			{
				event,
				formatSettings,
			},
		);
	};

	absolutePath = async (storageConnection: number, path: string): Promise<AbsolutePathResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(storageConnection)}/files/${encodeURIComponent(
				path,
			)}/absolute`,
			http.GET,
			this.workspaceID,
		);
	};

	tableSchema = async (connection: number, tableName: string): Promise<TableSchemaResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/tables/${encodeURIComponent(tableName)}`,
			http.GET,
			this.workspaceID,
		);
	};

	appUsers = async (connection: number, schema: ObjectType, cursor?: string): Promise<AppUsersResponse> => {
		let params = [];
		params.push(['schema', JSON.stringify(schema)]);
		if (cursor !== undefined) {
			params.push(['cursor', cursor]);
		}
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/users` + queryString(params),
			http.GET,
			this.workspaceID,
		);
	};

	previewSendEvent = async (
		connection: number,
		type: string,
		event: Event,
		outSchema: ObjectType,
		transformation?: Transformation,
	): Promise<PreviewSendEventResponse> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(connection)}/preview-send-event`,
			http.POST,
			this.workspaceID,
			{
				type,
				event,
				outSchema,
				transformation,
			},
		);
	};

	unlinkConnection = async (src: number, dst: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(src)}/links/${encodeURIComponent(dst)}`,
			http.DELETE,
			this.workspaceID,
		);
	};

	linkConnection = async (src: number, dst: number): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${encodeURIComponent(src)}/links/${encodeURIComponent(dst)}`,
			http.POST,
			this.workspaceID,
		);
	};
}

class EventListeners {
	apiURL: string;
	workspaceID: number;

	constructor(url: string, workspaceID: number) {
		this.apiURL = url;
		this.workspaceID = workspaceID;
	}

	create = async (size: number, filter: Filter): Promise<CreateEventListenerResponse> => {
		return await call(`${this.apiURL}/events/listeners`, http.POST, this.workspaceID, {
			size,
			filter,
		});
	};

	delete = async (eventListener: string): Promise<void> => {
		return await call(
			`${this.apiURL}/events/listeners/${encodeURIComponent(eventListener)}`,
			http.DELETE,
			this.workspaceID,
		);
	};

	events = async (eventListener: string): Promise<EventListenerEventsResponse> => {
		return await call(
			`${this.apiURL}/events/listeners/${encodeURIComponent(eventListener)}`,
			http.GET,
			this.workspaceID,
		);
	};
}

class Users {
	apiURL: string;
	workspaceID: number;

	constructor(url: string, workspaceID: number) {
		this.apiURL = url;
		this.workspaceID = workspaceID;
	}

	find = async (
		properties: string[],
		filter: Filter | null,
		order: string,
		orderDesc: boolean,
		first: number,
		limit: number,
	): Promise<FindUsersResponse> => {
		let params = [];
		properties.forEach(function (property) {
			params.push(['properties', property]);
		});
		if (filter != null) {
			params.push(['filter', JSON.stringify(filter)]);
		}
		params.push(['order', order]);
		params.push(['orderDesc', orderDesc]);
		params.push(['first', first]);
		params.push(['limit', limit]);
		return await call(`${this.apiURL}/users` + queryString(params), http.GET, this.workspaceID);
	};

	events = async (user: string): Promise<UserEventsResponse> => {
		let params = [];
		let properties = [
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
		];
		properties.forEach(function (property) {
			params.push(['properties', property]);
		});
		let filter = {
			logical: 'and',
			conditions: [
				{
					property: 'user',
					operator: 'is',
					values: [user],
				},
			],
		};
		params.push(['filter', JSON.stringify(filter)]);
		params.push(['order', 'timestamp']);
		params.push(['orderDesc', true]);
		params.push(['first', 0]);
		params.push(['limit', 10]);
		return await call(`${this.apiURL}/events` + queryString(params), http.GET, this.workspaceID);
	};

	traits = async (user: string): Promise<userTraitsResponse> => {
		return await call(`${this.apiURL}/users/${encodeURIComponent(user)}/traits`, http.GET, this.workspaceID);
	};

	identities = async (user: string, first: number, limit: number): Promise<UserIdentitiesResponse> => {
		return await call(
			`${this.apiURL}/users/${encodeURIComponent(user)}/identities?first=${first}&limit=${limit}`,
			http.GET,
			this.workspaceID,
		);
	};
}

class Workspaces {
	origin: string;
	apiURL: string;
	workspaceID: number;
	connections: Connections;
	eventListeners: EventListeners;
	users: Users;

	constructor(origin: string, apiURL: string, workspaceID: number) {
		this.origin = origin;
		this.apiURL = apiURL;
		this.workspaceID = workspaceID;
		this.connections = new Connections(apiURL, workspaceID);
		this.eventListeners = new EventListeners(apiURL, workspaceID);
		this.users = new Users(apiURL, workspaceID);
	}

	list = async (): Promise<Workspace[]> => {
		const res = await call(`${this.apiURL}/workspaces`, http.GET, this.workspaceID);
		return res.workspaces as Workspace[];
	};

	create = async (
		name: string,
		userSchema: ObjectType,
		warehouseType: string,
		warehouseMode: WarehouseMode,
		warehouseSettings: WarehouseSettings,
		warehouseMCPSettings: WarehouseSettings | null,
		uiPreferences: UIPreferences,
	): Promise<CreateWorkspaceResponse> => {
		return await call(`${this.apiURL}/workspaces`, http.POST, this.workspaceID, {
			name: name,
			userSchema: userSchema,
			warehouse: {
				type: warehouseType,
				mode: warehouseMode,
				settings: warehouseSettings,
				mcpSettings: warehouseMCPSettings,
			},
			uiPreferences: uiPreferences,
		});
	};

	testCreation = async (
		name: string,
		userSchema: ObjectType,
		warehouseType: string,
		warehouseMode: WarehouseMode,
		warehouseSettings: WarehouseSettings,
		mcpWarehouseSettings: WarehouseSettings | null,
		uiPreferences: UIPreferences,
	): Promise<void> => {
		return await call(`${this.apiURL}/workspaces/test`, http.POST, this.workspaceID, {
			name: name,
			userSchema: userSchema,
			warehouse: {
				type: warehouseType,
				mode: warehouseMode,
				settings: warehouseSettings,
				mcpSettings: mcpWarehouseSettings,
			},
			uiPreferences: uiPreferences,
		});
	};

	get = async (): Promise<Workspace> => {
		return await call(`${this.apiURL}/workspaces/current`, http.GET, this.workspaceID);
	};

	update = async (name: string, uiPreferences: UIPreferences): Promise<void> => {
		return await call(`${this.apiURL}/workspaces/current`, http.PUT, this.workspaceID, {
			name,
			uiPreferences,
		});
	};

	delete = async (): Promise<void> => {
		return await call(`${this.apiURL}/workspaces/current`, http.DELETE, this.workspaceID);
	};

	userSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/users/schema`, http.GET, this.workspaceID);
	};

	userPropertiesSuitableAsIdentifiers = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/users/schema/suitable-as-identifiers`, http.GET, this.workspaceID);
	};

	createConnection = async (connection: ConnectionToAdd, authToken: string): Promise<number> => {
		return await call(`${this.apiURL}/connections`, http.POST, this.workspaceID, {
			...connection,
			authToken: authToken,
		});
	};

	authToken = async (connector: string, authCode: string): Promise<string> => {
		const redirectURI = `${this.origin}${UI_BASE_PATH}oauth/authorize`;
		return await call(
			`${this.apiURL}/connections/auth-token?connector=${encodeURIComponent(connector)}&redirectURI=${encodeURIComponent(redirectURI)}&authCode=${encodeURIComponent(authCode)}`,
			http.GET,
			this.workspaceID,
		);
	};

	updateIdentityResolution = async (runOnBatchImport: boolean, identifiers: Identifiers): Promise<void> => {
		return await call(`${this.apiURL}/identity-resolution/settings`, http.PUT, this.workspaceID, {
			runOnBatchImport,
			identifiers,
		});
	};

	warehouse = async (): Promise<WarehouseResponse> => {
		return await call(`${this.apiURL}/warehouse`, http.GET, this.workspaceID);
	};

	updateWarehouseMode = async (mode: WarehouseMode, cancelIncompatibleOperations: boolean): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/mode`, http.PUT, this.workspaceID, {
			mode,
			cancelIncompatibleOperations,
		});
	};

	testWarehouseUpdate = async (settings: any, mcpSettings: any): Promise<void> => {
		return await call(`${this.apiURL}/warehouse/test`, http.PUT, this.workspaceID, {
			settings,
			mcpSettings,
		});
	};

	updateWarehouse = async (
		name: string,
		mode: WarehouseMode,
		settings: any,
		mcpSettings: any | null,
		cancelIncompatibleOperations: boolean,
	): Promise<void> => {
		return await call(`${this.apiURL}/warehouse`, http.PUT, this.workspaceID, {
			name,
			mode,
			settings,
			mcpSettings,
			cancelIncompatibleOperations,
		});
	};

	startIdentityResolution = async (): Promise<void> => {
		return await call(`${this.apiURL}/identity-resolution/start`, http.POST, this.workspaceID);
	};

	alterUserSchema = async (schema: ObjectType, primarySources: PrimarySources, rePaths: RePaths): Promise<void> => {
		return await call(`${this.apiURL}/users/schema`, http.PUT, this.workspaceID, {
			schema,
			primarySources,
			rePaths,
		});
	};

	previewAlterUserSchema = async (schema: ObjectType, rePaths: RePaths): Promise<PreviewAlterUserSchemaResponse> => {
		return await call(`${this.apiURL}/users/schema/preview`, http.PUT, this.workspaceID, {
			schema,
			rePaths,
		});
	};

	actionErrors = async (
		start: Date,
		end: Date,
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
			`${this.apiURL}/actions/errors/${encodeURIComponent(start.toISOString())}/${encodeURIComponent(end.toISOString())}?${actionsQueryString}&first=${encodeURIComponent(first)}&limit=${encodeURIComponent(limit)}${step ? `&step=${encodeURIComponent(step)}` : ''}`,
			http.GET,
			this.workspaceID,
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
			`${this.apiURL}/actions/metrics/dates/${encodeURIComponent(sd)}/${encodeURIComponent(ed)}?${actionsQueryString}`,
			http.GET,
			this.workspaceID,
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
			`${this.apiURL}/actions/metrics/days/${encodeURIComponent(days)}?${actionsQueryString}`,
			http.GET,
			this.workspaceID,
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
			`${this.apiURL}/actions/metrics/hours/${encodeURIComponent(hours)}?${actionsQueryString}`,
			http.GET,
			this.workspaceID,
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
			`${this.apiURL}/actions/metrics/minutes/${encodeURIComponent(minutes)}?${actionsQueryString}`,
			http.GET,
			this.workspaceID,
		);
		r.start = new Date(r.start);
		r.end = new Date(r.end);
		return r;
	};

	latestIdentityResolution = async (): Promise<LatestIdentityResolution> => {
		return await call(`${this.apiURL}/identity-resolution/latest`, http.GET, this.workspaceID);
	};

	LatestAlterUserSchema = async (): Promise<LatestAlterUserSchema> => {
		return await call(`${this.apiURL}/users/schema/latest-alter`, http.GET, this.workspaceID);
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
			`${this.apiURL}/connections/auth-url?connector=${encodeURIComponent(connector)}&role=${role}&redirectURI=${encodeURIComponent(redirectURI)}`,
			http.GET,
		);
	};

	find = async (): Promise<Connector[]> => {
		const res = await call(`${this.apiURL}/connectors`, http.GET);
		return res.connectors as Connector[];
	};

	get = async (connector: string): Promise<Connector> => {
		return await call(`${this.apiURL}/connectors/${connector}`, http.GET);
	};

	connectorDocumentation = async (connector: string): Promise<ConnectorDocumentation> => {
		return await call(`${this.apiURL}/connectors/${connector}/documentation`, http.GET);
	};

	ui = async (
		workspace: number,
		connector: string,
		role: ConnectionRole,
		authToken: string,
	): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/ui`, http.POST, workspace, {
			connector,
			role,
			authToken,
		});
	};

	uiEvent = async (
		workspace: number,
		connector: string,
		event: string,
		settings: ConnectorSettings,
		role: ConnectionRole,
		authToken: string,
	): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/ui-event`, http.POST, workspace, {
			connector,
			event,
			settings,
			role,
			authToken,
		});
	};
}

// TODO: review this for production.
if (typeof window !== 'undefined') {
	(window as any).API = API;
}

function queryString(parameters: Array<[string, any]>) {
	if (parameters.length == 0) {
		return '';
	}
	const parts: string[] = [];
	parameters.forEach(([key, value]) => {
		const encodedKey = encodeURIComponent(key);
		const encodedValue = encodeURIComponent(String(value));
		parts.push(`${encodedKey}=${encodedValue}`);
	});
	return '?' + parts.join('&');
}

export default API;
export { API_BASE_PATH };
