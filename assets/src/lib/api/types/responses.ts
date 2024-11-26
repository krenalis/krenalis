import { ActionError } from './action';
import { ObjectType } from './types';
import ConnectorField, { ConnectorButton, ConnectorAlert } from './ui';
import { UserEvent, UserIdentity, UserTraits } from './user';

interface authCodeURLResponse {
	url: string;
}

type ConnectorValues = Record<string, any>;

interface ConnectorUIResponse {
	alert: ConnectorAlert;
	fields: ConnectorField[];
	buttons: ConnectorButton[];
	values: ConnectorValues;
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

interface AddEventListenerResponse {
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
	properties: Record<string, any>;
}

interface FindUsersResponse {
	users: ResponseUser[];
	schema: ObjectType;
	count: number;
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
	count: number;
}

interface ConnectionIdentitiesResponse {
	identities: UserIdentity[];
	count: number;
}

interface EventPreviewResponse {
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

interface ChangeUserSchemaQueriesResponse {
	queries: string[];
}

interface ActionErrorsResponse {
	errors: ActionError[];
}

export type {
	authCodeURLResponse,
	ConnectorUIResponse,
	ConnectorValues,
	Execution,
	EventListenerEventsResponse,
	AddEventListenerResponse,
	ActionSchemasResponse,
	ExecQueryResponse,
	RecordsResponse,
	CompletePathResponse,
	SheetsResponse,
	TransformationLanguagesResponse,
	TransformDataResponse,
	FindUsersResponse,
	AppUsersResponse,
	UserEventsResponse,
	userTraitsResponse,
	UserIdentitiesResponse,
	ConnectionIdentitiesResponse,
	EventPreviewResponse,
	Event,
	Member,
	MemberToSet,
	MemberAvatar,
	MemberInvitationResponse,
	RePaths,
	ChangeUserSchemaQueriesResponse,
	ResponseUser,
	ActionError,
	ActionErrorsResponse,
};
