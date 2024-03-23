import { ActionTarget } from './action';

type ConnectorType = 'App' | 'Database' | 'File' | 'Mobile' | 'Server' | 'Storage' | 'Stream' | 'Website';

type WebhooksPer = 'None' | 'Connector' | 'Resource' | 'Source';

interface Connector {
	ID: number;
	Name: string;
	SourceDescription: string;
	DestinationDescription: string;
	Type: ConnectorType;
	HasSheets: boolean;
	HasSettings: boolean;
	Icon: string;
	FileExtension: string;
	SampleQuery: string;
	WebhooksPer: WebhooksPer;
	OAuth: boolean;
	TermForUsers: string;
	TermForGroups: string;
	Targets: Record<ActionTarget, boolean>;
}

export type { Connector, ConnectorType, WebhooksPer };
