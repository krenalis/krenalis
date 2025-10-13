type ConnectorType = 'App' | 'Database' | 'File' | 'FileStorage' | 'SDK' | 'Stream';

// type WebhooksPer = 'None' | 'Account' | 'Connection' | 'Connector'; // TODO(marco): implement webhooks

type SendingMode = 'Client' | 'Server' | 'ClientAndServer';

type ConnectorTarget = 'Event' | 'User' | 'Group';

interface Connector {
	code: string;
	label: string;
	type: ConnectorType;
	categories: Array<string>;
	asSource: SourceConnector | null;
	asDestination: DestinationConnector | null;
	identityIDLabel: string;
	hasSheets: boolean;
	fileExtension: string;
	oauth: ConnectorOAuth;
	terms: ConnectorTerms;
	strategies: boolean;
}

interface ConnectorOAuth {
	configured: boolean;
	disallow127_0_0_1: boolean;
	disallowLocalhost: boolean;
}

interface ConnectorImplementation {
	description: string;
	implemented: boolean;
	comingSoon: boolean;
}

interface PotentialConnector {
	code: string;
	label: string;
	categories: Array<string>;
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
	ConnectorOAuth,
	ConnectorTerms,
	SourceConnector,
	DestinationConnector,
	ConnectorType,
	PotentialConnector,
	ConnectorImplementation,
	ConnectorRoleDocumentation,
	ConnectorDocumentation,
	// WebhooksPer, // TODO(marco): implement webhooks
	SendingMode,
	ConnectorTarget,
};
