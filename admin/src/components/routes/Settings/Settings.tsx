import React, { useContext, useLayoutEffect } from 'react';
import './Settings.css';
import ListTile from '../../shared/ListTile/ListTile';
import { Outlet, useLocation } from 'react-router-dom';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import appContext from '../../../context/AppContext';

const Settings = () => {
	const { redirect, setTitle } = useContext(appContext);

	const location = useLocation();

	useLayoutEffect(() => setTitle('Workspace settings'));

	const onAnonymousIdentifiersClick = () => {
		redirect('settings/anonymous-identity');
	};

	const onDataWarehouseClick = () => {
		redirect('settings/data-warehouse');
	};

	if (location.pathname.endsWith('settings')) {
		return (
			<div className='settings'>
				<p className='settings__title'>Workspace settings</p>
				<ListTile
					className='settings__setting'
					icon={<SlIcon name='database' />}
					name={'Data Warehouse'}
					description='Connect a data warehouse to store the users and events or update its current configuration'
					onClick={onDataWarehouseClick}
				/>
				<ListTile
					className='settings__setting'
					icon={<SlIcon name='incognito' />}
					name={'Anonymous IDs'}
					description='Modify the anonymous identifiers used to resolve the identity of anonymous users'
					onClick={onAnonymousIdentifiersClick}
				/>
			</div>
		);
	}

	return <Outlet />;
};

export default Settings;
