import React, { ReactNode, useContext, useEffect, useState, useMemo } from 'react';
import AppContext from '../../../context/AppContext';
import { ConnectionIdentitiesResponse } from '../../../lib/api/types/responses';
import { UnprocessableError } from '../../../lib/api/errors';
import ConnectionContext from '../../../context/ConnectionContext';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { Identity } from '../../../lib/api/types/profile';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Link } from '../../base/Link/Link';

const useConnectionIdentities = () => {
	const [identities, setIdentities] = useState<Identity[]>();
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
					if (err.code === 'WarehouseError') {
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
				name: 'Last update',
				type: 'datetime',
				explanation: 'The last update time on the source.',
			},
			{
				name: connection.connector.getIdentityIDLabel(),
			},
			{
				name: 'Action',
			},
		];
		if (connection.hasAnonymousIdentifiers) {
			columns.splice(2, 0, { name: 'Anonymous IDs' });
		}

		const rows: GridRow[] = [];
		for (const identity of identities) {
			const actionName = connection.actions.find((a) => a.id === identity.action).name;
			const row: GridRow = {
				cells: [
					identity.lastChangeTime,
					identity.id ? (
						identity.id
					) : (
						<span className='connection-identities__anonymous-identity'>
							<SlIcon name='incognito' />
							anonymous
						</span>
					),
					<span className='connection-identities__action'>
						<Link path={`connections/${connection.id}/actions/edit/${identity.action}`}>{actionName}</Link>
					</span>,
				],
				key: identity.id,
			};
			if (connection.hasAnonymousIdentifiers) {
				const anonymousIds: ReactNode[] = [];
				if (identity.anonymousIds != null) {
					for (const id of identity.anonymousIds) {
						anonymousIds.push(<code>{id}</code>);
					}
				}
				row.cells.splice(2, 0, anonymousIds);
			}
			rows.push(row);
		}

		return { identityProperties: columns, identitiesRows: rows };
	}, [identities]);

	return { isLoading, identityProperties, identitiesRows };
};

export { useConnectionIdentities };
