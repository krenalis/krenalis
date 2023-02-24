import { useState, useEffect, useContext } from 'react';
import './ConnectorsList.css';
import Breadcrumbs from '../Breadcrumbs/Breadcrumbs';
import PrimaryBackground from '../PrimaryBackground/PrimaryBackground';
import Card from '../Card/Card';
import { AppContext } from '../../context/AppContext';
import { SlButton, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorsList = () => {
	let [connectors, setConnectors] = useState([]);
	let [goToConnectorSettings, setGoToConnectorSettings] = useState(0);

	let { API, showError, redirect } = useContext(AppContext);

	let connectionRole;
	let roleParam = new URL(document.location).searchParams.get('role');
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}

	useEffect(() => {
		const fetchConnectors = async () => {
			let [connectors, err] = await API.connectors.find();
			if (err != null) {
				showError(err);
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
		redirect(`/admin/connectors/${goToConnectorSettings}?role=${connectionRole}`);
		return;
	}

	return (
		<div className='ConnectorsList'>
			<PrimaryBackground height={250} overlap={100}>
				<Breadcrumbs
					breadcrumbs={[
						{ Name: 'Connections', Link: '/admin/connections' },
						{ Name: `Add a new ${connectionRole}` },
					]}
					onAccent={true}
				/>
			</PrimaryBackground>
			<div className='routeContent'>
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
