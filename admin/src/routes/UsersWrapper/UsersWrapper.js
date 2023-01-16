import { useEffect, useContext } from 'react';
import './UsersWrapper.css';
import { NavigationContext } from '../../context/NavigationContext';
import { Outlet } from 'react-router-dom';

const UsersWrapper = () => {
	let { setCurrentRoute } = useContext(NavigationContext);

	useEffect(() => {
		setCurrentRoute('users');
	}, []);

	return (
		<div className='UsersWrapper'>
			<Outlet />
		</div>
	);
};

export default UsersWrapper;
