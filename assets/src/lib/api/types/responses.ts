import { ActionError } from './action';
import { AccessKey } from './organization';
import { ObjectType } from './types';
import ConnectorField, { ConnectorButton, ConnectorAlert } from './ui';
import { UserEvent, UserIdentity, UserTraits } from './user';

interface authCodeURLResponse {
	url: string;
}

type ConnectorSettings = Record<string, any>;

interface ConnectorUIResponse {
	alert: ConnectorAlert;
	fields: ConnectorField[];
	buttons: ConnectorButton[];
	settings: ConnectorSettings;
}

interface Execution {
	id: number;
	action: number;
	startTime: string;
	endTime?: string;
	passed: number[];
	failed: number[];
	error: string;
}

type Event = Record<string, any>;

interface EventListenerEventsResponse {
	events: Event[];
	omitted: number;
}

interface CreateEventListenerResponse {
	id: string;
}

interface ActionMatchingSchemas {
	internal: ObjectType;
	external: ObjectType;
}

interface ActionSchemasResponse {
	in: ObjectType;
	out: ObjectType;
	matchings: ActionMatchingSchemas;
}

interface ExecQueryResponse {
	rows: Record<string, any>[];
	schema: ObjectType;
	issues: string[];
}

interface RecordsResponse {
	records: Record<string, any>[];
	schema: ObjectType;
	issues: string[];
}

interface AbsolutePathResponse {
	path: string;
}

interface SheetsResponse {
	sheets: string[];
}

interface TableSchemaResponse {
	schema: ObjectType;
	issues: string[];
}

type TelemetryLevel = 'none' | 'errors' | 'stats' | 'all';

interface TransformationLanguagesResponse {
	languages: string[];
}

interface TransformDataResponse {
	data: Record<string, any>;
}

interface ResponseUser {
	id: string;
	sourcesLastUpdate: string;
	traits: Record<string, any>;
}

interface FindUsersResponse {
	users: ResponseUser[];
	schema: ObjectType;
	total: number;
}

interface AppUsersResponse {
	users: Record<string, any>[];
	cursor: string;
}

interface UserEventsResponse {
	events: UserEvent[];
}

interface userTraitsResponse {
	traits: UserTraits;
}

interface UserIdentitiesResponse {
	identities: UserIdentity[];
	total: number;
}

interface ConnectionIdentitiesResponse {
	identities: UserIdentity[];
	total: number;
}

interface PreviewSendEventResponse {
	preview: string;
}

interface MemberAvatar {
	image: string;
	mimeType: string;
}

interface Member {
	id: number;
	name: string;
	email: string;
	avatar: MemberAvatar;
	invitation: '' | 'Invited' | 'Expired';
	createdAt: string;
}

interface MemberToSet {
	name: string;
	image?: string | null;
	email: string;
	password?: string;
}

interface MemberInvitationResponse {
	email: string;
	organization: string;
}

interface AccessKeyResponse {
	keys: AccessKey[];
}

interface CreateAccessKeyResponse {
	id: number;
	token: string;
}

type RePaths = Record<string, string | null>;

interface PreviewAlterUserSchemaResponse {
	queries: string[];
}

interface ActionErrorsResponse {
	errors: ActionError[];
}

interface PublicMetadata {
	installationID: string;
	externalURL: string;
	externalEventURL: string;
	javascriptSDKURL: string;
	memberEmailVerificationRequired: boolean;
	canSendMemberPasswordReset: boolean;
	telemetryLevel: TelemetryLevel;
}

export type {
	authCodeURLResponse,
	userTraitsResponse,
	ActionError,
	ActionErrorsResponse,
	ActionSchemasResponse,
	AppUsersResponse,
	AbsolutePathResponse,
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
	MemberAvatar,
	MemberInvitationResponse,
	MemberToSet,
	AccessKey,
	AccessKeyResponse,
	CreateAccessKeyResponse,
	PreviewSendEventResponse,
	PreviewAlterUserSchemaResponse,
	RePaths,
	RecordsResponse,
	ResponseUser,
	SheetsResponse,
	TableSchemaResponse,
	TelemetryLevel,
	TransformDataResponse,
	TransformationLanguagesResponse,
	UserEventsResponse,
	UserIdentitiesResponse,
	PublicMetadata,
};
