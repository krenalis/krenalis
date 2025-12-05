import {
	Connection,
	Health,
	ConnectionRole,
	ConnectorType,
	Compression,
	Strategy,
	SendingMode,
} from '../api/types/connection';
import { Pipeline, PipelineTarget, PipelineType } from '../api/types/pipeline';
import TransformedConnector from './connector';
import { Variant } from '../../components/routes/App/App.types';
import { ConnectorTarget } from '../api/types/connector';
import { TransformedEventType } from './pipeline';

interface ConnectionStatus {
	text: string;
	variant: Variant | '';
}

class TransformedConnection {
	id: number;
	name: string;
	connector: TransformedConnector;
	role: ConnectionRole;
	health: Health;
	storage: number;
	compression: Compression;
	strategy?: Strategy | null;
	sendingMode: SendingMode | null;
	status: ConnectionStatus;
	description: string;
	linkedFiles?: TransformedConnection[];
	pipelineTypes?: PipelineType[];
	pipelines: Pipeline[];
	eventTypes?: TransformedEventType[];
	linkedConnections?: number[];

	constructor(
		id: number,
		name: string,
		connector: TransformedConnector,
		role: ConnectionRole,
		health: Health,
		storage: number,
		compression: Compression,
		strategy: Strategy | null,
		sendingMode: SendingMode | null,
		status: ConnectionStatus,
		description: string,
		linkedFiles?: TransformedConnection[],
		pipelineTypes?: PipelineType[],
		pipelines?: Pipeline[],
		eventTypes?: TransformedEventType[],
		linkedConnections?: number[],
	) {
		this.id = id;
		this.name = name;
		this.connector = connector;
		this.role = role;
		this.health = health;
		this.storage = storage == null ? 0 : storage;
		this.compression = compression;
		this.strategy = strategy;
		this.sendingMode = sendingMode;
		this.status = status;
		this.description = description;
		this.linkedFiles = linkedFiles;
		this.pipelineTypes = pipelineTypes == null ? [] : pipelineTypes;
		this.pipelines = pipelines == null ? [] : pipelines;
		this.eventTypes = eventTypes == null ? [] : eventTypes;
		if (linkedConnections) {
			this.linkedConnections = linkedConnections;
		}
	}

	get isAPI() {
		return this.connector.type === 'API';
	}

	get isDatabase() {
		return this.connector.type === 'Database';
	}

	get isFile() {
		return this.connector.type === 'File' && this.storage !== 0;
	}

	get isFileStorage() {
		return this.connector.type === 'FileStorage';
	}

	get isMessageBroker() {
		return this.connector.type === 'MessageBroker';
	}

	get isSDK() {
		return this.connector.type === 'SDK';
	}

	get isWebhook() {
		return this.connector.type === 'Webhook';
	}

	get isSource() {
		return this.role === 'Source';
	}

	get isDestination() {
		return this.role === 'Destination';
	}

	get isEventBased() {
		return this.connector.type === 'SDK' || this.connector.type === 'Webhook';
	}

	get hasIdentities() {
		return this.role === 'Source' && this.connector.type !== 'MessageBroker';
	}

	get hasAnonymousIdentifiers() {
		return this.connector.type === 'SDK' || this.connector.type === 'Webhook';
	}

	get hasSettings(): boolean {
		return this.connector.hasSettings(this.role);
	}

	relations(connections: TransformedConnection[]): ('dwh-user' | 'dwh-event' | number)[] {
		let hasUsersPipelines = this.pipelines.some((p) => {
			if (this.isSDK || this.isWebhook) {
				return p.target === 'User' && p.enabled;
			}
			return p.target === 'User' && p.enabled && p.schedulePeriod != null;
		});
		let hasEventPipelines = this.pipelines.some((p) => p.target === 'Event' && p.enabled);

		const linkedTo: ('dwh-user' | 'dwh-event' | number)[] = [];
		if (hasUsersPipelines) {
			linkedTo.push('dwh-user');
		}
		if (this.isSource && hasEventPipelines) {
			linkedTo.push('dwh-event');
		}
		if (this.linkedConnections?.length > 0) {
			if (this.isSource) {
				linkedTo.push(
					...this.linkedConnections.filter((id) =>
						connections
							.find((conn) => conn.id === id)
							?.pipelines.some((p) => p.target === 'Event' && p.enabled),
					),
				);
			} else {
				if (this.pipelines.some((p) => p.target === 'Event' && p.enabled))
					linkedTo.push(...this.linkedConnections);
			}
		}
		return linkedTo;
	}
}

const getPipelineTypeFromConnection = (
	connection: TransformedConnection,
	target: PipelineTarget,
	eventType: string | null,
): PipelineType | undefined => {
	let pipelineType: PipelineType | undefined;
	if (target === 'Event') {
		if (eventType == null) {
			pipelineType = connection.pipelineTypes!.find((t) => t.target === 'Event' && t.eventType === null);
		} else {
			pipelineType = connection.pipelineTypes!.find((t) => t.eventType === eventType);
		}
	} else {
		pipelineType = connection.pipelineTypes!.find((t) => t.target === target);
	}
	return pipelineType;
};

const getConnectionDescription = (connection: Connection, connector: TransformedConnector): string => {
	let description: string;
	if (connection.role === 'Source') {
		description = connector.asSource.summary;
	} else {
		description = connector.asDestination.summary;
	}
	return description;
};

const getConnectionFullConnector = (
	connectorCode: string,
	connectors: TransformedConnector[],
): TransformedConnector => {
	return connectors.find((c) => c.code === connectorCode)!;
};

const getConnectionStatus = (connection: Connection): ConnectionStatus => {
	switch (connection.health) {
		case 'Healthy':
			return { text: 'Working properly', variant: 'success' };
		case 'NoRecentData':
			return { text: 'No recent Data', variant: 'danger' };
		case 'RecentError':
			return { text: 'Recent error', variant: 'danger' };
		default:
			return { text: '', variant: '' };
	}
};

const isSourceEventConnection = (role: ConnectionRole, type: ConnectorType): boolean => {
	return role === 'Source' && (type === 'SDK' || type == 'Webhook');
};

const isEventConnection = (role: ConnectionRole, type: ConnectorType, targets: ConnectorTarget[]): boolean => {
	return (
		(role === 'Source' && (type === 'SDK' || type === 'Webhook')) ||
		(role === 'Destination' && type === 'API' && targets.includes('Event'))
	);
};

const getFileStorageConnections = (
	storageID: number,
	connections: TransformedConnection[],
): TransformedConnection[] => {
	return connections.filter((c) => c.storage === storageID);
};

export default TransformedConnection;
export {
	getPipelineTypeFromConnection,
	getConnectionDescription,
	getConnectionFullConnector,
	getConnectionStatus,
	getFileStorageConnections,
	isEventConnection,
	isSourceEventConnection,
};
export type { ConnectionStatus };
