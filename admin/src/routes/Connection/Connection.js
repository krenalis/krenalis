import { useState, useEffect, useRef } from 'react';
import './Connection.css';
import NotFound from '../NotFound/NotFound';
import Toast from '../../components/Toast/Toast';
import ConnectionOverview from '../ConnectionOverview/ConnectionOverview';
import ConnectionSQL from '../ConnectionSQL/ConnectionSQL';
import ConnectionProperties from '../ConnectionProperties/ConnectionProperties';
import ConnectionSettings from '../ConnectionSettings/ConnectionSettings';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import ConnectionHeading from '../../components/ConnectionHeading/ConnectionHeading';
import call from '../../utils/call';
import { SlTab, SlTabGroup, SlTabPanel } from '@shoelace-style/shoelace/dist/react/index.js';

const Connection = () => {
	let [connection, setConnection] = useState(null);
	let [selectedSection, setSelectedSection] = useState('');
	let [notFound, setNotFound] = useState(false);
	let [status, setStatus] = useState(null);

	const toastRef = useRef();
	const connectionID = Number(String(window.location).split('/').pop());

	const onError = (err) => {
		setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
		toastRef.current.toast();
		return;
	};

	const onStatusChange = (status) => {
		setStatus(status);
		toastRef.current.toast();
	};

	const onConnectionChange = (c) => setConnection(c);

	const onSlTabShow = (e) => {
		setSelectedSection(e.detail.name);
	};

	useEffect(() => {
		const fetchConnection = async () => {
			let [connection, err] = await call('/admin/connections/get', 'POST', connectionID);
			if (err) {
				onError(err);
				return;
			}
			if (connection == null) {
				setNotFound(true);
				return;
			}
			setConnection(connection);
		};
		fetchConnection();
	}, []);

	if (notFound) {
		return <NotFound />;
	}

	let c = connection;
	if (c == null) return;
	return (
		<div className='Connection'>
			<PrimaryBackground contentWidth={1400} height={250}>
				<Breadcrumbs
					onAccent={true}
					breadcrumbs={[{ Name: 'Connections', Link: '/admin/connections' }, { Name: `${c.Name}` }]}
				/>
				<ConnectionHeading connection={c} />
			</PrimaryBackground>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				<SlTabGroup className='connectionSections' onSlTabShow={onSlTabShow}>
					<SlTab slot='nav' panel='overview'>
						Overview
					</SlTab>
					<SlTabPanel name='overview'>
						<ConnectionOverview
							connection={c}
							onError={onError}
							onStatusChange={onStatusChange}
							isSelected={selectedSection === 'overview'}
						/>
					</SlTabPanel>
					{c.Type === 'Database' && c.Role === 'Source' && (
						<>
							<SlTab slot='nav' panel='sqlquery'>
								SQL Query
							</SlTab>
							<SlTabPanel name='sqlquery'>
								<ConnectionSQL
									connection={c}
									onError={onError}
									onStatusChange={onStatusChange}
									isSelected={selectedSection === 'sqlquery'}
								/>
							</SlTabPanel>
						</>
					)}
					{(c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') && (
						<>
							<SlTab slot='nav' panel='properties'>
								Properties
							</SlTab>
							<SlTabPanel name='properties'>
								<ConnectionProperties
									connection={c}
									onError={onError}
									onStatuChange={onStatusChange}
									renderArrows={selectedSection}
									isSelected={selectedSection === 'properties'}
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
							onError={onError}
							onStatusChange={onStatusChange}
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
