import React from 'react';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { useLocation } from 'react-router-dom';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';
import LittleLogo from '../../base/LittleLogo/LittleLogo';

interface ConnectionTabsProps {
	connection: TransformedConnection;
}

const ConnectionTabs = ({ connection }: ConnectionTabsProps) => {
	const location = useLocation();
	const fragments = location.pathname.split('/');
	const tab = fragments[fragments.length - 1];

	const connectorLogo = <LittleLogo code={connection.connector.code} />;

	return (
		<div className='connection-wrapper__tabs'>
			{connectorLogo && <div className='connection-wrapper__connector-logo'>{connectorLogo}</div>}
			<div className='connection-wrapper__links'>
				<Link path={`connections/${connection.id}/actions`}>
					<div
						className={`connection-wrapper__link${tab === 'actions' ? ' connection-wrapper__link--selected' : ''}`}
					>
						<SlIcon name='play-circle'></SlIcon>
						Actions
					</div>
				</Link>
				<Link path={`connections/${connection.id}/metrics`}>
					<div
						className={`connection-wrapper__link${tab === 'metrics' ? ' connection-wrapper__link--selected' : ''}`}
					>
						<SlIcon name='bar-chart'></SlIcon>
						Metrics
					</div>
				</Link>
				{(connection.isMessageBroker || connection.isSDK || connection.isWebhook) && (
					<Link path={`connections/${connection.id}/events`}>
						<div
							className={`connection-wrapper__link${tab === 'events' ? ' connection-wrapper__link--selected' : ''}`}
						>
							<SlIcon name='bug'></SlIcon>
							Event debugger
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
						<SlIcon name='gear'></SlIcon>
						Settings
					</div>
				</Link>
			</div>
		</div>
	);
};

export default ConnectionTabs;
