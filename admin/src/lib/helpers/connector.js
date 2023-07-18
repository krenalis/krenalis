class Connector {
	constructor(
		id,
		name,
		type,
		hasSheets,
		hasSettings,
		icon,
		fileExtension,
		webhooksPer,
		oAuth,
		sourceDescription,
		destinationDescription
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

	static toConnectorsArray(connectors) {
		return connectors.map(
			(connector) =>
				new Connector(
					connector.ID,
					connector.Name,
					connector.Type,
					connector.HasSheets,
					connector.HasSettings,
					connector.Icon,
					connector.FileExtension,
					connector.WebhooksPer,
					connector.OAuth,
					connector.SourceDescription,
					connector.DestinationDescription
				)
		);
	}

	static toConnector(connector) {
		return new Connector(
			connector.ID,
			connector.Name,
			connector.Type,
			connector.HasSheets,
			connector.HasSettings,
			connector.Icon,
			connector.FileExtension,
			connector.WebhooksPer,
			connector.OAuth,
			connector.SourceDescription,
			connector.DestinationDescription
		);
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

export default Connector;
