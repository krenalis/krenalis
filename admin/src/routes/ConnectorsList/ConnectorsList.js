import { useState, useEffect, useRef } from 'react';
import './ConnectorsList.css';
import call from '../../utils/call';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import PrimaryBackground from '../../components/PrimaryBackground/PrimaryBackground';
import Card from '../../components/Card/Card';
import Toast from '../../components/Toast/Toast';
import { Navigate } from 'react-router-dom';
import { SlButton, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorsList = () => {
	let [connectors, setConnectors] = useState([]);
	let [goToConnectorSettings, setGoToConnectorSettings] = useState(0);
	let [status, setStatus] = useState(null);

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

	const authorizeWithOAuth = (c) => {
		localStorage.setItem('addConnectionID', c.ID);
		localStorage.setItem('addConnectionRole', connectionRole);
		window.location = c.OAuth.URL;
		return;
	};

	if (goToConnectorSettings !== 0) {
		return <Navigate to={`/admin/connectors/${goToConnectorSettings}?role=${connectionRole}`} />;
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
										onClick={
											c.OAuth == null
												? () => setGoToConnectorSettings(c.ID)
												: () => authorizeWithOAuth(c)
										}
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
		</div>
	);
};

export default ConnectorsList;
