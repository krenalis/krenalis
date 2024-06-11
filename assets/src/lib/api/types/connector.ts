import { ActionTarget } from './action';

type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'Mobile' | 'Server' | 'Stream' | 'Website';

type WebhooksPer = 'None' | 'Account' | 'Connection' | 'Connector';

type SendingMode = 'Cloud' | 'Device' | 'Combined';

interface Connector {
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
}

export type { Connector, ConnectorType, WebhooksPer, SendingMode };
