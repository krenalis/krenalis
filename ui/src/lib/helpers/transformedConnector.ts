import { ConnectorType, SendingMode, WebhooksPer } from '../../types/external/connector';
import { ActionTarget } from '../../types/external/action';

class TransformedConnector {
	id: number;
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
	oAuth: boolean;
	termForUsers: string;
	termForGroups: string;
	SendingMode: SendingMode | null;
	targets: Record<ActionTarget, boolean>;
	suggestedDisplayedID: string;

	constructor(
		id: number,
		name: string,
		type: ConnectorType,
		hasSheets: boolean,
		hasSettings: boolean,
		icon: string,
		fileExtension: string,
		sampleQuery: string,
		webhooksPer: WebhooksPer,
		oAuth: boolean,
		sourceDescription: string,
		destinationDescription: string,
		termForUsers: string,
		termForGroups: string,
		SendingMode: SendingMode,
		targets: Record<ActionTarget, boolean>,
		suggestedDisplayedID: string,
	) {
		this.id = id;
		this.name = name;
		this.type = type;
		this.hasSheets = hasSheets;
		this.hasSettings = hasSettings;
		this.icon = icon;
		this.fileExtension = fileExtension;
		this.sampleQuery = sampleQuery;
		this.webhooksPer = webhooksPer;
		this.oAuth = oAuth;
		this.sourceDescription = sourceDescription;
		this.destinationDescription = destinationDescription;
		this.termForUsers = termForUsers;
		this.termForGroups = termForGroups;
		this.SendingMode = SendingMode;
		this.targets = targets;
		this.suggestedDisplayedID = suggestedDisplayedID;
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
		switch (this.SendingMode) {
			case null:
				return [];
			case 'Combined':
				return ['Cloud', 'Device', 'Combined'];
			default:
				return [this.SendingMode];
		}
	}
}

export default TransformedConnector;
