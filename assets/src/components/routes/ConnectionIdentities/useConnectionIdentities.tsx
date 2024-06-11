import React, { ReactNode, useContext, useEffect, useState, useMemo } from 'react';
import AppContext from '../../../context/AppContext';
import { ConnectionIdentitiesResponse } from '../../../lib/api/types/responses';
import { UnprocessableError } from '../../../lib/api/errors';
import ConnectionContext from '../../../context/ConnectionContext';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { UserIdentity } from '../../../lib/api/types/user';

const useConnectionIdentities = () => {
	const [identities, setIdentities] = useState<UserIdentity[]>();
	const [isLoading, setIsLoading] = useState<boolean>(true);

	const { api, handleError, redirect } = useContext(AppContext);
	const { connection } = useContext(ConnectionContext);

	useEffect(() => {
		const fetchIdentities = async () => {
			setIsLoading(true);
			// Fetch the connection's identities.
			let identitiesResponse: ConnectionIdentitiesResponse;
			try {
				identitiesResponse = await api.workspaces.connections.identities(connection.id, 0, 1000);
			} catch (err) {
				setTimeout(() => setIsLoading(false), 200);
				if (err instanceof UnprocessableError) {
					if (err.code === 'NoWarehouse') {
						handleError('The workspace is not connected to any data warehouse');
						return;
					}
					if (err.code === 'DataWarehouseFailed') {
						handleError('An error occurred with the data warehouse');
						return;
					}
				}
				handleError(err);
				return;
			}
			setIdentities(identitiesResponse.identities);
			setTimeout(() => setIsLoading(false), 200);
			return;
		};

		if (!connection.hasIdentities) {
			redirect('connections');
		}

		fetchIdentities();
	}, []);

	const { identityProperties, identitiesRows } = useMemo(() => {
		if (identities == null || identities.length === 0) {
			const columns = [];
			const rows = [];
			return { identityProperties: columns, identitiesRows: rows };
		}

		const columns: GridColumn[] = [
			{
				name: 'Last change time',
				type: 'DateTime',
			},
			{
				name: 'Action',
				explanation: 'The ID of the action which imported this identity (TODO: this will be revised)',
			},
			{
				name: identities[0].IdentityId.Label,
			},
		];
		if (connection.hasAnonymousIdentifiers) {
			columns.push({
				name: 'Anonymous Ids',
			});
		}

		const rows: GridRow[] = [];
		for (const identity of identities) {
			const row: GridRow = {
				cells: [identity.LastChangeTime, identity.Action, identity.IdentityId.Value],
				key: identity.IdentityId.Value,
			};
			if (connection.hasAnonymousIdentifiers) {
				const anonymousIds: ReactNode[] = [];
				if (identity.AnonymousIds != null) {
					for (const id of identity.AnonymousIds) {
						anonymousIds.push(<code>{id}</code>);
					}
				}
				row.cells.push(anonymousIds);
			}
			rows.push(row);
		}

		return { identityProperties: columns, identitiesRows: rows };
	}, [identities]);

	return { isLoading, identityProperties, identitiesRows };
};

export { useConnectionIdentities };
