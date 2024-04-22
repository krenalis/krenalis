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

interface EventListenerEventsResponse {
	events: ObservedEvent[];
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

type FilterLogical = 'all' | 'any';

interface FilterCondition {
	Property: string;
	Operator: string;
	Value: string;
}

interface Filter {
	Logical: FilterLogical;
	Conditions: FilterCondition[];
}

interface FindUsersResponse {
	users: Record<string, any>[];
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

interface ObservedEventHeader {
	receivedAt: string;
	remoteAddr: string;
	method: string;
	proto: string;
	url: string;
	headers: Record<string, string>;
}

interface ObservedEvent {
	Source: number;
	Header: ObservedEventHeader;
	Data: string;
	Err: string;
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

interface ChangeUsersSchemaQueriesResponse {
	Queries: string[];
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
	FilterLogical,
	FilterCondition,
	Filter,
	FindUsersResponse,
	AppUsersResponse,
	UserEventsResponse,
	userTraitsResponse,
	UserIdentitiesResponse,
	ConnectionIdentitiesResponse,
	EventPreviewResponse,
	ObservedEvent,
	Member,
	MemberToSet,
	MemberAvatar,
	MemberInvitationResponse,
	RePaths,
	ChangeUsersSchemaQueriesResponse,
};
