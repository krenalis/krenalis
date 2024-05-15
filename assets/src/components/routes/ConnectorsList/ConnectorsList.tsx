import React, { useState, useContext, useLayoutEffect, useMemo } from 'react';
import './ConnectorsList.css';
import Card from '../../shared/Card/Card';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import { authCodeURLResponse } from '../../../types/external/api';
import { useLocation } from 'react-router-dom';
import { Link } from '../../shared/Link/Link';

const ConnectorsList = () => {
	const [searchTerm, setSearchTerm] = useState<string>('');

	const { api, handleError, connectors, setTitle } = useContext(AppContext);

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

	const authorizeWithOAuth = async (connectorName: string) => {
		localStorage.setItem('chichi_ui_add_connector_name', connectorName);
		localStorage.setItem('chichi_ui_add_connection_role', connectionRole);
		let res: authCodeURLResponse;
		try {
			res = await api.connectors.authCodeURL(connectorName);
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
					key={c.name}
					name={c.name}
					icon={c.icon}
					type={c.type}
					description={connectionRole === 'Source' ? c.sourceDescription : c.destinationDescription}
				>
					<SlTooltip content={`Add ${c.name}`}>
						<Link
							path={
								c.oAuth
									? null
									: c.isFile
										? `connectors/file/${c.name}?role=${connectionRole}`
										: `connectors/${c.name}?role=${connectionRole}`
							}
						>
							<SlButton
								size='medium'
								variant='default'
								onClick={c.oAuth ? () => authorizeWithOAuth(c.name) : null}
								circle
							>
								<SlIcon name='plus' />
							</SlButton>
						</Link>
					</SlTooltip>
				</Card>,
			);
		}
	}

	return (
		<div className='connectors-list'>
			<div className='route-content'>
				<SlInput
					className='connectors-list__search-bar'
					value={searchTerm}
					onSlInput={onSearchTermUpdate}
					placeholder='Search for a connector...'
				>
					<SlIcon name='search' slot='prefix' />
				</SlInput>
				<div className='connectors-list__connectors'>
					{connectorsCards.length > 0 ? (
						connectorsCards
					) : (
						<div className='connectors-list__no-connector'>
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
