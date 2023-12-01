import { ObjectType } from './types';
import ConnectorField, { ConnectorAction, ConnectorAlert } from './ui';
import { UserEvent, UserTraits } from './user';

interface authCodeURLResponse {
	url: string;
}

type UIValues = Record<string, any>;

interface UIForm {
	Fields: ConnectorField[];
	Actions: ConnectorAction[];
	Values: UIValues;
}

interface UIResponse {
	Form: UIForm;
	Alert: ConnectorAlert;
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
	users: any[][];
	schema: ObjectType;
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

export type {
	authCodeURLResponse,
	UIResponse,
	UIValues,
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
	EventPreviewResponse,
	ObservedEvent,
};
