import { createContext, useEffect, useState, useRef } from 'react';
import getConnectorLogo from '../../components/helpers/getConnectorLogo';
import { getConnectionFullConnector } from '../../lib/helpers/connection';
import { getConnectionStatus } from '../../lib/helpers/connection';
import { getConnectionDescription } from '../../lib/helpers/connection';
import { getStorageFileConnections } from '../../lib/helpers/connection';
import Connection from '../../lib/helpers/connection';
import Connector from '../../lib/helpers/connector';
import API from '../../lib/api/api';
import { SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

const defaultAppContext = {
	setTitle: () => {},
	api: new API(),
	showError: () => {},
	showStatus: () => {},
	showNotFound: () => {},
	redirect: () => {},
	account: 0,
	connectors: [],
	connections: [],
	setAreConnectionsStale: () => {},
};

const AppContext = createContext(defaultAppContext);

const AppProvider = ({ api, showError, children, ...delegated }) => {
	const [isLoading, setIsLoading] = useState(false);
	const [connectors, setConnectors] = useState(null);
	const [connections, setConnections] = useState(null);
	const [areConnectionsStale, setAreConnectionsStale] = useState(false);

	const isLoadingTimeoutID = useRef(0);

	useEffect(() => {
		isLoadingTimeoutID.current = setTimeout(() => setIsLoading(true), 100);
	}, []);

	useEffect(() => {
		const fetchConnectors = async () => {
			let fetchedConnectors;
			try {
				fetchedConnectors = await api.connectors.find();
			} catch (err) {
				showError(err);
				return;
			}
			const connectors = Connector.toConnectorsArray(fetchedConnectors);
			for (const connector of connectors) {
				connector.logo = getConnectorLogo(connector.icon);
			}
			setConnectors(connectors);
			setAreConnectionsStale(true);
		};
		fetchConnectors();
	}, []);

	useEffect(() => {
		const fetchConnections = async () => {
			let fetchedConnections;
			try {
				fetchedConnections = await api.connections.find();
			} catch (err) {
				setConnections([]);
				showError(err);
				return;
			}
			const connections = Connection.toConnectionsArray(fetchedConnections);
			for (const connection of connections) {
				connection.connector = getConnectionFullConnector(connection.connector, connectors);
				connection.logo = getConnectorLogo(connection.connector.icon);
				connection.status = getConnectionStatus(connection);
				connection.description = getConnectionDescription(connection);
				if (connection.isStorage) {
					connection.linkedFiles = getStorageFileConnections(connection.id, connections);
				}
			}
			setConnections(connections);
			setAreConnectionsStale(false);
		};
		if (areConnectionsStale) {
			fetchConnections();
		}
	}, [areConnectionsStale]);

	useEffect(() => {
		if (connectors == null || connections == null) {
			return;
		}
		if (isLoading) {
			setTimeout(() => setIsLoading(false), 300);
		} else {
			clearTimeout(isLoadingTimeoutID.current);
		}
	}, [connectors, connections]);

	const canRender = connectors != null && connections != null;
	return (
		<AppContext.Provider value={{ connectors, connections, setAreConnectionsStale, api, showError, ...delegated }}>
			{isLoading ? (
				<SlSpinner
					className='globalSpinner'
					style={{
						fontSize: '5rem',
						'--track-width': '6px',
					}}
				/>
			) : (
				canRender && children
			)}
		</AppContext.Provider>
	);
};

export { AppProvider, AppContext };
