type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'Mobile' | 'Server' | 'Stream' | 'Website';

// type WebhooksPer = 'None' | 'Account' | 'Connection' | 'Connector'; // TODO(marco): implement webhooks

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
	terms: ConnectorTerms;
	icon: string;
}

interface ConnectorTerms {
	user: string;
	users: string;
	// group: string;  // TODO(marco): Implement groups
	// groups: string;
}

interface SourceConnector {
	targets: ConnectorTarget[];
	hasSettings: boolean;
	sampleQuery: string;
	// WebhooksPer: WebhooksPer; // TODO(marco): implement webhooks
	summary: string;
}

interface DestinationConnector {
	targets: ConnectorTarget[];
	hasSettings: boolean;
	sendingMode: SendingMode | null;
	summary: string;
}

export type {
	Connector,
	ConnectorTerms,
	SourceConnector,
	DestinationConnector,
	ConnectorType,
	// WebhooksPer, // TODO(marco): implement webhooks
	SendingMode,
	ConnectorTarget,
};
