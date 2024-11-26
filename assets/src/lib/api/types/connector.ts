import { ActionTarget } from './action';

type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'Mobile' | 'Server' | 'Stream' | 'Website';

type WebhooksPer = 'None' | 'Account' | 'Connection' | 'Connector';

type SendingMode = 'Cloud' | 'Device' | 'Combined';

interface Connector {
	name: string;
	sourceDescription: string;
	destinationDescription: string;
	type: ConnectorType;
	hasSheets: boolean;
	hasUI: boolean;
	icon: string;
	fileExtension: string;
	sampleQuery: string;
	webhooksPer: WebhooksPer;
	oauth: boolean;
	termForUsers: string;
	termForGroups: string;
	sendingMode: SendingMode | null;
	targets: Record<ActionTarget, boolean>;
	identityIDLabel: string;
}

export type { Connector, ConnectorType, WebhooksPer, SendingMode };
