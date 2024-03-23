import React, { createContext, useState, useContext, useEffect } from 'react';
import AppContext from '../AppContext';
import { NotFoundError } from '../../lib/api/errors';
import { Outlet, useParams } from 'react-router-dom';
import { Connection } from '../../types/external/connection';
import TransformedConnection from '../../lib/helpers/transformedConnection';

interface ConnectionContextInterface {
	connection: TransformedConnection;
}

const ConnectionContext = createContext<ConnectionContextInterface>({} as ConnectionContextInterface);

const ConnectionProvider = () => {
	const [connection, setConnection] = useState<TransformedConnection>();

	const { api, handleError, showNotFound, connections } = useContext(AppContext);

	const params = useParams();

	useEffect(() => {
		const fetchConnection = async () => {
			let fetchedConnection: Connection;
			try {
				fetchedConnection = await api.workspaces.connections.get(Number(params.id));
			} catch (err) {
				if (err instanceof NotFoundError) {
					showNotFound();
					return;
				}
				handleError(err);
				return;
			}
			const providedConnection = connections.find((c) => c.id === Number(params.id));
			if (providedConnection == null) {
				return;
			}
			// enrich the transformed connection with the additional
			// fetched data.
			const connection = Object.assign(providedConnection);
			connection.actionTypes = fetchedConnection.ActionTypes;
			connection.actions = fetchedConnection.Actions;
			setConnection(connection);
		};
		fetchConnection();
	}, [connections, params.id]);

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
