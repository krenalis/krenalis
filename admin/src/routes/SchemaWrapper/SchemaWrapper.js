import { useEffect, useContext } from 'react';
import './SchemaWrapper.css';
import { NavigationContext } from '../../context/NavigationContext';
import { Outlet } from 'react-router-dom';

const SchemaWrapper = () => {
	let { setCurrentRoute } = useContext(NavigationContext);

	useEffect(() => {
		setCurrentRoute('schema');
	}, []);

	return (
		<div className='SchemaWrapper'>
			<Outlet />
		</div>
	);
};

export default SchemaWrapper;
