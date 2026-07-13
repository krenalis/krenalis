import call from './call';
import * as http from './http';
import Type, { Property, ObjectType, Role } from './types/types';
import { Connection, ConnectionRole, ConnectionToAdd, ConnectionToSet } from './types/connection';
import { Identifiers } from './types/identifiers';
import {
	PipelineTarget,
	SchedulePeriod,
	PipelineToSet,
	ExpressionToBeExtracted,
	Transformation,
	TransformationPurpose,
	PipelineStep,
	PipelineMetrics,
	Filter,
} from './types/pipeline';
import { Connector, ConnectorDocumentation } from './types/connector';
import { WarehouseMode, WarehouseResponse, WarehouseSettings } from './types/warehouse';
import Workspace, {
	CreateWorkspaceResponse,
	UIPreferences,
	LatestIdentityResolution,
	LatestAlterProfileSchema,
	PrimarySources,
} from './types/workspace';
import {
	AccessKeyResponse,
	PipelineErrorsResponse,
	PipelineSchemasResponse,
	ApplicationUsersResponse,
	AbsolutePathResponse,
	ConnectionIdentitiesResponse,
	ConnectorSettings,
	ConnectorUIResponse,
	CreateAccessKeyResponse,
	CreateConnectionResponse,
	CreateEventListenerResponse,
	CreateEventWriteKeyResponse,
	CreatePipelineResponse,
	Event,
	EventListenerEventsResponse,
	ExecQueryResponse,
	PipelineRun,
	FindProfilesResponse,
	Member,
	MemberInvitationResponse,
	MemberToSet,
	PreviewSendEventResponse,
	PreviewAlterProfileSchemaResponse,
	RePaths,
	RecordsResponse,
	SheetsResponse,
	TableSchemaResponse,
	TransformDataResponse,
	TransformationLanguagesResponse,
	ProfileEventsResponse,
	IdentitiesResponse,
	authURLResponse,
	authTokenResponse,
	profileAttributesResponse,
	PublicMetadata,
} from './types/responses';
import { AccessKeyType } from './types/organization';

const API_BASE_PATH = '/v1';

type PipelineMetricsFilter = {
	pipelines?: string[];
	workspaces?: string[];
	connections?: string[];
	target?: PipelineTarget;
};

class API {
	apiURL: string;
	workspaceID: string;
	workspaces: Workspaces;
	connectors: Connectors;

	constructor(origin: string, workspaceID: string) {
		const apiURL = origin + API_BASE_PATH;
		this.apiURL = apiURL;
		this.workspaceID = workspaceID;
		this.workspaces = new Workspaces(origin, apiURL, workspaceID);
		this.connectors = new Connectors(origin, apiURL);
	}

	login = async (email: string, password: string, isUnique?: boolean): Promise<[string, string]> => {
		return await call(`${this.apiURL}/members/login`, http.POST, this.workspaceID, {
			email,
			password,
			isUnique: isUnique == null ? false : isUnique,
		});
	};

	loginWithWorkOS = async (accessToken: string): Promise<void> => {
		return await call(`${this.apiURL}/members/login`, http.POST, this.workspaceID, {
			accessToken,
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

	eventsSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/events/schema`, http.GET, this.workspaceID);
	};

	publicMetadata = async (): Promise<PublicMetadata> => {
		return await call(`${this.apiURL}/public/metadata`, http.GET);
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

	deleteMember = async (member: string): Promise<void> => {
		return await call(`${this.apiURL}/members/${member}`, http.DELETE, this.workspaceID);
	};

	keys = async (): Promise<AccessKeyResponse> => {
		return await call(`${this.apiURL}/keys`, http.GET, this.workspaceID);
	};

	createAccessKey = async (
		name: string,
		workspace: string | null,
		type: AccessKeyType,
	): Promise<CreateAccessKeyResponse> => {
		return await call(`${this.apiURL}/keys`, http.POST, this.workspaceID, {
			name,
			workspace,
			type,
		});
	};

	updateAccessKey = async (key: string, name: string): Promise<void> => {
		return await call(`${this.apiURL}/keys/${key}`, http.PUT, this.workspaceID, {
			name,
		});
	};

	deleteAccessKey = async (key: string): Promise<void> => {
		return await call(`${this.apiURL}/keys/${key}`, http.DELETE, this.workspaceID);
	};
}

class Connections {
	apiURL: string;
	workspaceID: string;

	constructor(url: string, workspaceID: string) {
		this.apiURL = url;
		this.workspaceID = workspaceID;
	}

	find = async (): Promise<Connection[]> => {
		const res = await call(`${this.apiURL}/connections`, http.GET, this.workspaceID);
		for (let c of res.connections) {
			if (!('linkedConnections' in c)) {
				c.linkedConnections = null;
			}
		}
		return res.connections as Connection[];
	};

	get = async (connection: string): Promise<Connection> => {
		const c = await call(`${this.apiURL}/connections/${connection}`, http.GET, this.workspaceID);
		if (!('linkedConnections' in c)) {
			c.linkedConnections = null;
		}
		return c as Connection;
	};

	update = async (id: string, connection: ConnectionToSet) => {
		return await call(`${this.apiURL}/connections/${id}`, http.PUT, this.workspaceID, connection);
	};

	delete = async (connection: string): Promise<void> => {
		return await call(`${this.apiURL}/connections/${connection}`, http.DELETE, this.workspaceID);
	};

	run = async (id: string): Promise<PipelineRun> => {
		return await call(`${this.apiURL}/pipelines/runs/${id}`, http.GET, this.workspaceID);
	};

	runs = async (): Promise<PipelineRun[]> => {
		return await call(`${this.apiURL}/pipelines/runs`, http.GET, this.workspaceID);
	};

	identities = async (connection: string, first: number, limit: number): Promise<ConnectionIdentitiesResponse> => {
		return await call(
			`${this.apiURL}/connections/${connection}/identities?first=${first}&limit=${limit}`,
			http.GET,
			this.workspaceID,
		);
	};

	execQuery = async (connection: string, query: string, limit: number): Promise<ExecQueryResponse> => {
		return await call(`${this.apiURL}/connections/${connection}/query`, http.POST, this.workspaceID, {
			query: query,
			limit: limit,
		});
	};

	records = async (
		connection: string,
		path: string,
		format: string,
		sheet: string | null,
		compression: string,
		formatSettings: ConnectorSettings | null,
		limit: number,
	): Promise<RecordsResponse> => {
		let params = [];
		params.push(['path', path]);
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
			`${this.apiURL}/connections/${connection}/files` + queryString(params),
			http.GET,
			this.workspaceID,
		);
	};

	sheets = async (
		connection: string,
		path: string,
		format: string,
		compression: string,
		formatSettings: ConnectorSettings | null,
	): Promise<SheetsResponse> => {
		let params = [];
		params.push(['path', path]);
		params.push(['format', format]);
		params.push(['compression', compression]);
		if (formatSettings != null) {
			params.push(['formatSettings', JSON.stringify(formatSettings)]);
		}
		return await call(
			`${this.apiURL}/connections/${connection}/files/sheets` + queryString(params),
			http.GET,
			this.workspaceID,
		);
	};

	ui = async (connection: string): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/connections/${connection}/ui`, http.GET, this.workspaceID);
	};

	uiEvent = async (connection: string, event: string, settings: ConnectorSettings): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/connections/${connection}/ui-event`, http.POST, this.workspaceID, {
			event,
			settings,
		});
	};

	eventWriteKeys = async (connection: string): Promise<string[]> => {
		const res = await call(`${this.apiURL}/connections/${connection}/event-write-keys`, http.GET, this.workspaceID);
		return res.keys;
	};

	createEventWriteKey = async (connection: string): Promise<string> => {
		const res: CreateEventWriteKeyResponse = await call(
			`${this.apiURL}/connections/${connection}/event-write-keys`,
			http.POST,
			this.workspaceID,
		);
		return res.key;
	};

	deleteEventWriteKey = async (connection: string, key: string): Promise<void> => {
		return await call(
			`${this.apiURL}/connections/${connection}/event-write-keys/${key}`,
			http.DELETE,
			this.workspaceID,
		);
	};

	// TODO(Gianluca): this method is deprecated. See the issue
	// https://github.com/krenalis/krenalis/issues/1265.
	pipelineTypes = async (connection: string) => {
		return await call(`${this.apiURL}/connections/${connection}/pipeline-types`, http.GET, this.workspaceID);
	};

	// TODO(Gianluca): this method is deprecated. See the issue
	// https://github.com/krenalis/krenalis/issues/1266.
	pipelineSchemas = async (
		connection: string,
		target: PipelineTarget,
		eventType: string,
	): Promise<PipelineSchemasResponse> => {
		if (eventType != null) {
			return await call(
				`${this.apiURL}/connections/${connection}/pipelines/schemas/Events?type=${encodeURIComponent(eventType)}`,
				http.GET,
				this.workspaceID,
			);
		} else {
			return await call(
				`${this.apiURL}/connections/${connection}/pipelines/schemas/${target}`,
				http.GET,
				this.workspaceID,
			);
		}
	};

	createPipeline = async (
		connection: string,
		target: PipelineTarget,
		eventType: string,
		pipeline: PipelineToSet,
	): Promise<string> => {
		const res: CreatePipelineResponse = await call(`${this.apiURL}/pipelines`, http.POST, this.workspaceID, {
			connection,
			target,
			eventType,
			...pipeline,
		});
		return res.id;
	};

	updatePipeline = async (id: string, pipeline: PipelineToSet): Promise<void> => {
		return await call(`${this.apiURL}/pipelines/${id}`, http.PUT, this.workspaceID, pipeline);
	};

	deletePipeline = async (pipeline: string): Promise<void> => {
		return await call(`${this.apiURL}/pipelines/${pipeline}`, http.DELETE, this.workspaceID);
	};

	setPipelineStatus = async (pipeline: string, enabled: boolean): Promise<void> => {
		return await call(`${this.apiURL}/pipelines/${pipeline}/status`, http.PUT, this.workspaceID, {
			enabled,
		});
	};

	setPipelineSchedulePeriod = async (pipeline: string, period: SchedulePeriod | null): Promise<void> => {
		return await call(`${this.apiURL}/pipelines/${pipeline}/schedule`, http.PUT, this.workspaceID, {
			period,
		});
	};

	runPipeline = async (pipeline: string): Promise<string> => {
		const res = await call(`${this.apiURL}/pipelines/${pipeline}/runs`, http.POST, this.workspaceID, {
			incremental: null,
		});
		return res.id as string;
	};

	pipelineUiEvent = async (
		pipeline: string,
		event: string,
		formatSettings: ConnectorSettings,
	): Promise<ConnectorUIResponse> => {
		return await call(`${this.apiURL}/pipelines/${pipeline}/ui-event`, http.POST, this.workspaceID, {
			event,
			formatSettings,
		});
	};

	absolutePath = async (storageConnection: string, path: string): Promise<AbsolutePathResponse> => {
		return await call(
			`${this.apiURL}/connections/${storageConnection}/files/absolute?path=${encodeURIComponent(path)}`,
			http.GET,
			this.workspaceID,
		);
	};

	tableSchema = async (connection: string, tableName: string): Promise<TableSchemaResponse> => {
		return await call(
			`${this.apiURL}/connections/${connection}/tables?name=${encodeURIComponent(tableName)}`,
			http.GET,
			this.workspaceID,
		);
	};

	apiUsers = async (
		connection: string,
		schema: ObjectType,
		filter: Filter | null,
		cursor?: string,
	): Promise<ApplicationUsersResponse> => {
		let params = [];
		params.push(['schema', JSON.stringify(schema)]);
		if (filter != null) {
			params.push(['filter', JSON.stringify(filter)]);
		}
		if (cursor !== undefined) {
			params.push(['cursor', cursor]);
		}
		return await call(
			`${this.apiURL}/connections/${connection}/users` + queryString(params),
			http.GET,
			this.workspaceID,
		);
	};

	previewSendEvent = async (
		connection: string,
		type: string,
		event: Event,
		outSchema: ObjectType,
		transformation?: Transformation,
	): Promise<PreviewSendEventResponse> => {
		return await call(`${this.apiURL}/connections/${connection}/preview-send-event`, http.POST, this.workspaceID, {
			type,
			event,
			outSchema,
			transformation,
		});
	};

	unlinkConnection = async (src: string, dst: string): Promise<void> => {
		return await call(`${this.apiURL}/connections/${src}/links/${dst}`, http.DELETE, this.workspaceID);
	};

	linkConnection = async (src: string, dst: string): Promise<void> => {
		return await call(`${this.apiURL}/connections/${src}/links/${dst}`, http.POST, this.workspaceID);
	};
}

class EventListeners {
	apiURL: string;
	workspaceID: string;

	constructor(url: string, workspaceID: string) {
		this.apiURL = url;
		this.workspaceID = workspaceID;
	}

	create = async (
		connection: string | null,
		size: number | null,
		filter: Filter | null,
	): Promise<CreateEventListenerResponse> => {
		return await call(`${this.apiURL}/events/listeners`, http.POST, this.workspaceID, {
			connection,
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

class Profiles {
	apiURL: string;
	workspaceID: string;

	constructor(url: string, workspaceID: string) {
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
	): Promise<FindProfilesResponse> => {
		let params = [];
		params.push(['properties', properties.join(',')]);
		if (filter != null) {
			params.push(['filter', JSON.stringify(filter)]);
		}
		params.push(['order', order]);
		params.push(['orderDesc', orderDesc]);
		params.push(['first', first]);
		params.push(['limit', limit]);
		return await call(`${this.apiURL}/profiles` + queryString(params), http.GET, this.workspaceID);
	};

	events = async (kpid: string): Promise<ProfileEventsResponse> => {
		let params = [];
		let properties = [
			'kpid',
			'connectionId',
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
		params.push(['properties', properties.join(',')]);
		let filter = {
			logical: 'and',
			conditions: [
				{
					property: 'kpid',
					operator: 'is',
					values: [kpid],
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

	attributes = async (kpid: string): Promise<profileAttributesResponse> => {
		return await call(`${this.apiURL}/profiles/${encodeURIComponent(kpid)}/attributes`, http.GET, this.workspaceID);
	};

	identities = async (kpid: string, first: number, limit: number): Promise<IdentitiesResponse> => {
		return await call(
			`${this.apiURL}/profiles/${encodeURIComponent(kpid)}/identities?first=${first}&limit=${limit}`,
			http.GET,
			this.workspaceID,
		);
	};
}

class Workspaces {
	origin: string;
	apiURL: string;
	workspaceID: string;
	connections: Connections;
	eventListeners: EventListeners;
	profiles: Profiles;

	constructor(origin: string, apiURL: string, workspaceID: string) {
		this.origin = origin;
		this.apiURL = apiURL;
		this.workspaceID = workspaceID;
		this.connections = new Connections(apiURL, workspaceID);
		this.eventListeners = new EventListeners(apiURL, workspaceID);
		this.profiles = new Profiles(apiURL, workspaceID);
	}

	// Organization-scoped workspace endpoints must not send Krenalis-Workspace.
	list = async (): Promise<Workspace[]> => {
		const res = await call(`${this.apiURL}/workspaces`, http.GET);
		return res.workspaces as Workspace[];
	};

	create = async (
		name: string,
		profileSchema: ObjectType,
		warehousePlatform: string,
		warehouseMode: WarehouseMode,
		warehouseSettings: WarehouseSettings,
		uiPreferences: UIPreferences,
	): Promise<CreateWorkspaceResponse> => {
		return await call(`${this.apiURL}/workspaces`, http.POST, null, {
			name: name,
			profileSchema: profileSchema,
			warehouse: {
				platform: warehousePlatform,
				mode: warehouseMode,
				settings: warehouseSettings,
			},
			uiPreferences: uiPreferences,
		});
	};

	testCreation = async (
		name: string,
		profileSchema: ObjectType,
		warehousePlatform: string,
		warehouseMode: WarehouseMode,
		warehouseSettings: WarehouseSettings,
		uiPreferences: UIPreferences,
	): Promise<void> => {
		return await call(`${this.apiURL}/workspaces/test`, http.POST, null, {
			name: name,
			profileSchema: profileSchema,
			warehouse: {
				platform: warehousePlatform,
				mode: warehouseMode,
				settings: warehouseSettings,
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

	profileSchema = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/profiles/schema`, http.GET, this.workspaceID);
	};

	profilePropertiesSuitableAsIdentifiers = async (): Promise<ObjectType> => {
		return await call(`${this.apiURL}/profiles/schema/suitable-as-identifiers`, http.GET, this.workspaceID);
	};

	createConnection = async (connection: ConnectionToAdd, authToken: string): Promise<string> => {
		const res: CreateConnectionResponse = await call(`${this.apiURL}/connections`, http.POST, this.workspaceID, {
			...connection,
			authToken: authToken,
		});
		return res.id;
	};

	authToken = async (connector: string, authCode: string, redirectURI: string): Promise<string> => {
		const res: authTokenResponse = await call(
			`${this.apiURL}/connections/auth-token?connector=${connector}&redirectURI=${encodeURIComponent(redirectURI)}&authCode=${encodeURIComponent(authCode)}`,
			http.GET,
			this.workspaceID,
		);
		return res.authToken;
	};

	updateIdentityResolution = async (runOnBatchImport: boolean, identifiers: Identifiers): Promise<void> => {
		return await call(`${this.apiURL}/identity-resolution/settings`, http.PUT, this.workspaceID, {
			runOnBatchImport,
			identifiers,
		});
	};

	warehouse = async (workspaceID?: string): Promise<WarehouseResponse> => {
		return await call(`${this.apiURL}/warehouse`, http.GET, workspaceID != null ? workspaceID : this.workspaceID);
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

	alterProfileSchema = async (
		schema: ObjectType,
		primarySources: PrimarySources,
		rePaths: RePaths,
	): Promise<void> => {
		return await call(`${this.apiURL}/profiles/schema`, http.PUT, this.workspaceID, {
			schema,
			primarySources,
			rePaths,
		});
	};

	previewAlterProfileSchema = async (
		schema: ObjectType,
		rePaths: RePaths,
	): Promise<PreviewAlterProfileSchemaResponse> => {
		return await call(`${this.apiURL}/profiles/schema/preview`, http.PUT, this.workspaceID, {
			schema,
			rePaths,
		});
	};

	pipelineErrors = async (
		start: Date,
		end: Date,
		pipelines: string[],
		first: number,
		limit: number,
		step?: PipelineStep,
	): Promise<PipelineErrorsResponse> => {
		const r: PipelineErrorsResponse = await call(
			`${this.apiURL}/pipelines/errors/` +
				`${encodeURIComponent(start.toISOString())}/` +
				`${encodeURIComponent(end.toISOString())}` +
				`?pipelines=${pipelines.join(',')}` +
				`&first=${first}` +
				`&limit=${limit}` +
				(step ? `&step=${step}` : ''),
			http.GET,
			this.workspaceID,
		);
		for (let i = 0; i < r.errors.length; i++) {
			r.errors[i].lastOccurred = new Date(r.errors[i].lastOccurred);
		}
		return r;
	};

	pipelineMetricsQuery = (filter?: PipelineMetricsFilter): string => {
		if (!filter) {
			return '';
		}
		const filterCount = [filter.pipelines, filter.workspaces, filter.connections].filter((v) => v != null).length;
		if (filterCount > 1) {
			throw new Error('pipelines, workspaces and connections filters cannot be used together');
		}
		let params = [];
		if (filter.pipelines != null) {
			params.push(`pipelines=${filter.pipelines.map(encodeURIComponent).join(',')}`);
		}
		if (filter.workspaces != null) {
			params.push(`workspaces=${filter.workspaces.map(encodeURIComponent).join(',')}`);
		}
		if (filter.connections != null) {
			params.push(`connections=${filter.connections.map(encodeURIComponent).join(',')}`);
		}
		if (filter.target != null) {
			params.push(`target=${encodeURIComponent(filter.target)}`);
		}
		return params.length > 0 ? `?${params.join('&')}` : '';
	};

	pipelineMetricsPerDate = async (
		start: Date,
		end: Date,
		pipelines?: string[],
		filter?: Omit<PipelineMetricsFilter, 'pipelines'>,
	): Promise<PipelineMetrics> => {
		const sd = start.toISOString().split('T')[0];
		const ed = end.toISOString().split('T')[0];
		const q = this.pipelineMetricsQuery(filter ?? (pipelines != null ? { pipelines } : undefined));
		const r = await call(
			`${this.apiURL}/pipelines/metrics/dates/` + `${encodeURIComponent(sd)}/` + `${encodeURIComponent(ed)}` + q,
			http.GET,
			this.workspaceID,
		);
		r.start = new Date(r.start);
		r.end = new Date(r.end);
		return r;
	};

	pipelineMetricsPerDay = async (
		days: number,
		pipelines?: string[],
		filter?: Omit<PipelineMetricsFilter, 'pipelines'>,
	): Promise<PipelineMetrics> => {
		const q = this.pipelineMetricsQuery(filter ?? (pipelines != null ? { pipelines } : undefined));
		const r = await call(
			`${this.apiURL}/pipelines/metrics/days/` + `${encodeURIComponent(days)}` + q,
			http.GET,
			this.workspaceID,
		);
		r.start = new Date(r.start);
		r.end = new Date(r.end);
		return r;
	};

	pipelineMetricsPerHour = async (
		hours: number,
		pipelines?: string[],
		filter?: Omit<PipelineMetricsFilter, 'pipelines'>,
	): Promise<PipelineMetrics> => {
		const q = this.pipelineMetricsQuery(filter ?? (pipelines != null ? { pipelines } : undefined));
		const r = await call(
			`${this.apiURL}/pipelines/metrics/hours/` + `${encodeURIComponent(hours)}` + q,
			http.GET,
			this.workspaceID,
		);
		r.start = new Date(r.start);
		r.end = new Date(r.end);
		return r;
	};

	pipelineMetricsPerMinute = async (
		minutes: number,
		pipelines?: string[],
		filter?: Omit<PipelineMetricsFilter, 'pipelines'>,
	): Promise<PipelineMetrics> => {
		const q = this.pipelineMetricsQuery(filter ?? (pipelines != null ? { pipelines } : undefined));
		const r = await call(
			`${this.apiURL}/pipelines/metrics/minutes/` + `${encodeURIComponent(minutes)}` + q,
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

	LatestAlterProfileSchema = async (): Promise<LatestAlterProfileSchema> => {
		return await call(`${this.apiURL}/profiles/schema/latest-alter`, http.GET, this.workspaceID);
	};
}

class Connectors {
	origin: string;
	apiURL: string;

	constructor(origin: string, apiURL: string) {
		this.origin = origin;
		this.apiURL = apiURL;
	}

	authURL = async (connector: string, role: Role, redirectURI: string): Promise<authURLResponse> => {
		return await call(
			`${this.apiURL}/connections/auth-url?connector=${connector}&role=${role}&redirectURI=${encodeURIComponent(redirectURI)}`,
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
		workspace: string,
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
		workspace: string,
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
