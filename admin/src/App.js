import React from 'react';
import { Outlet } from 'react-router-dom';
import './App.css';
import '@shoelace-style/shoelace/dist/themes/light.css';
import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path.js';

setBasePath('https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.0.0-beta.85/dist/');

const App = () => {
	return (
		<div className='App'>
			<Outlet />
		</div>
	);
};

export default App;
