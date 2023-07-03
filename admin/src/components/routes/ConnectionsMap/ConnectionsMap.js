import { useState, useContext, useEffect, useLayoutEffect } from 'react';
import './ConnectionsMap.css';
import Arrow from '../../common/Arrow/Arrow';
import { getConnectionsBlocks } from './ConnectionsMap.helpers';
import { AppContext } from '../../../providers/AppProvider';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionsMap = () => {
	const [databaseArrows, setDatabaseArrows] = useState([]);

	const { redirect, connections, setTitle } = useContext(AppContext);

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
				</>
			);
		}, 0);
	}, []);

	const onAddNewSourceClick = () => {
		return redirect(`connectors?role=Source`);
	};

	const onAddNewDestinationClick = () => {
		return redirect(`connectors?role=Destination`);
	};

	const onUsersDatabaseClick = () => {
		return redirect(`users`);
	};

	const newConnectionID = Number(new URL(document.location).searchParams.get('newConnection'));
	const sources = [];
	const destinations = [];
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
					<SlButton className='addSource' variant='text' onClick={onAddNewSourceClick}>
						<SlIcon slot='suffix' name='plus-circle' />
						Add a new source
					</SlButton>
					<SlButton className='addDestination' variant='text' onClick={onAddNewDestinationClick}>
						<SlIcon slot='suffix' name='plus-circle' />
						Add a new destination
					</SlButton>
				</div>
				<div className='map'>
					<div className='sources'>{sourcesBlocks}</div>
					<div className='main'>
						<div className='centralLogo' id='centralLogo'>
							CDP
						</div>
						<div className='databases'>
							<div className='database users' id='usersDatabase' onClick={onUsersDatabaseClick}>
								<SlIcon name='database' />
								<div className='name'>Users</div>
							</div>
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
