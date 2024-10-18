import React, { useState, useContext, useEffect, useLayoutEffect, ReactNode } from 'react';
import './ConnectionsMap.css';
import Arrow from '../../base/Arrow/Arrow';
import { getConnectionsBlocks } from './ConnectionsMap.helpers';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';

const ConnectionsMap = () => {
	const [databaseArrows, setDatabaseArrows] = useState<ReactNode>([]);

	const { connections, setTitle, workspaces, selectedWorkspace } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Connections');
	}, []);

	useEffect(() => {
		// Must wait for the map to be painted and styled before proceding with
		// the render of the database's arrow.
		setTimeout(() => {
			setDatabaseArrows(
				<>
					<Arrow start='central-logo' end='users-database' startAnchor='bottom' endAnchor='top' />
					<Arrow start='central-logo' end='events-database' startAnchor='bottom' endAnchor='top' />
				</>,
			);
		}, 0);
	}, []);

	const newConnectionID = Number(new URL(document.location.href).searchParams.get('newConnection'));
	const sources: TransformedConnection[] = [];
	const destinations: TransformedConnection[] = [];
	for (const c of connections) {
		if (c.role === 'Source') sources.push(c);
		if (c.role === 'Destination') destinations.push(c);
	}
	const sourcesBlocks = getConnectionsBlocks(sources, newConnectionID);
	const destinationsBlocks = getConnectionsBlocks(destinations, newConnectionID);

	const warehouseMode = workspaces.find((w) => w.ID === selectedWorkspace).WarehouseMode;

	return (
		<div className='connections-map'>
			<div className='route-content'>
				<div className='connections-map__content'>
					<div className='connections-map__buttons'>
						<Link path={`connectors?role=Source`}>
							<SlButton className='connections-map__add-source' variant='text'>
								<SlIcon slot='suffix' name='plus-circle' />
								Add a new source
							</SlButton>
						</Link>
						<Link path={`connectors?role=Destination`}>
							<SlButton className='connections-map__add-destination' variant='text'>
								<SlIcon slot='suffix' name='plus-circle' />
								Add a new destination
							</SlButton>
						</Link>
					</div>
					<div className='connections-map__map'>
						<div className='connections-map__sources'>{sourcesBlocks}</div>
						<div className='connections-map__main'>
							<div className='connections-map__central-logo' id='central-logo'>
								CDP
							</div>
							<div className='connections-map__databases'>
								<Link path='users'>
									<div
										className='connections-map__database connections-map__database--users'
										id='users-database'
									>
										{warehouseMode === 'Normal' ? (
											<SlTooltip content='The warehouse is in Normal mode (full read and write access)'>
												<SlIcon name='database' />
											</SlTooltip>
										) : warehouseMode === 'Inspection' ? (
											<SlTooltip content='The warehouse is in Inspection mode (read-only for data inspection)'>
												<SlIcon name='database-lock' />
											</SlTooltip>
										) : (
											<SlTooltip
												content='The warehouse is in Maintenance mode (init and alter schema
											operations only)'
											>
												<SlIcon name='database-gear' />
											</SlTooltip>
										)}
										<div className='connections-map__database-name'>User Profiles</div>
									</div>
								</Link>
								<div
									className='connections-map__database connections-map__database--events'
									id='events-database'
								>
									<SlIcon name='database' />
									<div className='connections-map__database-name'>Events</div>
								</div>
							</div>
							{databaseArrows}
						</div>
						<div className='connections-map__destinations'>{destinationsBlocks}</div>
					</div>
				</div>
			</div>
		</div>
	);
};

export default ConnectionsMap;
