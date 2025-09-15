import React, { useState, useContext, useLayoutEffect, useMemo, useEffect } from 'react';
import './ConnectorsList.css';
import { Role } from '../../../lib/api/types/types';
import AppContext from '../../../context/AppContext';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { authCodeURLResponse, ConnectorsInfoResponse } from '../../../lib/api/types/responses';
import { useLocation } from 'react-router-dom';
import TransformedConnector from '../../../lib/core/connector';
import * as marked from 'marked';
import { connectorsInfo } from '../../../lib/api/connectorsInfo';
import { ConnectorInfo } from '../../../lib/api/types/connector';
import { ADD_CONNECTION_ROLE_KEY, ADD_CONNECTOR_NAME_KEY } from '../../../constants/storage';

const ConnectorsList = () => {
	const [additionalConnectorsInfo, setAdditionalConnectorsInfo] = useState<ConnectorInfo[]>([]);
	const [searchTerm, setSearchTerm] = useState<string>('');
	const [selectedConnector, setSelectedConnector] = useState<TransformedConnector>();
	const [isLoadingDocumentation, setIsLoadingDocumentation] = useState<boolean>(false);
	const [documentation, setDocumentation] = useState<string>();
	const [selectedCategory, setSelectedCategory] = useState<string>(
		new URLSearchParams(window.location.search).get('category') ?? 'All',
	);

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

	const searchedConnectors: any[] = useMemo(() => {
		const sortedConnectors = structuredClone(connectors).sort((a, b) => (a.name <= b.name ? -1 : 1));
		const sortedAdditionalConnectorsInfo = structuredClone(additionalConnectorsInfo).sort((a, b) =>
			a.name <= b.name ? -1 : 1,
		);
		let searchedConnectors = [];

		for (const c of [...sortedConnectors, ...sortedAdditionalConnectorsInfo]) {
			if (
				(connectionRole === 'Source' && c.asSource == null) ||
				(connectionRole === 'Destination' && c.asDestination == null)
			) {
				continue;
			}

			const isInfo = c['asSource']?.['implemented'] != null || c['asDestination']?.['implemented'] != null;
			if (isInfo) {
				const isAlreadyInstalled =
					sortedConnectors.findIndex(
						(conn) =>
							conn.name === c.name &&
							((connectionRole === 'Source' && c.asSource != null) ||
								(connectionRole === 'Destination' && c.asDestination != null)),
					) !== -1;
				if (isAlreadyInstalled) {
					continue;
				}
			}

			const name = c.name;
			let nameMatches = name.toLowerCase().includes(searchTerm.toLowerCase());
			let categoriesMatch = c.categories.some((category) =>
				category.toLowerCase().includes(searchTerm.toLowerCase()),
			);
			if (nameMatches || categoriesMatch) {
				searchedConnectors.push(c);
			}
		}
		return searchedConnectors;
	}, [connectors, additionalConnectorsInfo, connectionRole, searchTerm]);

	const categories: string[] = useMemo(() => {
		let categories = [];
		for (const connector of [...connectors, ...additionalConnectorsInfo]) {
			if (
				(connectionRole === 'Source' && connector.asSource == null) ||
				(connectionRole === 'Destination' && connector.asDestination == null)
			) {
				continue;
			}
			for (const category of connector.categories) {
				const isAlreadyIncluded = categories.includes(category);
				if (!isAlreadyIncluded) {
					categories.push(category);
				}
			}
		}
		categories.sort();
		categories.unshift('All');
		return categories;
	}, [connectors, additionalConnectorsInfo]);

	useLayoutEffect(() => {
		setTitle(`Add a ${connectionRole.toLocaleLowerCase()}`);
	}, [connectionRole]);

	useEffect(() => {
		const fetchConnectorsInfo = async () => {
			let res: ConnectorsInfoResponse;
			try {
				res = await connectorsInfo();
			} catch (err) {
				console.error(err);
				return;
			}
			setAdditionalConnectorsInfo(res.connectors);
		};
		fetchConnectorsInfo();
	}, []);

	const onConnectorAdd = async () => {
		let c = selectedConnector;
		setSelectedConnector(null);
		if (c.requiresAuth) {
			if (!c.authConfigured) {
				return;
			}
			localStorage.setItem(ADD_CONNECTOR_NAME_KEY, c.name);
			localStorage.setItem(ADD_CONNECTION_ROLE_KEY, connectionRole);
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

	const onSelectCategory = (category: string) => {
		setSelectedCategory(category);
	};

	const onSearchTermUpdate = (e) => {
		const value = e.currentTarget.value;
		setSearchTerm(value);
	};

	const cards = [];
	for (const c of searchedConnectors) {
		const isInfo = c['asSource']?.['implemented'] != null || c['asDestination']?.['implemented'] != null;
		if (selectedCategory === 'All' || c.categories.includes(selectedCategory)) {
			let card = (
				<ConnectorCard
					connector={!isInfo ? (c as TransformedConnector) : null}
					connectorInfo={isInfo ? (c as ConnectorInfo) : null}
					onClick={onConnectorClick}
					role={connectionRole}
				/>
			);
			cards.push(card);
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
				<div className='connectors-list__categories'>
					{categories.map((c) => {
						return (
							<button
								className={`connectors-list__category${selectedCategory === c ? ' connectors-list__category--selected' : ''}`}
								onClick={() => onSelectCategory(c)}
							>
								{c}
							</button>
						);
					})}
				</div>
				{cards.length > 0 ? (
					<div className='connectors-list__connectors'>
						{cards}
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
					<SlButton
						className='connectors-list__documentation-add'
						variant='primary'
						onClick={onConnectorAdd}
						disabled={selectedConnector?.requiresAuth && !selectedConnector?.authConfigured}
					>
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
					<>
						<div
							className='connectors-list__documentation'
							dangerouslySetInnerHTML={{ __html: documentation }}
						/>
						{selectedConnector?.requiresAuth && !selectedConnector?.authConfigured && (
							<div className='connectors-list__oauth-not-configured'>
								OAuth authentication for this connector is not configured. Please contact your Meergo
								administrator to set it up.{' '}
								<a href='#' target='_blank'>
									Our documentation
								</a>{' '}
								provides instructions on how to configure {selectedConnector.name} OAuth.
							</div>
						)}
					</>
				)}
			</SlDrawer>
		</div>
	);
};

interface ConnectorsCardProps {
	connector: TransformedConnector | null;
	connectorInfo: ConnectorInfo | null;
	onClick?: (c: TransformedConnector) => void;
	role: string;
}

const ConnectorCard = ({ connector, connectorInfo, onClick, role }: ConnectorsCardProps) => {
	if ((connector != null && connectorInfo != null) || (connector == null && connectorInfo == null)) {
		return null;
	}

	if (connector != null) {
		return (
			<div
				className='connectors-list__card'
				key={connector.name}
				data-name={connector.name}
				onClick={() => onClick(connector)}
			>
				<div className='connectors-list__card-beta-label'>Beta</div>
				<div className='connectors-list__card-top'>
					<div className='connectors-list__card-logo' dangerouslySetInnerHTML={{ __html: connector.icon }} />
					<div className='connectors-list__card-name'>{connector.name}</div>
					{connector.categories.map((category, index) => (
						<SlBadge key={index} className='connectors-list__card-type' variant='neutral'>
							{category}
						</SlBadge>
					))}
					<div className='connectors-list__card-summary'>
						{role === 'Source' ? connector.asSource.summary : connector.asDestination.summary}
					</div>
				</div>
			</div>
		);
	} else {
		const isComingSoon =
			(role === 'Source' && connectorInfo.asSource.comingSoon) ||
			(role === 'Destination' && connectorInfo.asDestination.comingSoon);

		const isUnderConsideration =
			(role === 'Source' && !connectorInfo.asSource.implemented) ||
			(role === 'Destination' && !connectorInfo.asDestination.implemented);

		const isInLatestVersion =
			(role === 'Source' && connectorInfo.asSource.implemented) ||
			(role === 'Destination' && connectorInfo.asDestination.implemented);

		return (
			<div
				className={`connectors-list__card connectors-list__card--info`}
				key={connectorInfo.name}
				data-name={connectorInfo.name}
			>
				{isComingSoon ? (
					<div className='connectors-list__card-coming-label'>Coming soon</div>
				) : isUnderConsideration ? (
					<div className='connectors-list__card-coming-label'>Under consideration</div>
				) : null}
				<div className='connectors-list__card-top'>
					<div
						className='connectors-list__card-logo'
						dangerouslySetInnerHTML={{ __html: connectorInfo.icon }}
					/>
					<div className='connectors-list__card-name'>{connectorInfo.name}</div>
					{connectorInfo.categories.map((category, index) => (
						<SlBadge key={index} className='connectors-list__card-type' variant='neutral'>
							{category}
						</SlBadge>
					))}
					<div className='connectors-list__card-summary'>
						{role === 'Source'
							? connectorInfo.asSource.description
							: connectorInfo.asDestination.description}
					</div>
					{isUnderConsideration && (
						<div className='connectors-list__card-contact-us'>Contact us if you are interested</div>
					)}
					{isInLatestVersion && (
						<div className='connectors-list__card-update-version'>
							Update to the latest version to use this connector
						</div>
					)}
				</div>
			</div>
		);
	}
};

export default ConnectorsList;
