import React, { ReactNode, useContext } from 'react';
import './Organization.css';
import ListTile from '../../base/ListTile/ListTile';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Link } from '../../base/Link/Link';
import { Outlet, useLocation } from 'react-router-dom';
import appContext from '../../../context/AppContext';
import { useAuth } from '@workos-inc/authkit-react';

const Organization = () => {
	const location = useLocation();

	const { publicMetadata } = useContext(appContext);

	let content: ReactNode;

	const hasWorkos = publicMetadata.workosClientID !== '';

	if (location.pathname.endsWith('organization')) {
		content = (
			<>
				{hasWorkos ? <WorkOSMembersLink /> : <MembersLink />}
				<Link path='organization/access-keys'>
					<ListTile
						className='organization__entry'
						icon={<SlIcon name='key' />}
						name={'API and MCP keys'}
						description='View and modify your API and MCP keys'
						showHover={true}
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

const WorkOSMembersLink = () => {
	const { isLoading, user, role } = useAuth();

	if (isLoading || !user) {
		return null;
	}

	if (role === 'admin') {
		return <MembersLink />;
	}
};

const MembersLink = () => {
	return (
		<Link path='organization/members'>
			<ListTile
				className='organization__entry'
				icon={<SlIcon name='people' />}
				name={'Team members'}
				description='View and modify your team members'
				showHover={true}
				action={<SlIcon name='chevron-right' />}
			/>
		</Link>
	);
};

export default Organization;
