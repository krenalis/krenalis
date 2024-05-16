import { useState, useContext, useEffect } from 'react';
import AppContext from '../../../context/AppContext';
import { useParams } from 'react-router-dom';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import { Connection } from '../../../types/external/connection';
import { NotFoundError } from '../../../lib/api/errors';

const useConnection = () => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [connection, setConnection] = useState<TransformedConnection>();

	const { api, handleError, showNotFound, connections } = useContext(AppContext);

	const params = useParams();

	useEffect(() => {
		const fetchConnection = async () => {
			const providedConnection = connections.find((c) => c.id === Number(params.id));
			if (providedConnection == null) {
				setIsLoading(false);
				showNotFound();
				return;
			}
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
			// enrich the transformed connection with the additional
			// fetched data.
			const connection = Object.assign(providedConnection);
			connection.actionTypes = fetchedConnection.ActionTypes;
			connection.actions = fetchedConnection.Actions;
			setConnection(connection);
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		fetchConnection();
	}, [connections, params.id]);

	return { isLoading, connection };
};

export { useConnection };
