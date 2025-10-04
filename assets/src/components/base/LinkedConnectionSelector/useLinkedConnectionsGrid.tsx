import React, { useContext, useMemo } from 'react';
import { GridColumn, GridRow } from '../Grid/Grid.types';
import TransformedConnection from '../../../lib/core/connection';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import LittleLogo from '../LittleLogo/LittleLogo';

const LINKED_CONNECTIONS_COLUMNS: GridColumn[] = [
	{
		name: 'Name',
	},
	{
		name: 'Type',
	},
	{
		name: 'Connector',
	},
	/* See issue https://github.com/meergo/meergo/issues/1255.
	{
		name: 'Health',
	},
*/
	{
		name: '', // the column for the remove button.
	},
];

const useLinkedConnectionsGrid = (
	linkedConnections: Number[] | null,
	setLinkedConnections: React.Dispatch<React.SetStateAction<Number[] | null>>,
	onUnlink: (id: number) => Promise<void>,
	isClickable: boolean,
) => {
	const { connections, redirect } = useContext(AppContext);

	const fullLinkedConnections = useMemo(() => {
		if (linkedConnections == null) {
			return [];
		}
		const fc: TransformedConnection[] = [];
		for (const c of connections) {
			const isLinkedConnection = linkedConnections.findIndex((ec) => ec === c.id) !== -1;
			if (isLinkedConnection) {
				fc.push(c);
			}
		}
		return fc;
	}, [linkedConnections, connections]);

	const unlinkConnection = async (e, idToUnlink: number) => {
		e.stopPropagation();
		let updated: Number[] | null = [];
		for (const id of linkedConnections) {
			if (id !== idToUnlink) {
				updated.push(id);
			}
		}
		if (updated.length === 0) {
			updated = null;
		}
		if (onUnlink) {
			try {
				await onUnlink(idToUnlink);
			} catch (err) {
				return;
			}
		}
		setLinkedConnections(updated);
	};

	const rows: GridRow[] = useMemo(() => {
		const r: GridRow = [];
		if (linkedConnections == null) {
			return r;
		}
		for (const fc of fullLinkedConnections) {
			const nameCell = (
				<div className='linked-connection-grid__name'>
					<LittleLogo icon={fc.connector.icon} />
					{fc.name}
				</div>
			);
			const unlinkButtonCell = (
				<SlButton variant='danger' size='small' onClick={(e) => unlinkConnection(e, fc.id)}>
					Unlink
				</SlButton>
			);
			/* See issue https://github.com/meergo/meergo/issues/1255.
			const healthCell = (
				<div className='linked-connection-grid__status'>
					<StatusDot status={fc.status} />
					{fc.health}
				</div>
			);
*/

			const row: GridRow = {
				cells: [nameCell, fc.connector.type, fc.connector.label, unlinkButtonCell],
				key: String(fc.id),
			};
			if (isClickable) {
				row.onClick = () => redirect(`connections/${fc.id}/actions`);
			}
			r.push(row);
		}
		return r;
	}, [linkedConnections]);

	return { rows: rows, columns: LINKED_CONNECTIONS_COLUMNS };
};

export { useLinkedConnectionsGrid };
