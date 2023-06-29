import { createContext, useEffect, useState } from 'react';
import getConnectionFullConnector from '../helpers/getConnectionFullConnector';
import getConnectorLogo from '../helpers/getConnectorLogo';
import getConnectionStatus from '../helpers/getConnectionStatus';
import getConnectionDescription from '../helpers/getConnectionDescription';
import getStorageFileConnections from '../helpers/getStorageFileConnections';
import Connection from '../lib/connections/connection';
import Connector from '../lib/connections/connector';
import API from '../lib/api/api';

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
	const [connectors, setConnectors] = useState([]);
	const [connections, setConnections] = useState([]);
	const [areConnectionsStale, setAreConnectionsStale] = useState(false);

	useEffect(() => {
		const fetchConnectors = async () => {
			const [fetchedConnectors, err] = await api.connectors.find();
			if (err != null) {
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
		const fetchData = async () => {
			const [fetchedConnections, err] = await api.connections.find();
			if (err) {
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
			fetchData();
		}
	}, [areConnectionsStale]);

	return (
		<AppContext.Provider value={{ connectors, connections, setAreConnectionsStale, api, showError, ...delegated }}>
			{children}
		</AppContext.Provider>
	);
};

export { AppProvider, AppContext };
