import { useContext } from 'react';
import './ConnectionsMap.css';
import Arrow from '../Arrow/Arrow';
import { renderConnectionsBlocks, renderConnectionsArrows } from '../../lib/connections/helper';
import { NavigationContext } from '../../context/NavigationContext';
import { ConnectionsContext } from '../../context/ConnectionsContext';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { NavLink } from 'react-router-dom';

const ConnectionsMap = () => {
	let { setCurrentTitle, setPreviousRoute } = useContext(NavigationContext);
	let { connections } = useContext(ConnectionsContext);

	setCurrentTitle('Connections');
	setPreviousRoute('');

	let newConnectionID = Number(new URL(document.location).searchParams.get('newConnection'));
	let sources = [];
	let destinations = [];
	for (let c of connections) {
		if (c.Role === 'Source') sources.push(c);
		if (c.Role === 'Destination') destinations.push(c);
	}
	const sourcesBlocks = renderConnectionsBlocks(sources, newConnectionID);
	const destinationsBlocks = renderConnectionsBlocks(destinations, newConnectionID);
	const sourcesArrows = renderConnectionsArrows(sources, newConnectionID);
	const destinationsArrows = renderConnectionsArrows(destinations, newConnectionID);

	return (
		<div className='ConnectionsMap'>
			<div className='routeContent'>
				<div className='buttons'>
					<SlButton className='addSource' variant='text'>
						<SlIcon slot='suffix' name='plus-circle' />
						Add a new source
						<NavLink to='/admin/connectors?role=Source'></NavLink>
					</SlButton>
					<SlButton className='addDestination' variant='text'>
						<SlIcon slot='suffix' name='plus-circle' />
						Add a new destination
						<NavLink to='/admin/connectors?role=Destination'></NavLink>
					</SlButton>
				</div>
				<div className='map'>
					<div className='sources'>{sourcesBlocks}</div>
					<div className='main'>
						<div className='centralLogo' id='centralLogo'>
							CDP
						</div>
						<div className='databases'>
							<div className='database users' id='usersDatabase'>
								<SlIcon name='database' />
								<div className='name'>Users</div>
								<NavLink to='/admin/users'></NavLink>
							</div>
							<div className='database events' id='eventsDatabase'>
								<SlIcon name='database' />
								<div className='name'>Events</div>
							</div>
						</div>
					</div>
					<div className='destinations'>{destinationsBlocks}</div>
				</div>
			</div>
			<div className='arrows'>
				{sourcesArrows}
				{destinationsArrows}
				<Arrow start='centralLogo' end='usersDatabase' startAnchor='bottom' endAnchor='top' />
				<Arrow start='centralLogo' end='eventsDatabase' startAnchor='bottom' endAnchor='top' />
			</div>
		</div>
	);
};

export default ConnectionsMap;
