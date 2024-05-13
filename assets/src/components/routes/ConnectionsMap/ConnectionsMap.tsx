import React, { useState, useContext, useEffect, useLayoutEffect, ReactNode } from 'react';
import './ConnectionsMap.css';
import Arrow from '../../shared/Arrow/Arrow';
import { getConnectionsBlocks } from './ConnectionsMap.helpers';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import { Link } from '../../shared/Link/Link';

const ConnectionsMap = () => {
	const [databaseArrows, setDatabaseArrows] = useState<ReactNode>([]);

	const { connections, setTitle } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Connections');
	}, []);

	useEffect(() => {
		// Must wait for the map to be painted and styled before proceding with
		// the render of the database's arrow.
		setTimeout(() => {
			setDatabaseArrows(
				<>
					<Arrow start='centralLogo' end='usersDatabase' startAnchor='bottom' endAnchor='top' />
					<Arrow start='centralLogo' end='eventsDatabase' startAnchor='bottom' endAnchor='top' />
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

	return (
		<div className='connectionsMap'>
			<div className='routeContent'>
				<div className='buttons'>
					<Link path={`connectors?role=Source`}>
						<SlButton className='addSource' variant='text'>
							<SlIcon slot='suffix' name='plus-circle' />
							Add a new source
						</SlButton>
					</Link>
					<Link path={`connectors?role=Destination`}>
						<SlButton className='addDestination' variant='text'>
							<SlIcon slot='suffix' name='plus-circle' />
							Add a new destination
						</SlButton>
					</Link>
				</div>
				<div className='map'>
					<div className='sources'>{sourcesBlocks}</div>
					<div className='main'>
						<div className='centralLogo' id='centralLogo'>
							CDP
						</div>
						<div className='databases'>
							<Link path='users'>
								<div className='database users' id='usersDatabase'>
									<SlIcon name='database' />
									<div className='name'>Users</div>
								</div>
							</Link>
							<div className='database events' id='eventsDatabase'>
								<SlIcon name='database' />
								<div className='name'>Events</div>
							</div>
						</div>
						{databaseArrows}
					</div>
					<div className='destinations'>{destinationsBlocks}</div>
				</div>
			</div>
		</div>
	);
};

export default ConnectionsMap;
