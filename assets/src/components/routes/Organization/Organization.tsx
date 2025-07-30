import React, { ReactNode, useContext, useLayoutEffect } from 'react';
import './Organization.css';
import ListTile from '../../base/ListTile/ListTile';
import AppContext from '../../../context/AppContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Link } from '../../base/Link/Link';
import { Outlet, useLocation } from 'react-router-dom';

const Organization = () => {
	const { setTitle } = useContext(AppContext);

	const location = useLocation();

	useLayoutEffect(() => {
		setTitle('Organization');
	}, []);

	let content: ReactNode;

	if (location.pathname.endsWith('organization')) {
		content = (
			<>
				<p className='organization__title'>Organization</p>
				<Link path='organization/members'>
					<ListTile
						className='organization__entry'
						icon={<SlIcon name='people' />}
						name={'Members'}
						description='View and modify your organization members'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
				<Link path='organization/access-keys'>
					<ListTile
						className='organization__entry'
						icon={<SlIcon name='key' />}
						name={'API and MCP keys'}
						description='View and modify your API and MCP keys'
						action={<SlIcon name='chevron-right' />}
					/>
				</Link>
			</>
		);
	} else {
		content = <Outlet />;
	}

	return (
		<div className='organization'>
			<div className='organization__content'>{content}</div>
		</div>
	);
};

export default Organization;
