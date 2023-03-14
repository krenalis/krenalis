import { useState, useEffect, useContext } from 'react';
import './Connection.css';
import PrimaryBackground from '../PrimaryBackground/PrimaryBackground';
import Breadcrumbs from '../Breadcrumbs/Breadcrumbs';
import ConnectionHeading from '../ConnectionHeading/ConnectionHeading';
import { NotFoundError } from '../../api/errors';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { Outlet, NavLink } from 'react-router-dom';

const Connection = () => {
	let [connection, setConnection] = useState(null);
	let [currentSection, setCurrentSection] = useState('');

	const { API, showError, showNotFound } = useContext(AppContext);
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
		};
		fetchConnection();
	}, []);

	let c = connection;
	if (c == null) return;
	return (
		<div className='Connection'>
			<PrimaryBackground height={200}>
				<Breadcrumbs
					onAccent={true}
					breadcrumbs={[{ Name: 'Connections', Link: '/admin/connections' }, { Name: `${c.Name}` }]}
				/>
				<ConnectionHeading connection={c} />
				<div className='links'>
					<div className={`link${currentSection === 'overview' ? ' selected' : ''}`}>
						<NavLink to='overview'></NavLink>
						Overview
					</div>
					{c.Type === 'Database' && c.Role === 'Source' && (
						<div className={`link${currentSection === 'sql' ? ' selected' : ''}`}>
							<NavLink to='sql'></NavLink>
							SQL Query
						</div>
					)}
					{(c.Type === 'Mobile' || c.Type === 'Website' || c.Type === 'Server' || c.Type === 'Stream') && (
						<div className={`link${currentSection === 'events' ? ' selected' : ''}`}>
							<NavLink to='events'></NavLink>
							Events
						</div>
					)}
					{c.Role === 'Destination' && (
						<div className={`link${currentSection === 'actions' ? ' selected' : ''}`}>
							<NavLink to='actions'></NavLink>
							Actions
						</div>
					)}
					{(c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') && (
						<div className={`link${currentSection === 'mappings' ? ' selected' : ''}`}>
							<NavLink to='mappings'></NavLink>
							Mappings
						</div>
					)}
					{(c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') && (
						<div className={`link${currentSection === 'transformation' ? ' selected' : ''}`}>
							<NavLink to='transformation'></NavLink>
							Transformation
						</div>
					)}
					<div className={`link${currentSection === 'settings' ? ' selected' : ''}`}>
						<NavLink to='settings'></NavLink>
						Settings
					</div>
				</div>
			</PrimaryBackground>
			<div className='routeContent'>
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
