import React from 'react';
import './NotFound.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { Link } from '../../base/Link/Link';

const NotFound = () => {
	return (
		<div className='not-found'>
			<div className='route-content'>
				<div className='not-found__box'>
					<div className='not-found__icon'></div>
					<div className='not-found__title'>404</div>
					<div className='not-found__description'>The page you searched for does not exist</div>
					<Link path='connections'>
						<SlButton className='not-found__go-back' size='large' variant='default'>
							Go to connections
						</SlButton>
					</Link>
				</div>
			</div>
		</div>
	);
};

export default NotFound;
