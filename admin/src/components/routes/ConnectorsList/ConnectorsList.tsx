import React, { useState, useContext } from 'react';
import './ConnectorsList.css';
import Card from '../../shared/Card/Card';
import { AppContext } from '../../../context/providers/AppProvider';
import { SlButton, SlIcon, SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';
import { authCodeURLResponse } from '../../../types/external/api';

const ConnectorsList = () => {
	const [goToConnectorSettings, setGoToConnectorSettings] = useState<number>(0);

	const { redirect, api, showError, connectors, setTitle } = useContext(AppContext);

	let connectionRole: string;
	const roleParam = new URL(document.location.href).searchParams.get('role');
	if (roleParam == null || roleParam === '') {
		connectionRole = 'Source';
	} else {
		connectionRole = roleParam;
	}

	setTitle(`Add a ${connectionRole.toLocaleLowerCase()} connection`);

	const authorizeWithOAuth = async (connectorID: number) => {
		localStorage.setItem('addConnectionID', String(connectorID));
		localStorage.setItem('addConnectionRole', connectionRole);
		let res: authCodeURLResponse;
		try {
			res = await api.connectors.authCodeURL(connectorID);
		} catch (err) {
			showError(err);
			return;
		}
		window.location.href = res.url;
		return;
	};

	if (goToConnectorSettings !== 0) {
		redirect(`connectors/${goToConnectorSettings}?role=${connectionRole}`);
		return null;
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
