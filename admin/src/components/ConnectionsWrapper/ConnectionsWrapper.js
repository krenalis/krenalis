import { useEffect, useContext } from 'react';
import './ConnectionsWrapper.css';
import { NavigationContext } from '../../context/NavigationContext';
import { Outlet } from 'react-router-dom';

const ConnectionsWrapper = () => {
	let { setCurrentRoute } = useContext(NavigationContext);

	useEffect(() => {
		setCurrentRoute('connections');
	}, []);

	return (
		<div className='ConnectionWrapper'>
			<Outlet />
		</div>
	);
};

export default ConnectionsWrapper;
