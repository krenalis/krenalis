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
import { authCodeURLResponse } from '../../../lib/api/types/responses';
import { useLocation } from 'react-router-dom';
import TransformedConnector from '../../../lib/core/connector';
import * as marked from 'marked';
import { potentialConnectors } from '../../../lib/api/potentialConnectors';
import { PotentialConnector } from '../../../lib/api/types/connector';
import { ADD_CONNECTION_ROLE_KEY, ADD_CONNECTOR_CODE_KEY } from '../../../constants/storage';
import { UI_BASE_PATH } from '../../../constants/paths';
import { ExternalLogo } from '../ExternalLogo/ExternalLogo';

const ConnectorsList = () => {
	const [additionalPotentialConnectors, setAdditionalPotentialConnectors] = useState<PotentialConnector[]>([]);
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

	const existingConnectorCodes = useMemo(() => {
		return new Set(connectors.map((connector) => connector.code));
	}, [connectors]);

	const searchedConnectors: any[] = useMemo(() => {
		const sortedConnectors = connectors.sort((a, b) => (a.label <= b.label ? -1 : 1));
		const sortedAdditionalPotentialConnectors = additionalPotentialConnectors.sort((a, b) =>
			a.label <= b.label ? -1 : 1,
		);
		let searchedConnectors = [];

		for (const c of [...sortedConnectors, ...sortedAdditionalPotentialConnectors]) {
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
							conn.code === c.code &&
							((connectionRole === 'Source' && c.asSource != null) ||
								(connectionRole === 'Destination' && c.asDestination != null)),
					) !== -1;
				if (isAlreadyInstalled) {
					continue;
				}
			}

			const label = c.label;
			let labelMatches = label.toLowerCase().includes(searchTerm.toLowerCase());
			let categoriesMatch = c.categories.some((category) =>
				category.toLowerCase().includes(searchTerm.toLowerCase()),
			);
			if (labelMatches || categoriesMatch) {
				searchedConnectors.push(c);
			}
		}
		return searchedConnectors;
	}, [connectors, additionalPotentialConnectors, connectionRole, searchTerm]);

	const categories: string[] = useMemo(() => {
		let categories = [];
		for (const connector of [...connectors, ...additionalPotentialConnectors]) {
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
	}, [connectors, additionalPotentialConnectors]);

	useLayoutEffect(() => {
		setTitle(`Add a new ${connectionRole.toLocaleLowerCase()}`);
	}, [connectionRole]);

	useEffect(() => {
		const fetchPotentialConnectors = async () => {
			let connectors: PotentialConnector[];
			try {
				connectors = await potentialConnectors(existingConnectorCodes);
			} catch (err) {
				console.error(err);
				return;
			}
			setAdditionalPotentialConnectors(connectors);
		};
		fetchPotentialConnectors();
	}, [existingConnectorCodes]);

	const onConnectorAdd = async () => {
		let c = selectedConnector;
		setSelectedConnector(null);
		if (c.oauth != null) {
			if (!c.oauth.configured) {
				return;
			}
			localStorage.setItem(ADD_CONNECTOR_CODE_KEY, c.code);
			localStorage.setItem(ADD_CONNECTION_ROLE_KEY, connectionRole);
			let res: authCodeURLResponse;
			const redirectURI = new URL(`${api.connectors.origin}${UI_BASE_PATH}oauth/authorize`);
			if (c.oauth.disallow127_0_0_1 && redirectURI.hostname === '127.0.0.1') {
				redirectURI.hostname = 'localhost';
			} else if (c.oauth.disallowLocalhost && redirectURI.hostname === 'localhost') {
				redirectURI.hostname = '127.0.0.1';
			}
			try {
				res = await api.connectors.authCodeURL(c.code, connectionRole as Role, redirectURI.toString());
			} catch (err) {
				handleError(err);
				return;
			}
			window.location.href = res.url;
			return;
		}
		if (c.isFile) {
			redirect(`connectors/file/${c.code}?role=${connectionRole}`);
		} else {
			redirect(`connectors/${c.code}?role=${connectionRole}`);
		}
	};

	const onConnectorClick = async (connector: TransformedConnector) => {
		setSelectedConnector(connector);
		setIsLoadingDocumentation(true);
		let doc: string;
		try {
			const res = await api.connectors.connectorDocumentation(connector.code);
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
		const isPotential = c['asSource']?.['implemented'] != null || c['asDestination']?.['implemented'] != null;
		if (selectedCategory === 'All' || c.categories.includes(selectedCategory)) {
			let card = (
				<ConnectorCard
					connector={!isPotential ? (c as TransformedConnector) : null}
					potentialConnector={isPotential ? (c as PotentialConnector) : null}
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
					<span>{selectedConnector?.label}</span>
					<SlButton
						className='connectors-list__documentation-add'
						variant='primary'
						onClick={onConnectorAdd}
						disabled={selectedConnector?.oauth != null && !selectedConnector?.oauth.configured}
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
						{selectedConnector?.oauth != null && !selectedConnector?.oauth.configured && (
							<div className='connectors-list__oauth-not-configured'>
								OAuth authentication for this connector is not configured. Please contact your Meergo
								administrator to set it up.{' '}
								<a href='#' target='_blank'>
									Our documentation
								</a>{' '}
								provides instructions on how to configure {selectedConnector.label} OAuth.
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
	potentialConnector: PotentialConnector | null;
	onClick?: (c: TransformedConnector) => void;
	role: string;
}

const ConnectorCard = ({ connector, potentialConnector, onClick, role }: ConnectorsCardProps) => {
	if ((connector != null && potentialConnector != null) || (connector == null && potentialConnector == null)) {
		return null;
	}

	if (connector != null) {
		return (
			<div
				className='connectors-list__card'
				key={connector.code}
				data-code={connector.code}
				onClick={() => onClick(connector)}
			>
				<div className='connectors-list__card-beta-label'>Beta</div>
				<div className='connectors-list__card-top'>
					<div className='connectors-list__card-logo'>
						<ExternalLogo code={connector.code} />
					</div>
					<div className='connectors-list__card-label'>{connector.label}</div>
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
			(role === 'Source' && potentialConnector.asSource.comingSoon) ||
			(role === 'Destination' && potentialConnector.asDestination.comingSoon);

		const isUnderConsideration =
			(role === 'Source' && !potentialConnector.asSource.implemented) ||
			(role === 'Destination' && !potentialConnector.asDestination.implemented);

		const isInLatestVersion =
			(role === 'Source' && potentialConnector.asSource.implemented) ||
			(role === 'Destination' && potentialConnector.asDestination.implemented);

		return (
			<div
				className={`connectors-list__card connectors-list__card--potential`}
				key={potentialConnector.code}
				data-code={potentialConnector.code}
			>
				{isComingSoon ? (
					<div className='connectors-list__card-coming-label'>Coming soon</div>
				) : isUnderConsideration ? (
					<div className='connectors-list__card-coming-label'>Under consideration</div>
				) : null}
				<div className='connectors-list__card-top'>
					<div className='connectors-list__card-logo'>
						<ExternalLogo code={potentialConnector.code} />
					</div>
					<div className='connectors-list__card-label'>{potentialConnector.label}</div>
					{potentialConnector.categories.map((category, index) => (
						<SlBadge key={index} className='connectors-list__card-type' variant='neutral'>
							{category}
						</SlBadge>
					))}
					<div className='connectors-list__card-summary'>
						{role === 'Source'
							? potentialConnector.asSource.description
							: potentialConnector.asDestination.description}
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
