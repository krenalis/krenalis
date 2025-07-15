type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'SDK' | 'Stream';

// type WebhooksPer = 'None' | 'Account' | 'Connection' | 'Connector'; // TODO(marco): implement webhooks

type SendingMode = 'Client' | 'Server' | 'ClientAndServer';

type ConnectorTarget = 'Event' | 'User' | 'Group';

interface Connector {
	name: string;
	type: ConnectorType;
	categories: Array<string>;
	asSource: SourceConnector | null;
	asDestination: DestinationConnector | null;
	identityIDLabel: string;
	hasSheets: boolean;
	fileExtension: string;
	requiresAuth: boolean;
	authConfigured: boolean;
	terms: ConnectorTerms;
	icon: string;
	strategies: boolean;
}

interface ConnectorImplementation {
	description: string;
	implemented: boolean;
	comingSoon: boolean;
}

interface ConnectorInfo {
	name: string;
	categories: Array<string>;
	icon: string;
	iconLicense: string;
	connectorType: ConnectorType;
	asSource: ConnectorImplementation;
	asDestination: ConnectorImplementation;
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
	ConnectorInfo,
	ConnectorRoleDocumentation,
	ConnectorDocumentation,
	// WebhooksPer, // TODO(marco): implement webhooks
	SendingMode,
	ConnectorTarget,
};
