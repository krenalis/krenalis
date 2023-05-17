import { useState, useContext } from 'react';
import './ConnectorsList.css';
import Card from '../Card/Card';
import { AppContext } from '../../context/AppContext';
import { NavigationContext } from '../../context/NavigationContext';
import { SlButton, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorsList = () => {
	let [goToConnectorSettings, setGoToConnectorSettings] = useState(0);

	let { redirect, connectors, API, showError } = useContext(AppContext);
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

	const authorizeWithOAuth = async (connectorID) => {
		localStorage.setItem('addConnectionID', connectorID);
		localStorage.setItem('addConnectionRole', connectionRole);
		let [res, err] = await API.connectors.authCodeURL(connectorID);
		if (err != null) {
			showError(err);
			return;
		}
		window.location = res.url;
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
							<Card
								name={c.Name}
								icon={c.Icon}
								type={c.Type}
								description={
									connectionRole === 'Source' ? c.SourceDescription : c.DestinationDescription
								}
							>
								<SlTooltip content={`Add ${c.Name}`}>
									<SlButton
										size='medium'
										variant='default'
										onClick={
											c.OAuth
												? () => authorizeWithOAuth(c.ID)
												: () => setGoToConnectorSettings(c.ID)
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
