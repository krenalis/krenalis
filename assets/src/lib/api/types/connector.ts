type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'Mobile' | 'Server' | 'Stream' | 'Website';

type WebhooksPer = 'None' | 'Account' | 'Connection' | 'Connector';

type SendingMode = 'Cloud' | 'Device' | 'Combined';

type ConnectorTarget = 'Events' | 'Users' | 'Groups';

interface Connector {
	name: string;
	type: ConnectorType;
	asSource: SourceConnector | null;
	asDestination: DestinationConnector | null;
	identityIDLabel: string;
	hasSheets: boolean;
	fileExtension: string;
	requiresAuth: boolean;
	termForUsers: string;
	termForGroups: string;
	icon: string;
}

interface SourceConnector {
	description: string;
	targets: ConnectorTarget[];
	hasSettings: boolean;
	sampleQuery: string;
	WebhooksPer: WebhooksPer;
}

interface DestinationConnector {
	description: string;
	targets: ConnectorTarget[];
	hasSettings: boolean;
	sendingMode: SendingMode | null;
}

export type {
	Connector,
	SourceConnector,
	DestinationConnector,
	ConnectorType,
	WebhooksPer,
	SendingMode,
	ConnectorTarget,
};
