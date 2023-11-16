import React, { useState, useEffect, useContext } from 'react';
import './ConnectionsList.css';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';
import Grid from '../../shared/Grid/Grid';
import StatusDot from '../../shared/StatusDot/StatusDot';
import { AppContext } from '../../../context/providers/AppProvider';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import { ConnectionRole } from '../../../types/external/connection';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import TransformedConnection from '../../../lib/helpers/transformedConnection';

const GRID_COLUMNS: GridColumn[] = [
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
		name: 'Actions',
		alignment: 'center',
	},
];

const ConnectionsList = () => {
	const [connectionsRows, setConnectionsRows] = useState<GridRow[]>([]);
	const [role, setRole] = useState<ConnectionRole>();

	const { redirect, connections, setTitle } = useContext(AppContext);

	useEffect(() => {
		setTitle(`${role}s`);
		const roleConnections: TransformedConnection[] = [];
		for (const c of connections) {
			if (c.role === role) {
				roleConnections.push(c);
			}
		}
		if (roleConnections.length === 0) {
			setConnectionsRows([]);
			return;
		}
		const rows: GridRow[] = [];
		for (const c of roleConnections) {
			rows.push({
				cells: [
					<div className='connectionNameCell'>
						{getConnectorLogo(c.connector.icon)} {c.name}
					</div>,
					c.type,
					c.connector.name,
					<div className='connectionStatusCell'>
						<StatusDot status={c.status} />
						{c.status.text}
					</div>,
					c.actionsCount,
				],
				onClick: () => {
					redirect(`connections/${c.id}/actions`);
				},
			});
		}
		setConnectionsRows(rows);
	}, [connections, role]);

	const path = window.location.pathname;
	const splitted = path.split('/');
	const urlRole = splitted[splitted.length - 1];
	let r: ConnectionRole;
	if (urlRole === 'sources') {
		r = 'Source';
	} else {
		r = 'Destination';
	}
	if (r !== role) {
		setRole(r);
	}

	if (connectionsRows == null) {
		return null;
	}

	return (
		<div className='connectionsList'>
			<div className='routeContent'>
				{connectionsRows.length === 0 ? (
					<div className='noConnection'>
						<IconWrapper name={role === 'Source' ? 'file-arrow-down' : 'file-arrow-up'} size={40} />
						<div className='noConnectionText'>You don't have any {role?.toLowerCase()} installed</div>
						<SlButton
							variant='primary'
							onClick={() => {
								redirect(`connectors?role=${role}`);
							}}
						>
							Add a {role?.toLowerCase()}...
						</SlButton>
					</div>
				) : (
					<>
						<SlButton
							variant='text'
							className='addNewConnection'
							onClick={() => {
								redirect(`connectors?role=${role}`);
							}}
						>
							<SlIcon slot='suffix' name='plus-circle' />
							Add a new {role?.toLowerCase()}
						</SlButton>
						<Grid columns={GRID_COLUMNS} rows={connectionsRows} />
					</>
				)}
			</div>
		</div>
	);
};

export default ConnectionsList;
