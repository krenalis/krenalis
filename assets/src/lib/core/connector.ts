import { ConnectorType, SendingMode, WebhooksPer } from '../api/types/connector';
import { ActionTarget } from '../api/types/action';
import * as icons from '../../constants/icons';

class TransformedConnector {
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
	oAuth: boolean;
	termForUsers: string;
	termForGroups: string;
	sendingMode: SendingMode | null;
	targets: Record<ActionTarget, boolean>;
	identityIDLabel: string;

	constructor(
		name: string,
		type: ConnectorType,
		hasSheets: boolean,
		hasUI: boolean,
		icon: string,
		fileExtension: string,
		sampleQuery: string,
		webhooksPer: WebhooksPer,
		oAuth: boolean,
		sourceDescription: string,
		destinationDescription: string,
		termForUsers: string,
		termForGroups: string,
		sendingMode: SendingMode,
		targets: Record<ActionTarget, boolean>,
		identityIDLabel: string,
	) {
		this.name = name;
		this.type = type;
		this.hasSheets = hasSheets;
		this.hasUI = hasUI;
		this.icon = icon ? icon : icons.PLUG;
		this.fileExtension = fileExtension;
		this.sampleQuery = sampleQuery;
		this.webhooksPer = webhooksPer;
		this.oAuth = oAuth;
		this.sourceDescription = sourceDescription;
		this.destinationDescription = destinationDescription;
		this.termForUsers = termForUsers;
		this.termForGroups = termForGroups;
		this.sendingMode = sendingMode;
		this.targets = targets;
		this.identityIDLabel = identityIDLabel;
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
		switch (this.sendingMode) {
			case null:
				return [];
			case 'Combined':
				return ['Cloud', 'Device', 'Combined'];
			default:
				return [this.sendingMode];
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
}

export default TransformedConnector;
