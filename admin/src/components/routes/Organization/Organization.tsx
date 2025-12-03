import React, { ReactNode } from 'react';
import './Organization.css';
import ListTile from '../../base/ListTile/ListTile';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Link } from '../../base/Link/Link';
import { Outlet, useLocation } from 'react-router-dom';

const Organization = () => {
	const location = useLocation();

	let content: ReactNode;

	if (location.pathname.endsWith('organization')) {
		content = (
			<>
				<Link path='organization/members'>
					<ListTile
						className='organization__entry'
						icon={<SlIcon name='people' />}
						name={'Team members'}
						description='View and modify your team members'
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
