import { createContext, useState, useContext, useEffect } from 'react';
import { AppContext } from '../providers/AppProvider';
import { NotFoundError } from '../lib/api/errors';
import { Outlet } from 'react-router-dom';

const defaultConnectionContext = {
	connection: {},
};

const ConnectionContext = createContext(defaultConnectionContext);

const ConnectionProvider = () => {
	const [connection, setConnection] = useState(null);

	const { api, showError, showNotFound, connections } = useContext(AppContext);

	const urlFragments = String(window.location).split('/');
	const fragmentIndex = urlFragments.findIndex((f) => f === 'connections');
	const connectionID = Number(urlFragments[fragmentIndex + 1]);

	useEffect(() => {
		const fetchConnection = async () => {
			const [fetchedConnection, err] = await api.connections.get(connectionID);
			if (err) {
				if (err instanceof NotFoundError) {
					showNotFound();
					return;
				}
				showError(err);
				return;
			}
			const providedConnection = connections.find((c) => c.id === connectionID);
			// enrich the provided connection with the additional fetched data.
			const connection = { ...providedConnection };
			connection.actionTypes = fetchedConnection.ActionTypes;
			connection.actions = fetchedConnection.Actions;
			setConnection(connection);
		};
		fetchConnection();
	}, [connections]);

	if (connection == null) {
		return;
	}

	return (
		<ConnectionContext.Provider value={{ connection }}>
			<Outlet />
		</ConnectionContext.Provider>
	);
};

export { ConnectionProvider, ConnectionContext };
