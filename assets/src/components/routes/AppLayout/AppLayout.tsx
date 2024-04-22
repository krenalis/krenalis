import React, { useContext } from 'react';
import './AppLayout.css';
import AppContext from '../../../context/AppContext';
import Sidebar from '../../layout/Sidebar/Sidebar';
import Header from '../../layout/Header/Header';
import { Outlet } from 'react-router-dom';

const AppLayout = () => {
	const { workspaces, warehouse, selectedWorkspace, setSelectedWorkspace, title, member } = useContext(AppContext);

	return (
		<div className='app'>
			<Sidebar
				workspaces={workspaces}
				warehouse={warehouse}
				selectedWorkspace={selectedWorkspace}
				setSelectedWorkspace={setSelectedWorkspace}
			/>
			<Header title={title} member={member} />
			<Outlet />
		</div>
	);
};

export default AppLayout;
