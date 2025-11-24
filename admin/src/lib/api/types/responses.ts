import { ActionError } from './action';
import { AccessKey } from './organization';
import { ObjectType } from './types';
import ConnectorField, { ConnectorButton, ConnectorAlert } from './ui';
import { ProfileEvent, Identity, ProfileAttributes } from './profile';

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

interface ResponseProfile {
	mpid: string;
	sourcesLastUpdate: string;
	attributes: Record<string, any>;
}

interface FindProfilesResponse {
	profiles: ResponseProfile[];
	schema: ObjectType;
	total: number;
}

interface APIUsersResponse {
	users: Record<string, any>[];
	cursor: string;
}

interface ProfileEventsResponse {
	events: ProfileEvent[];
}

interface profileAttributesResponse {
	attributes: ProfileAttributes;
}

interface IdentitiesResponse {
	identities: Identity[];
	total: number;
}

interface ConnectionIdentitiesResponse {
	identities: Identity[];
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

interface PreviewAlterProfileSchemaResponse {
	queries: string[];
}

interface ActionErrorsResponse {
	errors: ActionError[];
}

interface PublicMetadata {
	installationID: string;
	externalURL: string;
	externalEventURL: string;
	externalAssetsURLs: string[];
	javascriptSDKURL: string;
	memberEmailVerificationRequired: boolean;
	canSendMemberPasswordReset: boolean;
	telemetryLevel: TelemetryLevel;
}

export type {
	authCodeURLResponse,
	profileAttributesResponse,
	ActionError,
	ActionErrorsResponse,
	ActionSchemasResponse,
	APIUsersResponse,
	AbsolutePathResponse,
	ConnectionIdentitiesResponse,
	ConnectorSettings,
	ConnectorUIResponse,
	CreateEventListenerResponse,
	Event,
	EventListenerEventsResponse,
	ExecQueryResponse,
	Execution,
	FindProfilesResponse,
	Member,
	MemberAvatar,
	MemberInvitationResponse,
	MemberToSet,
	AccessKey,
	AccessKeyResponse,
	CreateAccessKeyResponse,
	PreviewSendEventResponse,
	PreviewAlterProfileSchemaResponse,
	RePaths,
	RecordsResponse,
	ResponseProfile,
	SheetsResponse,
	TableSchemaResponse,
	TelemetryLevel,
	TransformDataResponse,
	TransformationLanguagesResponse,
	ProfileEventsResponse,
	IdentitiesResponse,
	PublicMetadata,
};
