import {
	SourceConnector,
	DestinationConnector,
	ConnectorType,
	SendingMode,
	ConnectorTerms,
} from '../api/types/connector';
import * as icons from '../../constants/icons';
import { Role } from '../api/types/types';

class TransformedConnector {
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

	constructor(
		name: string,
		type: ConnectorType,
		categories: Array<string>,
		asSource: SourceConnector | null,
		asDestination: DestinationConnector | null,
		identityIDLabel: string,
		hasSheets: boolean,
		fileExtension: string,
		requiresAuth: boolean,
		authConfigured: boolean,
		terms: ConnectorTerms,
		icon: string,
		strategies: boolean,
	) {
		this.name = name;
		this.type = type;
		this.categories = categories;
		this.asSource = asSource;
		this.asDestination = asDestination;
		this.identityIDLabel = identityIDLabel;
		this.hasSheets = hasSheets;
		this.fileExtension = fileExtension;
		this.requiresAuth = requiresAuth;
		this.authConfigured = authConfigured;
		this.terms = terms;
		this.icon = icon ? icon : icons.PLUG;
		this.strategies = strategies;
	}

	get hasSnippet() {
		return this.type === 'SDK' && this.name !== 'Meergo API';
	}

	get isApp() {
		return this.type === 'App';
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

	get isSDK() {
		return this.type === 'SDK';
	}

	get isStream() {
		return this.type === 'Stream';
	}

	get supportedSendingModes(): SendingMode[] {
		if (this.asDestination == null) {
			return [];
		}
		switch (this.asDestination.sendingMode) {
			case null:
				return [];
			case 'Combined':
				return ['Cloud', 'Device', 'Combined'];
			default:
				return [this.asDestination.sendingMode];
		}
	}

	getIdentityIDLabel(): string {
		let identityIDLabel: string = '';
		if (this.isApp) {
			identityIDLabel = this.identityIDLabel;
			if (identityIDLabel === '') {
				identityIDLabel = 'ID';
			}
		} else if (this.isDatabase || this.isFileStorage) {
			identityIDLabel = 'ID';
		} else if (this.isSDK) {
			identityIDLabel = 'User ID';
		}
		return identityIDLabel;
	}

	hasSettings(role: Role): boolean {
		return (
			(role === 'Source' && this.asSource.hasSettings) ||
			(role === 'Destination' && this.asDestination.hasSettings)
		);
	}
}

export default TransformedConnector;
