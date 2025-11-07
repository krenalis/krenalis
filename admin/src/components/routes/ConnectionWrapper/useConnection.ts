import { useState, useContext, useEffect } from 'react';
import AppContext from '../../../context/AppContext';
import { useParams } from 'react-router-dom';
import TransformedConnection from '../../../lib/core/connection';
import { Connection } from '../../../lib/api/types/connection';
import { NotFoundError } from '../../../lib/api/errors';
import { ActionType } from '../../../lib/api/types/action';
import { TransformedEventType } from '../../../lib/core/action';
import { parseFilter } from '../../../utils/filters_parser';

const useConnection = () => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [connection, setConnection] = useState<TransformedConnection>();

	const { api, handleError, showNotFound, connections } = useContext(AppContext);

	const params = useParams();

	useEffect(() => {
		const fetchData = async () => {
			const connectionID = Number(params.id);
			let fetchedConnection: Connection;
			try {
				fetchedConnection = await api.workspaces.connections.get(connectionID);
			} catch (err) {
				if (err instanceof NotFoundError) {
					showNotFound();
					return;
				}
				handleError(err);
				return;
			}
			let actionTypes: ActionType[];
			try {
				actionTypes = await api.workspaces.connections.actionTypes(connectionID);
			} catch (err) {
				handleError(err);
				return;
			}
			const providedConnection = connections.find((c) => c.id === connectionID);
			if (providedConnection == null) {
				return;
			}
			// enrich the transformed connection with the additional
			// fetched data.
			const connection = Object.assign(providedConnection);
			connection.actionTypes = actionTypes;
			connection.actions = fetchedConnection.actions;
			if (fetchedConnection.eventTypes != null) {
				let eventTypes: TransformedEventType[] = [];
				for (const t of fetchedConnection.eventTypes) {
					let tt: TransformedEventType = {
						id: t.id,
						name: t.name,
						description: t.description,
						filter: null,
					};
					if (t.filter) {
						try {
							tt.filter = parseFilter(t.filter);
						} catch (err) {
							handleError(
								`The filter of the event type with ID "${t.id}" is invalid: ${err.message.toLowerCase()}`,
							);
						}
					}
					eventTypes.push(tt);
				}
				connection.eventTypes = eventTypes;
			}
			setConnection(connection);
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		fetchData();
	}, [connections, params.id]);

	return { isLoading, connection };
};

export { useConnection };
