import React from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { useLocation } from 'react-router-dom';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';

interface ConnectionTabsProps {
	connection: TransformedConnection;
}

const ConnectionTabs = ({ connection }: ConnectionTabsProps) => {
	const location = useLocation();
	const fragments = location.pathname.split('/');
	const tab = fragments[fragments.length - 1];

	return (
		<div className='connection-wrapper__links'>
			<Link path={`connections/${connection.id}/actions`}>
				<div
					className={`connection-wrapper__link${tab === 'actions' ? ' connection-wrapper__link--selected' : ''}`}
				>
					<SlIcon name='send-exclamation'></SlIcon>
					Actions
				</div>
			</Link>
			<Link path={`connections/${connection.id}/overview`}>
				<div
					className={`connection-wrapper__link${tab === 'overview' ? ' connection-wrapper__link--selected' : ''}`}
				>
					<SlIcon name='activity'></SlIcon>
					Overview
				</div>
			</Link>
			{(connection.isMobile || connection.isWebsite || connection.isServer || connection.isStream) && (
				<Link path={`connections/${connection.id}/events`}>
					<div
						className={`connection-wrapper__link${tab === 'events' ? ' connection-wrapper__link--selected' : ''}`}
					>
						<SlIcon name='play'></SlIcon>
						Live events
					</div>
				</Link>
			)}
			{connection.hasIdentities && (
				<Link path={`connections/${connection.id}/identities`}>
					<div
						className={`connection-wrapper__link${tab === 'identities' ? ' connection-wrapper__link--selected' : ''}`}
					>
						<SlIcon name='people'></SlIcon>
						Identities
					</div>
				</Link>
			)}
			<Link path={`connections/${connection.id}/settings`}>
				<div
					className={`connection-wrapper__link${tab === 'settings' ? ' connection-wrapper__link--selected' : ''}`}
				>
					<SlIcon name='sliders2'></SlIcon>
					Settings
				</div>
			</Link>
		</div>
	);
};

export default ConnectionTabs;
