import { ObjectType } from './types';
import ConnectorField, { ConnectorAction, ConnectorAlert } from './ui';

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
	events: Record<string, any>[];
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

export {
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
};
