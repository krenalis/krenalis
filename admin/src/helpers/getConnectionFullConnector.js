const getConnectionFullConnector = (connectorID, connectors) => {
	return connectors.find((c) => c.id === connectorID);
};

export default getConnectionFullConnector;
