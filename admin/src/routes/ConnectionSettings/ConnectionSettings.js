import { useState, useEffect, useRef } from 'react';
import './ConnectionSettings.css';
import NotFound from '../NotFound/NotFound';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import ConnectionHeading from '../../components/ConnectionHeading/ConnectionHeading';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import NavigationTabs from '../../components/NavigationTabs/NavigationTabs';
import ConnectionForm from '../../components/ConnectionForm/ConnectionForm';
import ConnectionDeletion from '../../components/ConnectionDeletion/ConnectionDeletion';
import ConnectionKeys from '../../components/ConnectionKeys/ConnectionKeys';
import ConnectionStream from '../../components/ConnectionStream/ConnectionStream';
import ConnectionStorage from '../../components/ConnectionStorage/ConnectionStorage';
import Toast from '../../components/Toast/Toast';
import call from '../../utils/call';
import { Navigate } from 'react-router-dom';
import { SlTab, SlTabGroup, SlTabPanel } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionSettings = () => {
	let [connection, setConnection] = useState(null);
	let [isDeleted, setIsDeleted] = useState(false);
	let [status, setStatus] = useState(null);
	let [notFound, setNotFound] = useState(false);

	const toastRef = useRef();
	const connectionID = Number(String(window.location).split('/').at(-2));

	const onError = (err) => {
		setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
		toastRef.current.toast();
		return;
	};

	useEffect(() => {
		const fetchData = async () => {
			// get the connection.
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
		fetchData();
	}, []);

	if (notFound) {
		return <NotFound />;
	}

	if (isDeleted) {
		return <Navigate to='/admin/connections' />;
	}

	let c = connection;
	if (c == null) return;
	let tabs = [
		{ Name: 'Overview', Link: `/admin/connections/${c.ID}`, Selected: false },
		{ Name: 'Settings', Link: `/admin/connections/${c.ID}/settings`, Selected: true },
	];
	if (c.Type === 'App' || c.Type === 'Database' || c.Type === 'File') {
		tabs.splice(1, 0, { Name: 'Properties', Link: `/admin/connections/${c.ID}/properties`, Selected: false });
	}
	if (c.Type === 'Database' && c.Role === 'Source') {
		tabs.splice(1, 0, { Name: 'SQL query', Link: `/admin/connections/${c.ID}/sql`, Selected: false });
	}

	return (
		<div className='ConnectionSettings'>
			<PrimaryBackground contentWidth={1400} height={300}>
				<Breadcrumbs
					onAccent={true}
					breadcrumbs={[{ Name: 'Connections', Link: '/admin/connections' }, { Name: `${c.Name}` }]}
				/>
				<ConnectionHeading connection={c} />
				<NavigationTabs tabs={tabs} onAccent={true} />
			</PrimaryBackground>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				<SlTabGroup className='settings' placement='start'>
					{c.HasSettings && (
						<>
							<SlTab slot='nav' panel='connection'>
								Connection
							</SlTab>
							<SlTabPanel name='connection'>
								<div className='panelTitle'>Connection</div>
								<ConnectionForm
									connection={c}
									onStatusChange={(status) => {
										setStatus(status);
										toastRef.current.toast();
									}}
									onError={onError}
								/>
							</SlTabPanel>
						</>
					)}
					{c.Type === 'Server' && c.Role === 'Source' && (
						<>
							<SlTab slot='nav' panel='apiKeys'>
								API Keys
							</SlTab>
							<SlTabPanel name='apiKeys'>
								<div className='panelTitle'>API Keys</div>
								<ConnectionKeys connection={c} onError={onError} />
							</SlTabPanel>
						</>
					)}
					{c.Type === 'File' && (
						<>
							<SlTab slot='nav' panel='storage'>
								Storage
							</SlTab>
							<SlTabPanel name='storage'>
								<div className='panelTitle'>Storage</div>
								<ConnectionStorage
									connection={c}
									onConnectionChange={(c) => setConnection(c)}
									onError={onError}
								/>
							</SlTabPanel>
						</>
					)}
					{(c.Type === 'Mobile' || c.Type === 'Website' || c.Type === 'Server') && c.Role === 'Source' && (
						<>
							<SlTab slot='nav' panel='eventStream'>
								Event Stream
							</SlTab>
							<SlTabPanel name='eventStream'>
								<div className='panelTitle'>Event Stream</div>
								<ConnectionStream
									connection={c}
									onConnectionChange={(c) => setConnection(c)}
									onError={onError}
								/>
							</SlTabPanel>
						</>
					)}
					<SlTab slot='nav' panel='deletion'>
						Deletion
					</SlTab>
					<SlTabPanel name='deletion'>
						<ConnectionDeletion connection={c} onDelete={() => setIsDeleted(true)} onError={onError} />
					</SlTabPanel>
				</SlTabGroup>
			</div>
		</div>
	);
};

export default ConnectionSettings;
