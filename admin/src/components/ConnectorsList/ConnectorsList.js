import { useState, useEffect, useContext } from 'react';
import './ConnectorsList.css';
import Card from '../Card/Card';
import { AppContext } from '../../context/AppContext';
import { NavigationContext } from '../../context/NavigationContext';
import { SlButton, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorsList = () => {
	let [connectors, setConnectors] = useState([]);
	let [goToConnectorSettings, setGoToConnectorSettings] = useState(0);

	let { API, showError, redirect } = useContext(AppContext);
	let { setCurrentTitle, setPreviousRoute } = useContext(NavigationContext);
	setPreviousRoute('/admin/connections');

	let connectionRole;
	let roleParam = new URL(document.location).searchParams.get('role');
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}

	setCurrentTitle(`Add a ${connectionRole.toLocaleLowerCase()} connection`);

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
			<div className='routeContent'>
				<div className='connectors'>
					{connectors.map((c) => {
						return (
							<Card key={c.ID} name={c.Name} logoURL={c.LogoURL} type={c.Type}>
								<SlTooltip content={`Add ${c.Name}`}>
									<SlButton
										size='medium'
										variant='default'
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
