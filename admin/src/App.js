import React from 'react';
import { Outlet } from 'react-router-dom';

import './App.css';

export default class App extends React.Component {
	render() {
		return (
			<div className='App'>
				<Outlet />
			</div>
		);
	}
}
