import { ActionError } from './action';
import { ObjectType } from './types';
import ConnectorField, { ConnectorButton, ConnectorAlert } from './ui';
import { UserEvent, UserIdentity, UserTraits } from './user';

interface authCodeURLResponse {
	url: string;
}

type ConnectorValues = Record<string, any>;

interface ConnectorUIResponse {
	Alert: ConnectorAlert;
	Fields: ConnectorField[];
	Buttons: ConnectorButton[];
	Values: ConnectorValues;
}

interface Execution {
	ID: number;
	Action: number;
	StartTime: string;
	EndTime?: string;
	Error: string;
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
	Internal: ObjectType;
	External: ObjectType;
}

interface ActionSchemasResponse {
	In: ObjectType;
	Out: ObjectType;
	Matchings: ActionMatchingSchemas;
}

interface ExecQueryResponse {
	Rows: Record<string, any>[];
	Schema: ObjectType;
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
	Image: string;
	MimeType: string;
}

interface Member {
	ID: number;
	Name: string;
	Email: string;
	Avatar: MemberAvatar;
	Invitation: '' | 'Invited' | 'Expired';
	CreatedAt: string;
}

interface MemberToSet {
	Name: string;
	Image: string;
	Email: string;
	Password?: string;
}

interface MemberInvitationResponse {
	email: string;
	organization: string;
}

type RePaths = Record<string, string | null>;

interface ChangeUserSchemaQueriesResponse {
	Queries: string[];
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
