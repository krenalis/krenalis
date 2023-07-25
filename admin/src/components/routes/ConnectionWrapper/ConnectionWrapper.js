import { useContext, useEffect } from 'react';
import './ConnectionWrapper.css';
import Flex from '../../shared/Flex/Flex';
import StatusDot from '../../shared/StatusDot/StatusDot';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { AppContext } from '../../../context/providers/AppProvider';
import { Outlet } from 'react-router-dom';
import ConnectionTabs from './ConnectionTabs';

const ConnectionWrapper = () => {
	const { connection } = useContext(ConnectionContext);
	const { setTitle } = useContext(AppContext);

	useEffect(() => {
		setTitle(
			<Flex alignItems='baseline' gap='10px'>
				<span style={{ position: 'relative', top: '3px' }}>{connection.logo}</span>
				<div className='text'>{connection.name}</div>
				<StatusDot status={connection.status} />
			</Flex>
		);
	}, [connection]);

	return (
		<div className='connectionWrapper'>
			<ConnectionTabs connection={connection} />
			<div className='routeContent connection'>
				<Outlet />
			</div>
		</div>
	);
};

export default ConnectionWrapper;
