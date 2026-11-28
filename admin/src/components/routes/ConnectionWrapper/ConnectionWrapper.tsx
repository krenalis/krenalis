import React, { useContext, useEffect } from 'react';
import './ConnectionWrapper.css';
import Flex from '../../base/Flex/Flex';
import StatusDot from '../../base/StatusDot/StatusDot';
import ConnectionContext from '../../../context/ConnectionContext';
import AppContext from '../../../context/AppContext';
import { Outlet } from 'react-router-dom';
import ConnectionTabs from './ConnectionTabs';
import { useConnection } from './useConnection';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaSpinner from '@awesome.me/webawesome/dist/react/spinner/index.js';

const ConnectionWrapper = () => {
	const { setTitle } = useContext(AppContext);

	const { isLoading, hasFailed, retry, connection, setConnection } = useConnection();

	useEffect(() => {
		if (isLoading || hasFailed) {
			setTitle('');
			return;
		}
		const roleLabel = connection.role === 'Source' ? 'Sources' : 'Destinations';
		setTitle(
			<Flex alignItems='center' gap={10}>
				<div className='connection-wrapper__name'>{`Connections / ${roleLabel} / ${connection.name}`}</div>
				<StatusDot status={connection.status} />
			</Flex>,
		);
	}, [isLoading, hasFailed, connection, setTitle]);

	if (hasFailed) {
		return <WaButton onClick={retry}>Retry</WaButton>;
	}
	if (isLoading) {
		return (
			<WaSpinner
				style={
					{
						display: 'block',
						position: 'relative',
						top: '50px',
						margin: 'auto',
						fontSize: '3rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			/>
		);
	}
	return (
		<ConnectionContext.Provider value={{ connection, setConnection }}>
			<div className='connection-wrapper'>
				<ConnectionTabs connection={connection} />
				<div className='route-content route-content--connection'>
					<Outlet />
				</div>
			</div>
		</ConnectionContext.Provider>
	);
};

export default ConnectionWrapper;
