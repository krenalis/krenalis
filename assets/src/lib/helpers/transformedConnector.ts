import { ConnectorType, SendingMode, WebhooksPer } from '../../types/external/connector';
import { ActionTarget } from '../../types/external/action';
import { PLUG_ICON } from '../../constants/icons';

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
	SendingMode: SendingMode | null;
	targets: Record<ActionTarget, boolean>;
	suggestedDisplayedProperty: string;

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
		SendingMode: SendingMode,
		targets: Record<ActionTarget, boolean>,
		suggestedDisplayeProperty: string,
	) {
		this.name = name;
		this.type = type;
		this.hasSheets = hasSheets;
		this.hasUI = hasUI;
		this.icon = icon ? icon : PLUG_ICON;
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
		this.suggestedDisplayedProperty = suggestedDisplayeProperty;
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
