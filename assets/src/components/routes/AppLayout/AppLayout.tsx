import React, { useContext } from 'react';
import './AppLayout.css';
import AppContext from '../../../context/AppContext';
import Sidebar from '../../base/Sidebar/Sidebar';
import Header from '../../base/Header/Header';
import { Outlet } from 'react-router-dom';

const AppLayout = () => {
	const { workspaces, warehouse, selectedWorkspace, setSelectedWorkspace, title, member } = useContext(AppContext);

	return (
		<div className='app'>
			<Header title={title} member={member} />
			<Sidebar
				workspaces={workspaces}
				warehouse={warehouse}
				selectedWorkspace={selectedWorkspace}
				setSelectedWorkspace={setSelectedWorkspace}
			/>
			<Outlet />
		</div>
	);
};

export default AppLayout;
