import React, { useContext } from 'react';
import './NotFound.css';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';

const NotFound = () => {
	const { redirect } = useContext(AppContext);

	const onGoToConnectionsClick = () => {
		redirect('connections');
	};

	return (
		<div className='notFound'>
			<div className='routeContent'>
				<div className='box'>
					<div className='icon'></div>
					<div className='title'>404</div>
					<div className='description'>The page you searched for does not exist</div>
					<SlButton className='goBack' size='large' variant='default' onClick={onGoToConnectionsClick}>
						Go to connections
					</SlButton>
				</div>
			</div>
		</div>
	);
};

export default NotFound;
