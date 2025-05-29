type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'SDK' | 'Stream';

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
	strategies: boolean;
}

interface ConnectorRoleDocumentation {
	Summary: string;
	Overview: string;
}

interface ConnectorDocumentation {
	Source: ConnectorRoleDocumentation;
	Destination: ConnectorRoleDocumentation;
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
	ConnectorRoleDocumentation,
	ConnectorDocumentation,
	// WebhooksPer, // TODO(marco): implement webhooks
	SendingMode,
	ConnectorTarget,
};
