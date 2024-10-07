import React from 'react';
import { Outlet } from 'react-router-dom';
import './WorkspacesWrapper.css';

const WorkspacesWrapper = () => {
	return (
		<div className='workspaces'>
			<div className='workspaces__content'>
				<Outlet />
			</div>
		</div>
	);
};

export { WorkspacesWrapper };
