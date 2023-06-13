import { useState, useEffect, useContext } from 'react';
import './Connection.css';
import getConnectionStatusInfos from '../../utils/getConnectionStatusInfos';
import LittleLogo from '../LittleLogo/LittleLogo';
import UnknownLogo from '../UnknownLogo/UnknownLogo';
import Flex from '../Flex/Flex';
import StatusDot from '../StatusDot/StatusDot';
import { NotFoundError } from '../../api/errors';
import { AppContext } from '../../context/AppContext';
import { NavigationContext } from '../../context/NavigationContext';
import { ConnectionsContext } from '../../context/ConnectionsContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { Outlet, NavLink } from 'react-router-dom';
import { SlBadge, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Connection = () => {
	let [connection, setConnection] = useState(null);
	let [currentSection, setCurrentSection] = useState('');

	const { API, showError, showNotFound, connectors } = useContext(AppContext);
	const { setCurrentTitle, setCurrentRoute, setPreviousRoute } = useContext(NavigationContext);
	const { connections } = useContext(ConnectionsContext);

	setPreviousRoute('/admin/connections');

	let urlFragments = String(window.location).split('/');
	let fragmentIndex = urlFragments.findIndex((f) => f === 'connections');
	let connectionID = Number(urlFragments[fragmentIndex + 1]);

	let connectionRole = connections.find((c) => c.ID === connectionID).Role;
	let currentRoute = connectionRole === 'Source' ? 'connections/sources' : 'connections/destinations';
	setCurrentRoute(currentRoute);

	useEffect(() => {
		const fetchConnection = async () => {
			let [connection, err] = await API.connections.get(connectionID);
			if (err) {
				if (err instanceof NotFoundError) {
					showNotFound();
					return;
				}
				showError(err);
				return;
			}
			setConnection(connection);
			let c = connection;
			let { text: statusText, variant: statusVariant } = getConnectionStatusInfos(c);
			let connector = connectors.find((connector) => connector.ID === c.Connector);
			let logo;
			if (connector.Icon === '') {
				logo = <UnknownLogo size={21} />;
			} else {
				logo = <LittleLogo icon={connector.Icon} />;
			}
			setCurrentTitle(
				<Flex alignItems='baseline' gap='10px'>
					<span style={{ position: 'relative', top: '3px' }}>{logo}</span>
					<div className='text'>{c.Name}</div>
					<StatusDot statusText={statusText} statusVariant={statusVariant} />
					<SlBadge className='type' variant='neutral'>
						{c.Type}
					</SlBadge>
					<SlBadge className='role' variant='neutral'>
						{c.Role}
					</SlBadge>
				</Flex>
			);
		};
		fetchConnection();
	}, []);

	let c = connection;
	if (c == null) {
		return;
	}

	return (
		<div className='Connection'>
			<div className='links'>
				{
					<div className={`link${currentSection === 'actions' ? ' selected' : ''}`}>
						<NavLink to='actions'></NavLink>
						<SlIcon name='send-exclamation'></SlIcon>
						Actions
					</div>
				}
				<div className={`link${currentSection === 'overview' ? ' selected' : ''}`}>
					<NavLink to='overview'></NavLink>
					<SlIcon name='activity'></SlIcon>
					Overview
				</div>
				{(c.Type === 'Mobile' || c.Type === 'Website' || c.Type === 'Server' || c.Type === 'Stream') && (
					<div className={`link${currentSection === 'events' ? ' selected' : ''}`}>
						<NavLink to='events'></NavLink>
						<SlIcon name='play'></SlIcon>
						Live events
					</div>
				)}
				<div className={`link${currentSection === 'settings' ? ' selected' : ''}`}>
					<NavLink to='settings'></NavLink>
					<SlIcon name='sliders2'></SlIcon>
					Settings
				</div>
			</div>
			<div className='routeContent connection'>
				<ConnectionContext.Provider
					value={{
						connection: c,
						setCurrentConnectionSection: setCurrentSection,
						setConnection: setConnection,
					}}
				>
					<Outlet />
				</ConnectionContext.Provider>
			</div>
		</div>
	);
};

export default Connection;
