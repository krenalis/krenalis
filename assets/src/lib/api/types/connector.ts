type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'Mobile' | 'Server' | 'Stream' | 'Website';

type WebhooksPer = 'None' | 'Account' | 'Connection' | 'Connector';

type SendingMode = 'Cloud' | 'Device' | 'Combined';

type ConnectorTarget = 'Events' | 'Users' | 'Groups';

interface Connector {
	name: string;
	sourceDescription: string;
	destinationDescription: string;
	type: ConnectorType;
	hasSheets: boolean;
	hasSettings: boolean;
	icon: string;
	fileExtension: string;
	sampleQuery: string;
	webhooksPer: WebhooksPer;
	requiresAuth: boolean;
	termForUsers: string;
	termForGroups: string;
	sendingMode: SendingMode | null;
	targets: ConnectorTarget[];
	identityIDLabel: string;
}

export type { Connector, ConnectorType, WebhooksPer, SendingMode, ConnectorTarget };
