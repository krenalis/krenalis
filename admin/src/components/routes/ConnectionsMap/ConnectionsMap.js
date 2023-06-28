import { useContext } from 'react';
import './ConnectionsMap.css';
import Arrow from '../../common/Arrow/Arrow';
import useConnectionsBlocks from './useConnectionsBlocks';
import useConnectionsArrows from './useConnectionsArrows';
import { splitConnectionsByRole } from '../../../lib/connections/helpers';
import { AppContext } from '../../../providers/AppProvider';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionsMap = () => {
	const { redirect, connections, setTitle } = useContext(AppContext);

	setTitle('Connections');

	const newConnectionID = Number(new URL(document.location).searchParams.get('newConnection'));
	const splittedByRole = splitConnectionsByRole(connections);
	const sourcesBlocks = useConnectionsBlocks(splittedByRole.sources, newConnectionID);
	const sourcesArrows = useConnectionsArrows(splittedByRole.sources, newConnectionID);
	const destinationsBlocks = useConnectionsBlocks(splittedByRole.destinations, newConnectionID);
	const destinationsArrows = useConnectionsArrows(splittedByRole.destinations, newConnectionID);

	const onAddNewSourceClick = () => {
		return redirect(`connectors?role=Source`);
	};

	const onAddNewDestinationClick = () => {
		return redirect(`connectors?role=Destination`);
	};

	const onUsersDatabaseClick = () => {
		return redirect(`users`);
	};

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
