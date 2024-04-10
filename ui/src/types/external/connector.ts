import { ActionTarget } from './action';

type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'Mobile' | 'Server' | 'Stream' | 'Website';

type WebhooksPer = 'None' | 'Connector' | 'Resource' | 'Source';

type SendingMode = 'Cloud' | 'Device' | 'Combined';

interface Connector {
	ID: number;
	Name: string;
	SourceDescription: string;
	DestinationDescription: string;
	Type: ConnectorType;
	HasSheets: boolean;
	HasUI: boolean;
	Icon: string;
	FileExtension: string;
	SampleQuery: string;
	WebhooksPer: WebhooksPer;
	OAuth: boolean;
	TermForUsers: string;
	TermForGroups: string;
	SendingMode: SendingMode | null;
	Targets: Record<ActionTarget, boolean>;
	SuggestedDisplayedID: string;
}

export type { Connector, ConnectorType, WebhooksPer, SendingMode };
