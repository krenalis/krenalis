import { PipelineError } from './pipeline';
import { AccessKey } from './organization';
import { ObjectType } from './types';
import ConnectorField, { ConnectorButton, ConnectorAlert } from './ui';
import { ProfileEvent, Identity, ProfileAttributes } from './profile';

interface authURLResponse {
	authUrl: string;
}

interface authTokenResponse {
	authToken: string;
}

type ConnectorSettings = Record<string, any>;

interface ConnectorUIResponse {
	alert: ConnectorAlert;
	fields: ConnectorField[];
	buttons: ConnectorButton[];
	settings: ConnectorSettings;
}

interface PipelineRun {
	id: number;
	pipeline: number;
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

interface CreateEventWriteKeyResponse {
	key: string;
}

interface CreatePipelineResponse {
	id: number;
}

interface CreateConnectionResponse {
	id: number;
}

interface PipelineMatchingSchemas {
	internal: ObjectType;
	external: ObjectType;
}

interface PipelineSchemasResponse {
	in: ObjectType;
	out: ObjectType;
	matchings: PipelineMatchingSchemas;
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
	kpid: string;
	updatedAt: string;
	attributes: Record<string, any>;
}

interface FindProfilesResponse {
	profiles: ResponseProfile[];
	schema: ObjectType;
	total: number;
}

interface ApplicationUsersResponse {
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

interface PipelineErrorsResponse {
	errors: PipelineError[];
}

interface PublicMetadata {
	installationID: string;
	externalURL: string;
	externalEventURL: string;
	externalAssetsURLs: string[];
	potentialConnectorsURL: string | null;
	javascriptSDKURL: string;
	inviteMembersViaEmail: boolean;
	canSendMemberPasswordReset: boolean;
	telemetryLevel: TelemetryLevel;
	workosClientID: string;
	workosDevMode: boolean;
}

export type {
	authURLResponse,
	authTokenResponse,
	profileAttributesResponse,
	PipelineError,
	PipelineErrorsResponse,
	PipelineSchemasResponse,
	ApplicationUsersResponse,
	AbsolutePathResponse,
	ConnectionIdentitiesResponse,
	ConnectorSettings,
	ConnectorUIResponse,
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
