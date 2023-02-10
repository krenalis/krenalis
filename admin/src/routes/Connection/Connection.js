import { useState, useEffect, useContext } from 'react';
import './Connection.css';
import ConnectionOverview from '../ConnectionOverview/ConnectionOverview';
import ConnectionSQL from '../ConnectionSQL/ConnectionSQL';
import ConnectionMappings from '../ConnectionMappings/ConnectionMappings';
import ConnectionEvents from '../../components/ConnectionEvents/ConnectionEvents';
import ConnectionTransformation from '../ConnectionTransformation/ConnectionTransformation';
import ConnectionSettings from '../ConnectionSettings/ConnectionSettings';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import ConnectionHeading from '../../components/ConnectionHeading/ConnectionHeading';
import { NotFoundError } from '../../api/errors';
import { AppContext } from '../../context/AppContext';
import { SlTab, SlTabGroup, SlTabPanel } from '@shoelace-style/shoelace/dist/react/index.js';

const Connection = () => {
	let [connection, setConnection] = useState(null);
	let [selectedSection, setSelectedSection] = useState('');

	const { API, showError, showNotFound } = useContext(AppContext);
	const connectionID = Number(String(window.location).split('/').pop());

	const onConnectionChange = (c) => setConnection(c);

	const onSlTabShow = (e) => {
		setSelectedSection(e.detail.name);
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
			<PrimaryBackground height={200} overlap={65}>
				<Breadcrumbs
					onAccent={true}
					breadcrumbs={[{ Name: 'Connections', Link: '/admin/connections' }, { Name: `${c.Name}` }]}
				/>
				<ConnectionHeading connection={c} />
			</PrimaryBackground>
			<div className='routeContent'>
				<SlTabGroup className='connectionSections' onSlTabShow={onSlTabShow}>
					<SlTab slot='nav' panel='overview'>
						Overview
					</SlTab>
					<SlTabPanel name='overview'>
						<ConnectionOverview connection={c} isSelected={selectedSection === 'overview'} />
					</SlTabPanel>
					{c.Type === 'Database' && c.Role === 'Source' && (
						<>
							<SlTab slot='nav' panel='sqlquery'>
								SQL Query
							</SlTab>
							<SlTabPanel name='sqlquery'>
								<ConnectionSQL connection={c} isSelected={selectedSection === 'sqlquery'} />
							</SlTabPanel>
						</>
					)}
					{(c.Type === 'Mobile' || c.Type === 'Website' || c.Type === 'Server' || c.Type === 'Stream') && (
						<>
							<SlTab slot='nav' panel='events'>
								Events
							</SlTab>
							<SlTabPanel name='events'>
								<ConnectionEvents connection={c} isSelected={selectedSection === 'events'} />
							</SlTabPanel>
						</>
					)}
					{(c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') && (
						<>
							<SlTab slot='nav' panel='mappings'>
								Mappings
							</SlTab>
							<SlTabPanel name='mappings'>
								<ConnectionMappings
									connection={c}
									onConnectionChange={onConnectionChange}
									isSelected={selectedSection === 'mappings'}
								/>
							</SlTabPanel>
						</>
					)}
					{(c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') && (
						<>
							<SlTab slot='nav' panel='transformation'>
								Transformation
							</SlTab>
							<SlTabPanel name='transformation'>
								<ConnectionTransformation
									connection={c}
									onConnectionChange={onConnectionChange}
									isSelected={selectedSection === 'transformation'}
								/>
							</SlTabPanel>
						</>
					)}
					<SlTab slot='nav' panel='settings'>
						Settings
					</SlTab>
					<SlTabPanel name='settings'>
						<ConnectionSettings
							connection={c}
							onConnectionChange={onConnectionChange}
							isSelected={selectedSection === 'settings'}
						/>
					</SlTabPanel>
				</SlTabGroup>
			</div>
		</div>
	);
};

export default Connection;
