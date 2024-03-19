import React, { ReactNode, useContext, useEffect, useState, useMemo } from 'react';
import AppContext from '../../../context/AppContext';
import { ConnectionIdentitiesResponse } from '../../../types/external/api';
import { UnprocessableError } from '../../../lib/api/errors';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import { UserIdentity } from '../../../types/external/user';

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

	const { identityColumns, identitiesRows } = useMemo(() => {
		if (identities == null || identities.length === 0) {
			const columns = [];
			const rows = [];
			return { identityColumns: columns, identitiesRows: rows };
		}

		const isBusinessIdDefined = identities[0].BusinessId !== '';

		const columns: GridColumn[] = [
			{
				name: 'Last update',
				type: 'DateTime',
			},
			{
				name: identities[0].ExternalId.Label,
			},
		];
		if (isBusinessIdDefined) {
			columns.push({
				name: 'Business ID',
				explanation: 'TODO: Description of Business ID',
			});
		}
		if (connection.hasAnonymousIdentifiers) {
			columns.push({
				name: 'Anonymous Ids',
			});
		}

		const rows: GridRow[] = [];
		for (const identity of identities) {
			const row: GridRow = {
				cells: [identity.UpdatedAt, identity.ExternalId.Value],
				key: identity.ExternalId.Value,
			};
			if (isBusinessIdDefined) {
				row.cells.push(identity.BusinessId);
			}
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

		return { identityColumns: columns, identitiesRows: rows };
	}, [identities]);

	return { isLoading, identityColumns, identitiesRows };
};

export { useConnectionIdentities };
