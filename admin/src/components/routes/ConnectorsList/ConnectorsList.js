import { useState, useContext } from 'react';
import './ConnectorsList.css';
import Card from '../../common/Card/Card';
import { AppContext } from '../../../providers/AppProvider';
import { SlButton, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectorsList = () => {
	const [goToConnectorSettings, setGoToConnectorSettings] = useState(0);

	const { redirect, api, showError, connectors, setTitle } = useContext(AppContext);

	let connectionRole;
	const roleParam = new URL(document.location).searchParams.get('role');
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}

	setTitle(`Add a ${connectionRole.toLocaleLowerCase()} connection`);

	const authorizeWithOAuth = async (connectorID) => {
		localStorage.setItem('addConnectionID', connectorID);
		localStorage.setItem('addConnectionRole', connectionRole);
		const [res, err] = await api.connectors.authCodeURL(connectorID);
		if (err != null) {
			showError(err);
			return;
		}
		window.location = res.url;
		return;
	};

	if (goToConnectorSettings !== 0) {
		redirect(`connectors/${goToConnectorSettings}?role=${connectionRole}`);
		return;
	}

	return (
		<div className='connectorsList'>
			<div className='routeContent'>
				<div className='connectors'>
					{connectors.map((c) => {
						return (
							<Card
								name={c.name}
								icon={c.icon}
								type={c.type}
								description={
									connectionRole === 'Source' ? c.sourceDescription : c.destinationDescription
								}
							>
								<SlTooltip content={`Add ${c.name}`}>
									<SlButton
										size='medium'
										variant='default'
										onClick={
											c.oAuth
												? () => authorizeWithOAuth(c.id)
												: () => setGoToConnectorSettings(c.id)
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
