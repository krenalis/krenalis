import React from 'react';
import './NotFound.css';
import { SlButton } from '@shoelace-style/shoelace/dist/react/index.js';
import { NavLink } from 'react-router-dom';

export default class NotFound extends React.Component {
	render() {
		return (
			<div className='NotFound'>
				<div className='routeContent'>
					<div className='box'>
						<div className='icon'></div>
						<div className='title'>404</div>
						<div className='description'>The page you searched for does not exist</div>
						<SlButton className='goBack' size='large' variant='default'>
							Go to connections
							<NavLink to='/admin/connections'></NavLink>
						</SlButton>
					</div>
				</div>
			</div>
		);
	}
}
