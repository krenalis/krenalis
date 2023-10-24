import { ObjectType } from './types';
import ConnectorField, { ConnectorAction, ConnectorAlert } from './ui';
import { AppUser, UserEvent, UserTraits } from './user';

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

interface Import {
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

interface ActionSchemasResponse {
	In: ObjectType;
	Out: ObjectType;
}

interface ExecQueryResponse {
	Rows: string[][];
	Schema: ObjectType;
}

interface RecordsResponse {
	records: any[][];
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

interface TransformationPreviewResponse {
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
	count: number;
	schema: ObjectType;
	users: any[][];
}

interface AppUsersResponse {
	users: AppUser[];
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
	Import,
	EventListenerEventsResponse,
	AddEventListenerResponse,
	ActionSchemasResponse,
	ExecQueryResponse,
	RecordsResponse,
	CompletePathResponse,
	SheetsResponse,
	TransformationLanguagesResponse,
	TransformationPreviewResponse,
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
