import { useState, useEffect, useContext } from 'react';
import './Connection.css';
import getConnectionStatusInfos from '../../utils/getConnectionStatusInfos';
import LittleLogo from '../LittleLogo/LittleLogo';
import Flex from '../Flex/Flex';
import StatusDot from '../StatusDot/StatusDot';
import { NotFoundError } from '../../api/errors';
import { AppContext } from '../../context/AppContext';
import { NavigationContext } from '../../context/NavigationContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { Outlet, NavLink } from 'react-router-dom';
import { SlBadge, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const Connection = () => {
	let [connection, setConnection] = useState(null);
	let [currentSection, setCurrentSection] = useState('');

	const { API, showError, showNotFound } = useContext(AppContext);
	const { setCurrentTitle, setPreviousRoute } = useContext(NavigationContext);

	setPreviousRoute('/admin/connections');

	let urlFragments = String(window.location).split('/');
	let fragmentIndex = urlFragments.findIndex((f) => f === 'connections');
	let connectionID = Number(urlFragments[fragmentIndex + 1]);

	const setCurrentConnectionSection = (section) => {
		setCurrentSection(section);
	};

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
			setCurrentTitle(
				<Flex alignItems='baseline' gap='10px'>
					<span style={{ position: 'relative', top: '3px' }}>
						<LittleLogo url={c.LogoURL} alternativeText={`${c.Name}'s logo`}></LittleLogo>
					</span>
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
	if (c == null) return;
	return (
		<div className='Connection'>
			<div className='links'>
				<div className={`link${currentSection === 'overview' ? ' selected' : ''}`}>
					<NavLink to='overview'></NavLink>
					<SlIcon name='activity'></SlIcon>
					Overview
				</div>
				{c.Type === 'Database' && c.Role === 'Source' && (
					<div className={`link${currentSection === 'sql' ? ' selected' : ''}`}>
						<NavLink to='sql'></NavLink>
						<SlIcon name='filetype-sql'></SlIcon>
						SQL Query
					</div>
				)}
				{(c.Type === 'Mobile' || c.Type === 'Website' || c.Type === 'Server' || c.Type === 'Stream') && (
					<div className={`link${currentSection === 'events' ? ' selected' : ''}`}>
						<NavLink to='events'></NavLink>
						<SlIcon name='play'></SlIcon>
						Live events
					</div>
				)}
				{c.Role === 'Destination' && (
					<div className={`link${currentSection === 'actions' ? ' selected' : ''}`}>
						<NavLink to='actions'></NavLink>
						<SlIcon name='play'></SlIcon>
						Live actions
					</div>
				)}
				{(c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') && (
					<div className={`link${currentSection === 'mappings' ? ' selected' : ''}`}>
						<NavLink to='mappings'></NavLink>
						<SlIcon name='diagram-3'></SlIcon>
						Mappings
					</div>
				)}
				{(c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') && (
					<div className={`link${currentSection === 'transformation' ? ' selected' : ''}`}>
						<NavLink to='transformation'></NavLink>
						<SlIcon name='braces'></SlIcon>
						Transformation
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
						setCurrentConnectionSection: setCurrentConnectionSection,
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
