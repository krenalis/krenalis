import React from 'react';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import { useLocation } from 'react-router-dom';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

interface ConnectionTabsProps {
	connection: TransformedConnection;
}

const ConnectionTabs = ({ connection }: ConnectionTabsProps) => {
	const location = useLocation();
	const fragments = location.pathname.split('/');
	const tab = fragments[fragments.length - 1];

	const connectorLogo = <LittleLogo code={connection.connector.code} path={CONNECTORS_ASSETS_PATH} />;

	return (
		<div className='connection-wrapper__tabs'>
			{connectorLogo && <div className='connection-wrapper__connector-logo'>{connectorLogo}</div>}
			<div className='connection-wrapper__links'>
				<Link path={`connections/${connection.id}/pipelines`}>
					<div
						className={`connection-wrapper__link${tab === 'pipelines' ? ' connection-wrapper__link--selected' : ''}`}
					>
						<WaIcon name='play-circle'></WaIcon>
						Pipelines
					</div>
				</Link>
				<Link path={`connections/${connection.id}/metrics`}>
					<div
						className={`connection-wrapper__link${tab === 'metrics' ? ' connection-wrapper__link--selected' : ''}`}
					>
						<WaIcon name='bar-chart'></WaIcon>
						Metrics
					</div>
				</Link>
				{(connection.isMessageBroker || connection.isSDK || connection.isWebhook) && (
					<Link path={`connections/${connection.id}/events`}>
						<div
							className={`connection-wrapper__link${tab === 'events' ? ' connection-wrapper__link--selected' : ''}`}
						>
							<WaIcon name='bug'></WaIcon>
							Event debugger
						</div>
					</Link>
				)}
				{connection.hasIdentities && (
					<Link path={`connections/${connection.id}/identities`}>
						<div
							className={`connection-wrapper__link${tab === 'identities' ? ' connection-wrapper__link--selected' : ''}`}
						>
							<WaIcon name='people'></WaIcon>
							Identities
						</div>
					</Link>
				)}
				<Link path={`connections/${connection.id}/settings`}>
					<div
						className={`connection-wrapper__link${tab === 'settings' ? ' connection-wrapper__link--selected' : ''}`}
					>
						<WaIcon name='gear'></WaIcon>
						Settings
					</div>
				</Link>
			</div>
		</div>
	);
};

export default ConnectionTabs;
