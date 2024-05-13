import React from 'react';
import './NotFound.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { Link } from '../../shared/Link/Link';

const NotFound = () => {
	return (
		<div className='notFound'>
			<div className='routeContent'>
				<div className='box'>
					<div className='icon'></div>
					<div className='title'>404</div>
					<div className='description'>The page you searched for does not exist</div>
					<Link path='connections'>
						<SlButton className='goBack' size='large' variant='default'>
							Go to connections
						</SlButton>
					</Link>
				</div>
			</div>
		</div>
	);
};

export default NotFound;
