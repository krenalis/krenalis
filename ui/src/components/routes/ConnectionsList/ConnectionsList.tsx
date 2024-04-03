import React, { useState, useEffect, useContext, ReactNode } from 'react';
import './ConnectionsList.css';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';
import Grid from '../../shared/Grid/Grid';
import StatusDot from '../../shared/StatusDot/StatusDot';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import { ConnectionRole } from '../../../types/external/connection';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import LittleLogo from '../../shared/LittleLogo/LittleLogo';

const ConnectionsList = () => {
	const [connectionsRows, setConnectionsRows] = useState<GridRow[]>([]);
	const [connectionsColumns, setConnectionColumns] = useState<GridColumn[]>([]);
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

		const columns: GridColumn[] = [
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

		if (roleConnections.length === 0) {
			setConnectionsRows([]);
			setConnectionColumns(columns);
			return;
		}

		const hasEventConnections = roleConnections.findIndex((c) => c.eventConnections != null) !== -1;
		if (hasEventConnections) {
			columns.push({
				name: `${role === 'Source' ? 'Event destinations' : 'Event sources'}`,
				alignment: 'left',
			});
		}

		const rows: GridRow[] = [];
		for (const c of roleConnections) {
			const cells = [
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
			];
			if (hasEventConnections) {
				if (c.eventConnections != null) {
					const fullEventConnections: TransformedConnection[] = [];
					for (const id of c.eventConnections) {
						const fullConnection = connections.find((c) => c.id === id);
						fullEventConnections.push(fullConnection);
					}
					fullEventConnections.sort((a, b) => {
						if (a.name < b.name) {
							return -1;
						} else if (a.name > b.name) {
							return 1;
						}
						return 0;
					});
					const connectionLogos: ReactNode[] = [];
					for (const ec of fullEventConnections) {
						connectionLogos.push(<LittleLogo key={String(ec.id)} icon={ec.connector.icon} />);
					}
					cells.push(<div className='connectionEventConnectionsCell'>{connectionLogos}</div>);
				} else {
					cells.push('-');
				}
			}
			rows.push({
				cells: cells,
				onClick: () => {
					redirect(`connections/${c.id}/actions`);
				},
			});
		}
		setConnectionsRows(rows);
		setConnectionColumns(columns);
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
						<Grid columns={connectionsColumns} rows={connectionsRows} />
					</>
				)}
			</div>
		</div>
	);
};

export default ConnectionsList;
