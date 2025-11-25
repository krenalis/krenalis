import { PotentialConnector, ConnectorType, ConnectorImplementation } from './types/connector';

const POTENTIAL_CONNECTORS_TIMEOUT_MS = 2000; // in milliseconds
const CONNECTOR_CODE_REGEX = /^[a-z0-9-]+$/;
const ALLOWED_CONNECTOR_TYPES: ReadonlyArray<ConnectorType> = [
	'API',
	'Database',
	'File',
	'FileStorage',
	'MessageBroker',
	'SDK',
	'Webhook',
];

const potentialConnectors = async (
	existingConnectorCodes: ReadonlySet<string> = new Set<string>(),
	potentialConnectorsURL: string,
): Promise<PotentialConnector[]> => {
	const abortController = new AbortController();
	const timeoutId = setTimeout(() => abortController.abort(), POTENTIAL_CONNECTORS_TIMEOUT_MS);

	try {
		const res = await fetch(potentialConnectorsURL, { signal: abortController.signal }).catch((err: unknown) => {
			const message = err instanceof Error ? err.message : 'unknown error';
			if (err instanceof DOMException && err.name === 'AbortError') {
				console.warn(
					`aborted the request to ${potentialConnectorsURL} because it exceeded ${POTENTIAL_CONNECTORS_TIMEOUT_MS} ms`,
				);
			} else {
				console.warn(`failed to fetch ${potentialConnectorsURL}: ${message}`);
			}
			return null;
		});

		if (res == null) {
			return [];
		}

		if (!res.ok) {
			console.error(`received status ${res.status} while fetching ${potentialConnectorsURL}`);
			return [];
		}

		let rawText: string;
		try {
			const buffer = await res.arrayBuffer();
			const decoder = new TextDecoder('utf-8', { fatal: true });
			rawText = decoder.decode(buffer);
		} catch (err) {
			const message = err instanceof Error ? err.message : 'unknown error';
			console.error(`unable to read the response body from ${potentialConnectorsURL}: ${message}`);
			return [];
		}

		let parsed: unknown;
		try {
			parsed = JSON.parse(rawText);
		} catch (err) {
			const message = err instanceof Error ? err.message : 'unknown error';
			console.error(`file ${potentialConnectorsURL} does not contain valid JSON: ${message}`);
			return [];
		}

		return validatePotentialConnectorsResponse(parsed, existingConnectorCodes);
	} catch (err) {
		const message = err instanceof Error ? err.message : 'unknown error';
		console.error(`unexpected error while loading ${potentialConnectorsURL}: ${message}`);
		return [];
	} finally {
		clearTimeout(timeoutId);
	}
};

const validatePotentialConnectorsResponse = (
	catalog: unknown,
	existingConnectorCodes: ReadonlySet<string>,
): PotentialConnector[] => {
	if (!isObject(catalog)) {
		console.warn(`parsing potential connectors: it is not an object`);
		return [];
	}
	if (!('connectors' in catalog) || !Array.isArray(catalog.connectors)) {
		console.warn(`parsing potential connectors: 'connectors' is not an array`);
		return [];
	}
	const alreadySeen = new Set<string>(existingConnectorCodes);
	const connectors: PotentialConnector[] = [];
	for (let i = 0; i < catalog.connectors.length; i++) {
		const c = catalog.connectors[i];
		try {
			const connector = validatePotentialConnector(c);
			if (alreadySeen.has(connector.code)) {
				if (!existingConnectorCodes.has(connector.code)) {
					console.warn(`connector ${connector.code} is already declared in the file`);
				}
				continue;
			}
			connectors.push(connector);
			alreadySeen.add(connector.code);
		} catch (error) {
			console.warn(`parsing potential connectors: ${error}`);
		}
	}
	return connectors;
};

const validatePotentialConnector = (connector: unknown): PotentialConnector => {
	if (!isObject(connector)) {
		throw new Error(`connector is not an object`);
	}

	const code =
		'code' in connector && typeof connector.code === 'string' && CONNECTOR_CODE_REGEX.test(connector.code)
			? connector.code
			: null;
	if (code == null) {
		throw new Error(`code of a connector is invalid`);
	}

	const label =
		'label' in connector && typeof connector.label === 'string' && connector.label.trim().length > 0
			? connector.label.trim()
			: null;
	if (label == null) {
		throw new Error(`connector '${code}' has an invalid label`);
	}

	let categories: string[] = [];
	if ('categories' in connector) {
		if (!Array.isArray(connector.categories)) {
			throw new Error(`connector '${code}' has categories that is not an array`);
		}
		for (let category of connector.categories as unknown[]) {
			if (typeof category !== 'string') {
				throw new Error(`connector '${code}' has a category that is not a string`);
			}
			if (category === '') {
				throw new Error(`connector '${code}' has an empty category`);
			}
			categories.push(category);
		}
	}

	const connectorType =
		'connectorType' in connector && typeof connector.connectorType === 'string'
			? (connector.connectorType as ConnectorType)
			: null;
	if (connectorType == null || !ALLOWED_CONNECTOR_TYPES.includes(connectorType)) {
		throw new Error(`connector '${code}' has in invalid connectorType`);
	}

	const asSource = validateConnectorImplementation(code, 'asSource', connector.asSource);
	const asDestination = validateConnectorImplementation(code, 'asDestination', connector.asDestination);

	if (connectorType === 'SDK' && asDestination != null) {
		throw new Error(`connector '${code}' cannot have 'asDestination' because is an SDK`);
	}
	if (connectorType === 'Webhook' && asDestination != null) {
		throw new Error(`connector '${code}' cannot have 'asDestination' because is a webhook`);
	}

	return { code, label, categories, connectorType, asSource, asDestination };
};

const validateConnectorImplementation = (
	code: string | null,
	field: 'asSource' | 'asDestination',
	value: unknown,
): ConnectorImplementation => {
	if (value === undefined) {
		return null;
	}
	if (!isObject(value)) {
		throw new Error(`connector ${code} has ${field} that is not an object`);
	}
	const description = 'description' in value && typeof value.description === 'string' ? value.description : null;
	if (description == null) {
		throw new Error(`connector ${code} has an invalid 'description' field`);
	}
	const implemented = 'implemented' in value && typeof value.implemented === 'boolean' ? value.implemented : null;
	if (implemented == null) {
		throw new Error(`connector ${code} has an invalid 'implemented' field`);
	}
	const comingSoon = 'comingSoon' in value && typeof value.comingSoon === 'boolean' ? value.comingSoon : null;
	if (comingSoon == null) {
		throw new Error(`connector ${code} has an invalid 'comingSoon' field`);
	}
	return { description, implemented, comingSoon };
};

const isObject = (value: unknown): value is Record<string, unknown> => {
	return value != null && typeof value === 'object' && !Array.isArray(value);
};

export { potentialConnectors };
