import { useState } from 'react';
import './NavigationWrapper.css';
import Sidebar from '../../components/Sidebar/Sidebar';
import { NavigationContext } from '../../context/NavigationContext';
import { Outlet } from 'react-router-dom';

const NavigationWrapper = () => {
	let [currentRoute, setCurrentRoute] = useState('');

	return (
		<NavigationContext.Provider value={{ setCurrentRoute: setCurrentRoute }}>
			<div className='NavigationWrapper'>
				<Sidebar currentRoute={currentRoute} />
				<Outlet />
			</div>
		</NavigationContext.Provider>
	);
};

export default NavigationWrapper;
