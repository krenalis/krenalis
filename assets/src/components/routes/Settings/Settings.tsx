import React, { useContext, useLayoutEffect, ReactNode } from 'react';
import './Settings.css';
import ListTile from '../../shared/ListTile/ListTile';
import { Outlet, useLocation } from 'react-router-dom';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import appContext from '../../../context/AppContext';
import { Link } from '../../shared/Link/Link';

const Settings = () => {
	const { setTitle } = useContext(appContext);

	const location = useLocation();

	useLayoutEffect(() => setTitle('Workspace settings'));

	let content: ReactNode;

	if (location.pathname.endsWith('settings')) {
		content = (
			<div className='settings'>
				<p className='settings__title'>Workspace settings</p>
				<Link path='settings/general'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='person-workspace' />}
						name={'General'}
						description='Update your workspace name and privacy region or delete it'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='settings/data-warehouse'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='database' />}
						name={'Data Warehouse'}
						description='Connect a data warehouse to store the users and events or update its current configuration'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='settings/identifiers'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='person-check' />}
						name={'Identifiers'}
						description='Modify the identifiers used to resolve the identity of the users'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
			</div>
		);
	} else {
		content = <Outlet />;
	}

	return <div className='settings__content'>{content}</div>;
};

export default Settings;
