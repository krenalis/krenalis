import React, { useState, useContext, useLayoutEffect, useMemo } from 'react';
import './ConnectorsList.css';
import { Role } from '../../../lib/api/types/types';
import AppContext from '../../../context/AppContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { authCodeURLResponse } from '../../../lib/api/types/responses';
import { useLocation } from 'react-router-dom';
import TransformedConnector from '../../../lib/core/connector';
import * as marked from 'marked';

const ConnectorsList = () => {
	const [searchTerm, setSearchTerm] = useState<string>('');
	const [selectedConnector, setSelectedConnector] = useState<TransformedConnector>();
	const [isLoadingDocumentation, setIsLoadingDocumentation] = useState<boolean>(false);
	const [documentation, setDocumentation] = useState<string>();

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

	const onConnectorAdd = async () => {
		let c = selectedConnector;
		if (c.isStream) {
			// Stream connectors are not available yet.
			return;
		}
		if (c.requiresAuth) {
			localStorage.setItem('meergo_ui_add_connector_name', c.name);
			localStorage.setItem('meergo_ui_add_connection_role', connectionRole);
			let res: authCodeURLResponse;
			try {
				res = await api.connectors.authCodeURL(c.name, connectionRole as Role);
			} catch (err) {
				handleError(err);
				return;
			}
			window.location.href = res.url;
			return;
		}
		if (c.isFile) {
			redirect(`connectors/file/${c.name}?role=${connectionRole}`);
		} else {
			redirect(`connectors/${c.name}?role=${connectionRole}`);
		}
	};

	const onConnectorClick = async (connector: TransformedConnector) => {
		setSelectedConnector(connector);
		setIsLoadingDocumentation(true);
		let doc: string;
		try {
			const res = await api.connectors.connectorDocumentation(connector.name);
			doc = await marked.parse(res[connectionRole].Overview);
		} catch (err) {
			setSelectedConnector(null);
			setIsLoadingDocumentation(false);
			handleError(err);
			return;
		}
		setDocumentation(doc);
		setIsLoadingDocumentation(false);
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
					<div className='connectors-list__card-beta-label'>BETA</div>
					<div className='connectors-list__card-top'>
						<div className='connectors-list__card-logo' dangerouslySetInnerHTML={{ __html: c.icon }} />
						<div className='connectors-list__card-name'>{name}</div>
						{c.type && (
							<SlBadge className='connectors-list__card-type' variant='neutral'>
								{c.type}
							</SlBadge>
						)}
						<div className='connectors-list__card-summary'>
							{connectionRole === 'Source' ? c.asSource.summary : c.asDestination.summary}
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

	// TODO(@Andrea): add the link to the feedback page on the Meergo
	// website when it is implemented.
	const feedbackMessage = (
		<span className='connectors-list__feedback-message'>
			Can't find the connector you're looking for? <a target='_blank'>Contact us</a>
		</span>
	);

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
					<div className='connectors-list__connectors'>
						{connectorsCards}
						<div className='connectors-list__feedback'>
							<SlIcon name='chat-dots' />
							{feedbackMessage}
						</div>
					</div>
				) : (
					<div className='connectors-list__no-connector'>
						<SlIcon name='exclamation-circle' />
						<div className='connectors-list__no-connector-title'>Nothing found</div>
						<div className='connectors-list__no-connector-feedback'>{feedbackMessage}</div>
					</div>
				)}
			</div>
			<SlDrawer
				style={{ '--size': '600px' } as React.CSSProperties}
				open={selectedConnector != null}
				className='connectors-list__documentation-drawer'
				onSlAfterHide={() => {
					setSelectedConnector(null);
				}}
			>
				<div className='connectors-list__documentation-drawer-label' slot='label'>
					<span>{selectedConnector?.name}</span>
					<SlButton className='connectors-list__documentation-add' variant='primary' onClick={onConnectorAdd}>
						Add {connectionRole.toLowerCase()}...
					</SlButton>
				</div>
				{isLoadingDocumentation ? (
					<SlSpinner
						style={
							{
								fontSize: '3rem',
								'--track-width': '6px',
							} as React.CSSProperties
						}
					/>
				) : (
					<div
						className='connectors-list__documentation'
						dangerouslySetInnerHTML={{ __html: documentation }}
					/>
				)}
			</SlDrawer>
		</div>
	);
};

export default ConnectorsList;
