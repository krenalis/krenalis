import { useState } from 'react';
import './NavigationWrapper.css';
import Header from '../Header/Header';
import Sidebar from '../Sidebar/Sidebar';
import { NavigationContext } from '../../context/NavigationContext';
import { Outlet } from 'react-router-dom';

const NavigationWrapper = () => {
	let [currentRoute, setCurrentRoute] = useState('');
	let [currentTitle, setCurrentTitle] = useState('');
	let [previousRoute, setPreviousRoute] = useState('');

	return (
		<NavigationContext.Provider
			value={{
				setCurrentRoute: setCurrentRoute,
				setCurrentTitle: setCurrentTitle,
				setPreviousRoute: setPreviousRoute,
			}}
		>
			<div className='NavigationWrapper'>
				<Sidebar route={currentRoute} />
				<Header title={currentTitle} previousRoute={previousRoute} />
				<Outlet />
			</div>
		</NavigationContext.Provider>
	);
};

export default NavigationWrapper;
