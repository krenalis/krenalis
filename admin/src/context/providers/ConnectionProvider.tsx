import React, { createContext, useState, useContext, useEffect } from 'react';
import { AppContext } from './AppProvider';
import { NotFoundError } from '../../lib/api/errors';
import { Outlet } from 'react-router-dom';
import { Connection } from '../../types/external/connection';
import TransformedConnection from '../../lib/helpers/transformedConnection';

interface ConnectionContextInterface {
	connection: TransformedConnection;
}

const ConnectionContext = createContext<ConnectionContextInterface>({} as ConnectionContextInterface);

const ConnectionProvider = () => {
	const [connection, setConnection] = useState<TransformedConnection>();

	const { api, showError, showNotFound, connections } = useContext(AppContext);

	const urlFragments = String(window.location).split('/');
	const fragmentIndex = urlFragments.findIndex((f) => f === 'connections');
	const connectionID = Number(urlFragments[fragmentIndex + 1]);

	useEffect(() => {
		const fetchConnection = async () => {
			let fetchedConnection: Connection;
			try {
				fetchedConnection = await api.workspaces.connections.get(connectionID);
			} catch (err) {
				if (err instanceof NotFoundError) {
					showNotFound();
					return;
				}
				showError(err);
				return;
			}
			const providedConnection = connections.find((c) => c.id === connectionID);
			if (providedConnection == null) {
				return;
			}
			// enrich the transformed connection with the additional fetched
			// data.
			const connection = Object.assign(providedConnection);
			connection.actionTypes = fetchedConnection.ActionTypes;
			connection.actions = fetchedConnection.Actions;
			setConnection(connection);
		};
		fetchConnection();
	}, [connections]);

	if (connection == null) {
		return null;
	}

	return (
		<ConnectionContext.Provider value={{ connection }}>
			<Outlet />
		</ConnectionContext.Provider>
	);
};

export { ConnectionProvider, ConnectionContext };
