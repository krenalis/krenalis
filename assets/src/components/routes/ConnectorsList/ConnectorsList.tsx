import React, { useState, useContext, useLayoutEffect, useMemo } from 'react';
import './ConnectorsList.css';
import { Role } from '../../../lib/api/types/types';
import AppContext from '../../../context/AppContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import { authCodeURLResponse } from '../../../lib/api/types/responses';
import { useLocation } from 'react-router-dom';
import TransformedConnector from '../../../lib/core/connector';

const ConnectorsList = () => {
	const [searchTerm, setSearchTerm] = useState<string>('');

	const { api, handleError, connectors, setTitle, redirect } = useContext(AppContext);

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

	const onConnectorClick = async (connector: TransformedConnector) => {
		if (connector.isStream) {
			// Stream connectors are not available yet.
			return;
		}
		if (connector.requiresAuth) {
			localStorage.setItem('meergo_ui_add_connector_name', connector.name);
			localStorage.setItem('meergo_ui_add_connection_role', connectionRole);
			let res: authCodeURLResponse;
			try {
				res = await api.connectors.authCodeURL(connector.name, connectionRole as Role);
			} catch (err) {
				handleError(err);
				return;
			}
			window.location.href = res.url;
			return;
		}
		if (connector.isFile) {
			redirect(`connectors/file/${connector.name}?role=${connectionRole}`);
		} else {
			redirect(`connectors/${connector.name}?role=${connectionRole}`);
		}
	};

	const onSearchTermUpdate = (e) => {
		const value = e.currentTarget.value;
		setSearchTerm(value);
	};

	const connectorsCards = [];
	for (const c of connectors) {
		if (
			(connectionRole === 'Source' && c.asSource == null) ||
			(connectionRole === 'Destination' && c.asDestination == null)
		) {
			continue;
		}
		const name = c.name;
		if (name.toLowerCase().includes(searchTerm.toLowerCase())) {
			let card = (
				<div
					className={`connectors-list__card${c.isStream ? ' connectors-list__card--disabled' : ''}`}
					key={c.name}
					data-name={c.name}
					onClick={() => onConnectorClick(c)}
				>
					<div className='connectors-list__card-top'>
						<div className='connectors-list__card-logo' dangerouslySetInnerHTML={{ __html: c.icon }} />
						<div className='connectors-list__card-name'>{name}</div>
						{c.type && (
							<SlBadge className='connectors-list__card-type' variant='neutral'>
								{c.type}
							</SlBadge>
						)}
						<div className='connectors-list__card-description'>
							{connectionRole === 'Source' ? c.asSource.description : c.asDestination.description}
						</div>
					</div>
				</div>
			);
			if (c.isStream) {
				connectorsCards.push(
					<SlTooltip placement='top' content={'Stream connectors will be available soon'}>
						{card}
					</SlTooltip>,
				);
			} else {
				connectorsCards.push(card);
			}
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
				{connectorsCards.length > 0 ? (
					<div className='connectors-list__connectors'>{connectorsCards}</div>
				) : (
					<div className='connectors-list__no-connector'>
						<SlIcon name='exclamation-circle' />
						Nothing found
					</div>
				)}
			</div>
		</div>
	);
};

export default ConnectorsList;
