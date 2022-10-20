import React from 'react';
import { Outlet } from 'react-router-dom';
import './App.css';
import '@shoelace-style/shoelace/dist/themes/light.css';
import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path';

setBasePath('https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.0.0-beta.83/dist/');

export default class App extends React.Component {
	render() {
		return (
			<div className='App'>
				<Outlet />
			</div>
		);
	}
}
