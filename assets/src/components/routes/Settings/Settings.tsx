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

	useLayoutEffect(() => setTitle('Customization'));

	let content: ReactNode;

	if (location.pathname.endsWith('settings')) {
		content = (
			<div className='settings__content'>
				<Link path='settings/general'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='sliders2' />}
						name={'General'}
						description='Update your workspace name or delete it'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='schema'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='bookmark-check' />}
						name={'Customer Model'}
						description='Define and manage the schema of your customer data used to model and understand your customers'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='settings/identity-resolution'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='person-arms-up' />}
						name={'Identity Resolution'}
						description='Modify the settings of the Identity Resolution, used to resolve the identity of the users'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='settings/data-warehouse'>
					<ListTile
						className='settings__setting'
						icon={<SlIcon name='database' />}
						name={'Data Warehouse'}
						description='Manage data warehouse mode and settings'
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
