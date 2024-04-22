import React, { useContext, useMemo } from 'react';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import LittleLogo from '../LittleLogo/LittleLogo';
import StatusDot from '../StatusDot/StatusDot';

const EVENT_CONNECTIONS_COLUMNS: GridColumn[] = [
	{
		name: 'Name',
	},
	{
		name: 'Type',
	},
	{
		name: 'Connector',
	},
	{
		name: 'Health',
	},
	{
		name: '', // the column for the remove button.
	},
];

const useEventConnectionsGrid = (
	eventConnections: Number[] | null,
	setEventConnections: React.Dispatch<React.SetStateAction<Number[] | null>>,
	onRemove: (id: number) => Promise<void>,
	isClickable: boolean,
) => {
	const { connections, redirect } = useContext(AppContext);

	const fullEventConnections = useMemo(() => {
		if (eventConnections == null) {
			return [];
		}
		const fc: TransformedConnection[] = [];
		for (const c of connections) {
			const isEventConnection = eventConnections.findIndex((ec) => ec === c.id) !== -1;
			if (isEventConnection) {
				fc.push(c);
			}
		}
		return fc;
	}, [eventConnections, connections]);

	const removeConnection = async (e, idToRemove: number) => {
		e.stopPropagation();
		let updated: Number[] | null = [];
		for (const id of eventConnections) {
			if (id !== idToRemove) {
				updated.push(id);
			}
		}
		if (updated.length === 0) {
			updated = null;
		}
		if (onRemove) {
			try {
				await onRemove(idToRemove);
			} catch (err) {
				return;
			}
		}
		setEventConnections(updated);
	};

	const rows: GridRow[] = useMemo(() => {
		const r: GridRow = [];
		if (eventConnections == null) {
			return r;
		}
		for (const fc of fullEventConnections) {
			const nameCell = (
				<div className='event-connection-grid__name'>
					<LittleLogo icon={fc.connector.icon} />
					{fc.name}
				</div>
			);
			const removeButtonCell = (
				<SlButton variant='danger' size='small' onClick={(e) => removeConnection(e, fc.id)}>
					Remove
				</SlButton>
			);
			const healthCell = (
				<div className='event-connection-grid__status'>
					<StatusDot status={fc.status} />
					{fc.health}
				</div>
			);

			const row: GridRow = {
				cells: [nameCell, fc.type, fc.connector.name, healthCell, removeButtonCell],
				key: String(fc.id),
			};
			if (isClickable) {
				row.onClick = () => redirect(`connections/${fc.id}/actions`);
			}
			r.push(row);
		}
		return r;
	}, [eventConnections]);

	return { rows: rows, columns: EVENT_CONNECTIONS_COLUMNS };
};

export { useEventConnectionsGrid };
