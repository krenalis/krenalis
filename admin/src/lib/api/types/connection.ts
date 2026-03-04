import { Pipeline, PipelineType } from './pipeline';
import { ConnectorSettings } from './responses';

type ConnectorType = 'Application' | 'Database' | 'File' | 'FileStorage' | 'MessageBroker' | 'SDK' | 'Webhook';

type ConnectionRole = 'Source' | 'Destination';

type Health = 'Healthy' | 'NoRecentData' | 'RecentError';

interface EventType {
	id: string;
	name: string;
	description: string;
	defaultFilter: string;
}

interface Connection {
	id: number;
	name: string;
	connector: string;
	connectorType: ConnectorType;
	role: ConnectionRole;
	storage: number;
	compression: Compression;
	strategy?: Strategy | null;
	sendingMode: SendingMode | null;
	hasSettings: boolean;
	health: Health;
	pipelineTypes?: PipelineType[];
	pipelines?: Pipeline[];
	eventTypes?: EventType[]; // Only returned by 'api/connections/{id}'
	linkedConnections?: number[];
}

type Compression = '' | 'Zip' | 'Gzip' | 'Snappy';

type Strategy = 'Conversion' | 'Fusion' | 'Isolation' | 'Preservation';

type SendingMode = 'Client' | 'Server' | 'ClientAndServer';

interface ConnectionToAdd {
	name: string;
	role: string;
	connector: string;
	strategy?: Strategy | null;
	sendingMode?: SendingMode | null;
	settings?: ConnectorSettings | null;
	linkedConnections: Number[] | null;
}

interface ConnectionToSet {
	name: string;
	strategy?: Strategy | null;
	sendingMode?: SendingMode | null;
}

export type {
	Connection,
	ConnectionRole,
	Compression,
	Strategy,
	SendingMode,
	ConnectionToAdd,
	ConnectionToSet,
	ConnectorType,
	Health,
};
