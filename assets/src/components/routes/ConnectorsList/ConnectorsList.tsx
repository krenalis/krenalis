import React, { useState, useContext, useEffect, useLayoutEffect, useMemo } from 'react';
import './ConnectorsList.css';
import Card from '../../shared/Card/Card';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { authCodeURLResponse } from '../../../types/external/api';
import { useLocation } from 'react-router-dom';

const ConnectorsList = () => {
	const [goToConnectorSettings, setGoToConnectorSettings] = useState<number>(0);
	const [searchTerm, setSearchTerm] = useState<string>('');

	const { redirect, api, handleError, connectors, setTitle } = useContext(AppContext);

	const location = useLocation();

	const connectionRole = useMemo(() => {
		const roleParam = new URL(document.location.href).searchParams.get('role');
		if (roleParam == null || roleParam === '') {
			return 'Source';
		} else {
			return roleParam;
		}
	}, [location]);

	useLayoutEffect(() => {
		setTitle(`Add a ${connectionRole.toLocaleLowerCase()}`);
	}, [connectionRole]);

	useEffect(() => {
		if (goToConnectorSettings !== 0) {
			const connector = connectors.find((c) => c.id === goToConnectorSettings);
			if (connector.isFile) {
				redirect(`connectors/file/${goToConnectorSettings}?role=${connectionRole}`);
				return;
			}
			redirect(`connectors/${goToConnectorSettings}?role=${connectionRole}`);
		}
	}, [goToConnectorSettings]);

	const authorizeWithOAuth = async (connectorID: number) => {
		localStorage.setItem('chichi_ui_add_connection_id', String(connectorID));
		localStorage.setItem('chichi_ui_add_connection_role', connectionRole);
		let res: authCodeURLResponse;
		try {
			res = await api.connectors.authCodeURL(connectorID);
		} catch (err) {
			handleError(err);
			return;
		}
		window.location.href = res.url;
		return;
	};

	const onSearchTermUpdate = (e) => {
		const value = e.currentTarget.value;
		setSearchTerm(value);
	};

	const connectorsCards = [];
	for (const c of connectors) {
		if (connectionRole === 'Destination' && (c.type === 'Website' || c.type === 'Mobile' || c.type === 'Server')) {
			continue;
		}
		const name = c.name;
		if (name.toLowerCase().includes(searchTerm.toLowerCase())) {
			connectorsCards.push(
				<Card
					key={c.id}
					name={c.name}
					icon={c.icon}
					type={c.type}
					description={connectionRole === 'Source' ? c.sourceDescription : c.destinationDescription}
				>
					<SlTooltip content={`Add ${c.name}`}>
						<SlButton
							size='medium'
							variant='default'
							onClick={c.oAuth ? () => authorizeWithOAuth(c.id) : () => setGoToConnectorSettings(c.id)}
							circle
						>
							<SlIcon name='plus' />
						</SlButton>
					</SlTooltip>
				</Card>,
			);
		}
	}

	return (
		<div className='connectorsList'>
			<div className='routeContent'>
				<SlInput
					className='searchBar'
					value={searchTerm}
					onSlInput={onSearchTermUpdate}
					placeholder='Search for a connector...'
				>
					<SlIcon name='search' slot='prefix' />
				</SlInput>
				<div className='connectors'>
					{connectorsCards.length > 0 ? (
						connectorsCards
					) : (
						<div className='noConnector'>
							<SlIcon name='exclamation-circle' />
							Nothing found
						</div>
					)}
				</div>
			</div>
		</div>
	);
};

export default ConnectorsList;
