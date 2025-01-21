import { ActionError } from './action';
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
	error: string;
}

type Event = Record<string, any>;

interface EventListenerEventsResponse {
	events: Event[];
	discarded: number;
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
}

interface RecordsResponse {
	records: Record<string, any>[];
	schema: ObjectType;
}

interface CompletePathResponse {
	path: string;
}

interface SheetsResponse {
	sheets: string[];
}

interface TransformationLanguagesResponse {
	languages: string[];
}

interface TransformDataResponse {
	data: Record<string, any>;
}

interface ResponseUser {
	id: string;
	lastChangeTime: string;
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
	image: string;
	email: string;
	password?: string;
}

interface MemberInvitationResponse {
	email: string;
	organization: string;
}

type RePaths = Record<string, string | null>;

interface PreviewUserSchemaUpdateResponse {
	queries: string[];
}

interface ActionErrorsResponse {
	errors: ActionError[];
}

export type {
	authCodeURLResponse,
	userTraitsResponse,
	ActionError,
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
	MemberAvatar,
	MemberInvitationResponse,
	MemberToSet,
	PreviewSendEventResponse,
	PreviewUserSchemaUpdateResponse,
	RePaths,
	RecordsResponse,
	ResponseUser,
	SheetsResponse,
	TransformDataResponse,
	TransformationLanguagesResponse,
	UserEventsResponse,
	UserIdentitiesResponse,
};
