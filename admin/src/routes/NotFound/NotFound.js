import React from 'react';
import './NotFound.css';

export default class NotFound extends React.Component {
	render() {
		return (
			<div className='NotFound'>
				<div className='routeContent'>
					<div className='box'>
						<div className='icon'></div>
						<div className='title'>404</div>
						<div className='description'>The page you searched for does not exist</div>
					</div>
				</div>
			</div>
		);
	}
}
