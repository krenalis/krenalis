import { ConnectorType, WebhooksPer } from '../../types/external/connector';

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

	constructor(
		id: number,
		name: string,
		type: ConnectorType,
		hasSheets: boolean,
		hasSettings: boolean,
		icon: string,
		fileExtension: string,
		webhooksPer: WebhooksPer,
		oAuth: boolean,
		sourceDescription: string,
		destinationDescription: string
	) {
		this.id = id;
		this.name = name;
		this.type = type;
		this.hasSheets = hasSheets;
		this.hasSettings = hasSettings;
		this.icon = icon;
		this.fileExtension = fileExtension;
		this.webhooksPer = webhooksPer;
		this.oAuth = oAuth;
		this.sourceDescription = sourceDescription;
		this.destinationDescription = destinationDescription;
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

	get isMobile() {
		return this.type === 'Mobile';
	}

	get isServer() {
		return this.type === 'Server';
	}

	get isStorage() {
		return this.type === 'Storage';
	}

	get isStream() {
		return this.type === 'Stream';
	}

	get isWebsite() {
		return this.type === 'Website';
	}
}

export default TransformedConnector;
