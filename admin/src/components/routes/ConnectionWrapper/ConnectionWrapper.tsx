import React, { useContext, useEffect } from 'react';
import './ConnectionWrapper.css';
import Flex from '../../base/Flex/Flex';
import StatusDot from '../../base/StatusDot/StatusDot';
import ConnectionContext from '../../../context/ConnectionContext';
import AppContext from '../../../context/AppContext';
import { Outlet } from 'react-router-dom';
import ConnectionTabs from './ConnectionTabs';
import { useConnection } from './useConnection';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';

const ConnectionWrapper = () => {
	const { setTitle } = useContext(AppContext);

	const { isLoading, connection } = useConnection();

	useEffect(() => {
		if (isLoading) {
			setTitle('');
			return;
		}
		setTitle(
			<Flex alignItems='center' gap={10}>
				<div className='connection-wrapper__name'>{connection.name}</div>
				<StatusDot status={connection.status} />
			</Flex>,
		);
	}, [isLoading, connection]);

	if (isLoading) {
		return (
			<SlSpinner
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
		<ConnectionContext.Provider value={{ connection }}>
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
