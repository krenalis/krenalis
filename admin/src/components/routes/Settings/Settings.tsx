import React, { useContext, useLayoutEffect, ReactNode } from 'react';
import './Settings.css';
import ListTile from '../../base/ListTile/ListTile';
import { Outlet, useLocation } from 'react-router-dom';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import appContext from '../../../context/AppContext';
import { Link } from '../../base/Link/Link';

const Settings = () => {
	const { setTitle } = useContext(appContext);

	const location = useLocation();

	useLayoutEffect(() => {
		if (location.pathname.endsWith('settings')) {
			setTitle('Settings');
		}
	}, [location.pathname, setTitle]);

	let content: ReactNode;

	if (location.pathname.endsWith('settings')) {
		content = (
			<div className='settings__content'>
				<Link path='settings/general'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='list' />}
						name={'General'}
						description='Update your workspace name or delete it'
						showHover={true}
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='settings/data-warehouse'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='database' />}
						name={'Data Warehouse'}
						description='Manage data warehouse mode and settings'
						showHover={true}
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
			</div>
		);
	} else {
		content = <Outlet />;
	}

	return (
		<div className='settings'>
			<div className='route-content'>{content}</div>
		</div>
	);
};

export default Settings;
