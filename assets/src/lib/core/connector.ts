import {
	SourceConnector,
	DestinationConnector,
	ConnectorTarget,
	ConnectorType,
	SendingMode,
} from '../api/types/connector';
import * as icons from '../../constants/icons';
import { Role } from '../api/types/types';

class TransformedConnector {
	name: string;
	type: ConnectorType;
	source: SourceConnector | null;
	destination: DestinationConnector | null;
	targets: ConnectorTarget[];
	identityIDLabel: string;
	hasSheets: boolean;
	fileExtension: string;
	requiresAuth: boolean;
	termForUsers: string;
	termForGroups: string;
	icon: string;

	constructor(
		name: string,
		type: ConnectorType,
		source: SourceConnector | null,
		destination: DestinationConnector | null,
		targets: ConnectorTarget[],
		identityIDLabel: string,
		hasSheets: boolean,
		fileExtension: string,
		requiresAuth: boolean,
		termForUsers: string,
		termForGroups: string,
		icon: string,
	) {
		this.name = name;
		this.type = type;
		this.source = source;
		this.destination = destination;
		this.targets = targets;
		this.identityIDLabel = identityIDLabel;
		this.hasSheets = hasSheets;
		this.fileExtension = fileExtension;
		this.requiresAuth = requiresAuth;
		this.termForUsers = termForUsers;
		this.termForGroups = termForGroups;
		this.icon = icon ? icon : icons.PLUG;
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

	get isMobile() {
		return this.type === 'Mobile';
	}

	get isServer() {
		return this.type === 'Server';
	}

	get isStream() {
		return this.type === 'Stream';
	}

	get isWebsite() {
		return this.type === 'Website';
	}

	get supportedSendingModes(): SendingMode[] {
		if (this.destination == null) {
			return [];
		}
		switch (this.destination.sendingMode) {
			case null:
				return [];
			case 'Combined':
				return ['Cloud', 'Device', 'Combined'];
			default:
				return [this.destination.sendingMode];
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
		} else if (this.isMobile || this.isServer || this.isWebsite) {
			identityIDLabel = 'User ID';
		}
		return identityIDLabel;
	}

	hasSettings(role: Role): boolean {
		return (
			(role === 'Source' && this.source.hasSettings) || (role === 'Destination' && this.destination.hasSettings)
		);
	}
}

export default TransformedConnector;
