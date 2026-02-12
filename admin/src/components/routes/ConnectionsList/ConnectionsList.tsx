import React, { useState, useEffect, useContext, ReactNode } from 'react';
import './ConnectionsList.css';
import IconWrapper from '../../base/IconWrapper/IconWrapper';
import Grid from '../../base/Grid/Grid';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { ConnectionRole } from '../../../lib/api/types/connection';
import TransformedConnection from '../../../lib/core/connection';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { Link } from '../../base/Link/Link';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

const ConnectionsList = () => {
	const [connectionsRows, setConnectionsRows] = useState<GridRow[]>();
	const [connectionsColumns, setConnectionColumns] = useState<GridColumn[]>([]);
	const [role, setRole] = useState<ConnectionRole>();

	const { redirect, connections, setTitle } = useContext(AppContext);

	useEffect(() => {
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
			/* See issue https://github.com/meergo/meergo/issues/1255.
			{
				name: 'Health',
			},
*/
			{
				name: 'Pipelines',
				alignment: 'center',
			},
		];

		if (roleConnections.length === 0) {
			setConnectionsRows([]);
			setConnectionColumns(columns);
			return;
		}

		roleConnections.sort((a, b) => {
			if (a.name < b.name) {
				return -1;
			} else if (a.name > b.name) {
				return 1;
			} else {
				// The names are equal, compare the IDs.
				return a.id < b.id ? -1 : 1;
			}
		});

		const hasEventConnections = roleConnections.findIndex((c) => c.linkedConnections != null) !== -1;
		if (hasEventConnections) {
			columns.push({
				name: `${role === 'Source' ? 'Event destinations' : 'Event sources'}`,
				alignment: 'left',
			});
		}

		const rows: GridRow[] = [];
		for (const c of roleConnections) {
			const cells = [
				<div className='connections-list__name-cell'>
					<LittleLogo code={c.connector.code} path={CONNECTORS_ASSETS_PATH} /> {c.name}
				</div>,
				c.connector.type === 'FileStorage' ? 'File storage' : c.connector.type,
				c.connector.label,
				/* See issue https://github.com/meergo/meergo/issues/1255.
				<div className='connections-list__status-cell'>
					<StatusDot status={c.status} />
					<div>{c.status.text}</div>
				</div>,
				*/
				c.pipelines.length,
			];
			if (hasEventConnections) {
				if (c.linkedConnections != null) {
					const fullEventConnections: TransformedConnection[] = [];
					for (const id of c.linkedConnections) {
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
						connectionLogos.push(
							<LittleLogo key={String(ec.id)} code={ec.connector.code} path={CONNECTORS_ASSETS_PATH} />,
						);
					}
					cells.push(<div className='connections-list__event-connections-cell'>{connectionLogos}</div>);
				} else {
					cells.push('-');
				}
			}
			rows.push({
				cells: cells,
				id: String(c.id),
				onClick: () => {
					redirect(`connections/${c.id}/pipelines`);
				},
			});
		}
		setConnectionsRows(rows);
		setConnectionColumns(columns);
	}, [connections, role]);

	useEffect(() => {
		if (role) {
			const section = role === 'Source' ? 'Sources' : 'Destinations';
			setTitle(`Connections / ${section}`);
		}
	}, [role, setTitle]);

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
		<div className='connections-list'>
			<div className='route-content'>
				{connectionsRows.length === 0 ? (
					<div className='connections-list__no-connection'>
						{role === 'Source' ? (
							<IconWrapper name='arrow-down-right-square' size={40} />
						) : (
							<IconWrapper name='arrow-up-right-square' size={40} />
						)}
						<div className='connections-list__no-connection-text'>
							You don't have any {role?.toLowerCase()} installed
						</div>
						<Link path={`connectors?role=${role}`}>
							<SlButton variant='primary'>Add a new {role?.toLowerCase()}...</SlButton>
						</Link>
					</div>
				) : (
					<>
						<Link path={`connectors?role=${role}`}>
							<SlButton variant='text' className='connections-list__add-new-connection'>
								<SlIcon slot='suffix' name='plus-circle' />
								Add a new {role?.toLowerCase()}
							</SlButton>
						</Link>
						<Grid columns={connectionsColumns} rows={connectionsRows} />
						<div className='grid-learn-more'>
							Learn more about{' '}
							<a href='https://www.meergo.com/docs/ref/admin/integrations' target='_blank'>
								integrations
							</a>
						</div>
					</>
				)}
			</div>
		</div>
	);
};

export default ConnectionsList;
