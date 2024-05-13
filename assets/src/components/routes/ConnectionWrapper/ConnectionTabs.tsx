import React from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { useLocation } from 'react-router-dom';
import TransformedConnection from '../../../lib/helpers/transformedConnection';
import { Link } from '../../shared/Link/Link';

interface ConnectionTabsProps {
	connection: TransformedConnection;
}

const ConnectionTabs = ({ connection }: ConnectionTabsProps) => {
	const location = useLocation();
	const fragments = location.pathname.split('/');
	const tab = fragments[fragments.length - 1];

	return (
		<div className='links'>
			<Link path={`connections/${connection.id}/actions`}>
				<div className={`link${tab === 'actions' ? ' selected' : ''}`}>
					<SlIcon name='send-exclamation'></SlIcon>
					Actions
				</div>
			</Link>
			<Link path={`connections/${connection.id}/overview`}>
				<div className={`link${tab === 'overview' ? ' selected' : ''}`}>
					<SlIcon name='activity'></SlIcon>
					Overview
				</div>
			</Link>
			{(connection.isMobile || connection.isWebsite || connection.isServer || connection.isStream) && (
				<Link path={`connections/${connection.id}/events`}>
					<div className={`link${tab === 'events' ? ' selected' : ''}`}>
						<SlIcon name='play'></SlIcon>
						Live events
					</div>
				</Link>
			)}
			{connection.hasIdentities && (
				<Link path={`connections/${connection.id}/identities`}>
					<div className={`link${tab === 'identities' ? ' selected' : ''}`}>
						<SlIcon name='people'></SlIcon>
						Identities
					</div>
				</Link>
			)}
			<Link path={`connections/${connection.id}/settings`}>
				<div className={`link${tab === 'settings' ? ' selected' : ''}`}>
					<SlIcon name='sliders2'></SlIcon>
					Settings
				</div>
			</Link>
		</div>
	);
};

export default ConnectionTabs;
