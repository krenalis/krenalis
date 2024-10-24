import React, { useState, useContext, useEffect, useMemo } from 'react';
import './ConnectionSettings.css';
import ConnectionGeneralSettings from './ConnectionGeneralSettings';
import ConnectionConnectorSettings from './ConnectionConnectorSettings';
import ConnectionKeys from './ConnectionKeys';
import ConnectionSnippet from './ConnectionSnippet';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import SlTab from '@shoelace-style/shoelace/dist/react/tab/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlTabGroup from '@shoelace-style/shoelace/dist/react/tab-group/index.js';
import SlTabPanel from '@shoelace-style/shoelace/dist/react/tab-panel/index.js';
import { isEventConnection } from '../../../lib/core/connection';
import { LinkedConnections } from './LinkedConnections';
import { ConnectorUIResponse } from '../../../lib/api/types/responses';
import { debounce } from '../../../utils/debounce';

type TabName = 'general' | 'linked-connections' | 'snippet' | 'connection' | 'keys';

const ConnectionSettings = () => {
	const [isDeleted, setIsDeleted] = useState<boolean>(false);
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isSmallViewport, setIsSmallViewport] = useState<boolean>(false);
	const [isLinkedConnectionsPanelShown, setIsLinkedConnectionsPanelShown] = useState<boolean>(false); // used to recompute the event connections grid.
	const [hasUIFields, setHasUIFields] = useState<boolean>(false);

	const { redirect, api, handleError } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	const onTabShow = (e) => {
		if (e.detail.name === 'linked-connections') {
			setIsLinkedConnectionsPanelShown(true);
			return;
		}
		setIsLinkedConnectionsPanelShown(false);
	};

	useEffect(() => {
		if (isDeleted) {
			redirect('connections');
		}
	}, [isDeleted]);

	useEffect(() => {
		const fetchUI = async () => {
			let ui: ConnectorUIResponse;
			try {
				ui = await api.workspaces.connections.ui(c.id);
			} catch (err) {
				setTimeout(() => {
					handleError(err);
					setIsLoading(false);
				}, 300);
				return;
			}
			if (ui.Fields.length === 0) {
				setHasUIFields(false);
			} else {
				setHasUIFields(true);
			}
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		if (c.hasUI) {
			fetchUI();
		} else {
			setIsLoading(false);
		}
	}, [c]);

	useEffect(() => {
		const checkViewport = () => {
			if (window.innerWidth < 1260) {
				setIsSmallViewport(true);
			} else {
				setIsSmallViewport(false);
			}
		};

		const onResize = () => {
			checkViewport();
		};

		const debouncedOnResize = debounce(onResize, 200);
		window.addEventListener('resize', debouncedOnResize);

		checkViewport();

		return () => {
			window.removeEventListener('resize', debouncedOnResize);
		};
	}, []);

	const tabs = useMemo(() => {
		const tabs: TabName[] = ['general'];

		if (isEventConnection(c.role, c.type, c.connector.targets)) {
			tabs.push('linked-connections');
		}

		if ((c.type === 'Website' || c.type === 'Mobile') && c.role === 'Source') {
			tabs.push('snippet');
		}

		if (hasUIFields) {
			tabs.push('connection');
		}

		if ((c.type === 'Mobile' || c.type === 'Server' || c.type === 'Website') && c.role === 'Source') {
			tabs.push('keys');
		}

		return tabs;
	}, [c, hasUIFields]);

	if (isLoading) {
		return (
			<SlSpinner
				className='connection-settings__spinner'
				style={
					{
						fontSize: '3rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			></SlSpinner>
		);
	}

	return (
		<div className={`connection-settings${isSmallViewport ? ' connection-settings--small-viewport' : ''}`}>
			{tabs.length > 1 ? (
				<SlTabGroup onSlTabShow={onTabShow} placement={isSmallViewport ? 'top' : 'start'}>
					<SlTab slot='nav' panel='general'>
						General
					</SlTab>
					<SlTabPanel name='general'>
						<div className='connection-settings__panel-title'>General</div>
						<ConnectionGeneralSettings connection={c} onDelete={() => setIsDeleted(true)} />
					</SlTabPanel>

					{tabs.includes('linked-connections') && (
						<>
							<SlTab slot='nav' panel='linked-connections'>
								{c.isSource ? 'Linked Destinations' : 'Linked Sources'}
							</SlTab>
							<SlTabPanel name='linked-connections'>
								<div className='connection-settings__panel-title'>
									{c.isSource ? 'Linked Destinations' : 'Linked Sources'}
								</div>
								<LinkedConnections connection={c} isShown={isLinkedConnectionsPanelShown} />
							</SlTabPanel>
						</>
					)}

					{tabs.includes('snippet') && (
						<>
							<SlTab slot='nav' panel='snippet'>
								Snippet
							</SlTab>
							<SlTabPanel name='snippet'>
								<div className='connection-settings__panel-title'>Snippet</div>
								<ConnectionSnippet />
							</SlTabPanel>
						</>
					)}

					{tabs.includes('connection') && (
						<>
							<SlTab slot='nav' panel='connection'>
								{c.type} Settings
							</SlTab>
							<SlTabPanel name='connection'>
								<div className='connection-settings__panel-title'>{c.type} Settings</div>
								<ConnectionConnectorSettings connection={c} />
							</SlTabPanel>
						</>
					)}

					{tabs.includes('keys') && (
						<>
							<SlTab slot='nav' panel='keys'>
								Write Keys
							</SlTab>
							<SlTabPanel name='keys'>
								<div className='connection-settings__panel-title'>Write Keys</div>
								<ConnectionKeys connection={c} />
							</SlTabPanel>
						</>
					)}
				</SlTabGroup>
			) : (
				<>
					<div className='connection-settings__panel-title'>General</div>
					<ConnectionGeneralSettings connection={c} onDelete={() => setIsDeleted(true)} />
				</>
			)}
		</div>
	);
};

export default ConnectionSettings;
