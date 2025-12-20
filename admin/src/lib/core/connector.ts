import {
	SourceConnector,
	DestinationConnector,
	ConnectorType,
	SendingMode,
	ConnectorTerms,
} from '../api/types/connector';
import { Role } from '../api/types/types';
import { ConnectorOAuth } from '../api/types/connector';

class TransformedConnector {
	code: string;
	label: string;
	type: ConnectorType;
	categories: Array<string>;
	asSource: SourceConnector | null;
	asDestination: DestinationConnector | null;
	hasSheets: boolean;
	fileExtension: string;
	oauth: ConnectorOAuth;
	terms: ConnectorTerms;
	strategies: boolean;

	constructor(
		code: string,
		label: string,
		type: ConnectorType,
		categories: Array<string>,
		asSource: SourceConnector | null,
		asDestination: DestinationConnector | null,
		hasSheets: boolean,
		fileExtension: string,
		oauth: ConnectorOAuth,
		terms: ConnectorTerms,
		strategies: boolean,
	) {
		this.code = code;
		this.label = label;
		this.type = type;
		this.categories = categories;
		this.asSource = asSource;
		this.asDestination = asDestination;
		this.hasSheets = hasSheets;
		this.fileExtension = fileExtension;
		this.oauth = oauth;
		this.terms = terms;
		this.strategies = strategies;
	}

	get hasSnippet() {
		return this.type === 'SDK';
	}

	get isApplication() {
		return this.type === 'Application';
	}

	get isDatabase() {
		return this.type === 'Database';
	}

	get isFile() {
		return this.type === 'File';
	}

	get isFileStorage() {
		return this.type === 'FileStorage';
	}

	get isMessageBroker() {
		return this.type === 'MessageBroker';
	}

	get isSDK() {
		return this.type === 'SDK';
	}

	get isWebhook() {
		return this.type === 'Webhook';
	}

	get supportedSendingModes(): SendingMode[] {
		if (this.asDestination == null) {
			return [];
		}
		switch (this.asDestination.sendingMode) {
			case null:
				return [];
			case 'ClientAndServer':
				return ['Client', 'Server', 'ClientAndServer'];
			default:
				return [this.asDestination.sendingMode];
		}
	}

	hasSettings(role: Role): boolean {
		return (
			(role === 'Source' && this.asSource.hasSettings) ||
			(role === 'Destination' && this.asDestination.hasSettings)
		);
	}
}

export default TransformedConnector;
