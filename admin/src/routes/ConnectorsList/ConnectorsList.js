import { useState, useEffect, useRef } from 'react';
import './ConnectorsList.css';
import call from '../../utils/call';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import Card from '../../components/Card/Card';
import Toast from '../../components/Toast/Toast';
import { Navigate } from 'react-router-dom';
import { SlButton, SlDialog, SlIcon, SlTooltip, SlInput } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorsList = () => {
	let [connectors, setConnectors] = useState([]);
	let [storageConnections, setStorageConnections] = useState([]);
	let [connectorToAdd, setConnectorToAdd] = useState(null);
	let [goToConnectionAdded, setGoToConnectionAdded] = useState(0);
	let [showStorage, setShowStorage] = useState(false);
	let [askWebsiteInformations, setAskWebsiteInformations] = useState(false);
	let [status, setStatus] = useState(null);
	let [websitePort, setWebsitePort] = useState('');
	let [websiteHost, setWebsiteHost] = useState('');

	const toastRef = useRef();
	let connectionRole;
	let roleParam = new URL(document.location).searchParams.get('role');
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}

	const onError = (err) => {
		setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
		toastRef.current.toast();
		return;
	};

	useEffect(() => {
		const fetchConnectors = async () => {
			let [connectors, err] = await call('/admin/connectors/find', 'GET');
			if (err != null) {
				onError(err);
				return;
			}
			setConnectors(connectors);
		};
		fetchConnectors();
	}, []);

	const installConnection = async (c, storage, host) => {
		let body = { Connector: c.ID, Storage: 0, Role: connectionRole, Host: '' };
		if (c.OAuth === null) {
			if (c.Type === 'File') body.Storage = storage;
			if (c.Type === 'Website') body.Host = host;
			let [, err] = await call('/admin/add-connection', 'POST', body);
			if (err != null) {
				onError(err);
				return;
			}
			setGoToConnectionAdded(c.ID);
			return;
		}
		// install with OAuth.
		document.cookie = `add-connection=${c.ID};path=/`;
		document.cookie = `role=${connectionRole};path=/`;
		window.location = c.OAuth.URL;
		return;
	};

	const addConnection = async (c) => {
		setConnectorToAdd(c);
		if (c.Type === 'File') {
			let [cns, err] = await call('/admin/connections/find', 'GET');
			if (err != null) {
				onError(err);
				return;
			}
			let storageConnections = [];
			for (let c of cns) {
				if (c.Type === 'Storage' && c.Role === connectionRole) storageConnections.push(c);
			}
			setStorageConnections(storageConnections);
			setShowStorage(true);
			return;
		}
		if (c.Type === 'Website') {
			setAskWebsiteInformations(true);
			return;
		}
		await installConnection(c);
	};

	const addFileConnection = async (storageID) => {
		await installConnection(connectorToAdd, storageID, '');
		setShowStorage(false);
	};

	const addWebsiteConnection = async () => {
		await installConnection(connectorToAdd, 0, websiteHost + ':' + websitePort);
		setAskWebsiteInformations(false);
	};

	if (goToConnectionAdded !== 0) {
		return <Navigate to={`added/${goToConnectionAdded}?role=${connectionRole}`} />;
	}

	return (
		<div className='ConnectorsList'>
			<PrimaryBackground height={300} overlap={100}>
				<Breadcrumbs
					breadcrumbs={[
						{ Name: 'Connections', Link: '/admin/connections' },
						{ Name: `Add a new ${connectionRole}` },
					]}
					onAccent={true}
				/>
			</PrimaryBackground>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				<div className='connectors'>
					{connectors.map((c) => {
						return (
							<Card key={c.ID} name={c.Name} logoURL={c.LogoURL} type={c.Type}>
								<SlTooltip content={`Add ${c.Name}`}>
									<SlButton
										size='medium'
										variant='primary'
										onClick={async () => {
											await addConnection(c);
										}}
										circle
									>
										<SlIcon name='plus' />
									</SlButton>
								</SlTooltip>
							</Card>
						);
					})}
				</div>
			</div>
			<SlDialog
				label='Select a storage'
				open={showStorage}
				onSlAfterHide={() => {
					setShowStorage(false);
				}}
				style={{ '--width': '600px' }}
			>
				{storageConnections.length === 0 ? (
					<div className='no-storage'>No storage available</div>
				) : (
					storageConnections.map((s) => {
						return (
							<div className='storage'>
								<div className='name'>{s.Name}</div>
								<SlButton
									variant='primary'
									onClick={async () => {
										await addFileConnection(s.ID);
									}}
									className='addStorage'
								>
									<SlIcon name='arrow-right' />
								</SlButton>
							</div>
						);
					})
				)}
			</SlDialog>
			<SlDialog
				label='Website informations'
				open={askWebsiteInformations}
				onSlAfterHide={() => {
					setAskWebsiteInformations(false);
				}}
				style={{ '--width': '600px' }}
			>
				<div className='websiteInfo'>
					<SlInput
						label='Host'
						className='hostInput'
						onSlChange={(e) => {
							setWebsiteHost(e.currentTarget.value);
						}}
						value={websiteHost}
					/>
					<SlInput
						label='Port'
						className='portInput'
						onSlChange={(e) => {
							setWebsitePort(e.currentTarget.value);
						}}
						value={websitePort}
					/>
					<SlButton className='addWebsite' variant='primary' onClick={addWebsiteConnection}>
						Add website
					</SlButton>
				</div>
			</SlDialog>
		</div>
	);
};

export default ConnectorsList;
