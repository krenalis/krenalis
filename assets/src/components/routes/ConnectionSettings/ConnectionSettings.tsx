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
import { ConnectorUIResponse } from '../../../lib/api/types/responses';
import { debounce } from '../../../utils/debounce';

type TabName = 'general' | 'snippet' | 'connection' | 'keys';

const ConnectionSettings = () => {
	const [isDeleted, setIsDeleted] = useState<boolean>(false);
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isSmallViewport, setIsSmallViewport] = useState<boolean>(false);
	const [hasUIFields, setHasUIFields] = useState<boolean>(false);

	const { redirect, api, handleError } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

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
			if (ui.fields.length === 0) {
				setHasUIFields(false);
			} else {
				setHasUIFields(true);
			}
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		if (c.hasSettings) {
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

		if ((c.connector.type === 'Website' || c.connector.type === 'Mobile') && c.role === 'Source') {
			tabs.push('snippet');
		}

		if (hasUIFields) {
			tabs.push('connection');
		}

		if (
			(c.connector.type === 'Mobile' || c.connector.type === 'Server' || c.connector.type === 'Website') &&
			c.role === 'Source'
		) {
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
			<SlTabGroup placement={isSmallViewport ? 'top' : 'start'}>
				<SlTab slot='nav' panel='general'>
					General
				</SlTab>
				<SlTabPanel name='general'>
					<div className='connection-settings__panel-title'>General</div>
					<ConnectionGeneralSettings connection={c} onDelete={() => setIsDeleted(true)} />
				</SlTabPanel>

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
							{c.connector.type} Settings
						</SlTab>
						<SlTabPanel name='connection'>
							<div className='connection-settings__panel-title'>{c.connector.type} Settings</div>
							<ConnectionConnectorSettings connection={c} />
						</SlTabPanel>
					</>
				)}

				{tabs.includes('keys') && (
					<>
						<SlTab slot='nav' panel='keys'>
							Event write keys
						</SlTab>
						<SlTabPanel name='keys'>
							<div className='connection-settings__panel-title'>Event write keys</div>
							<ConnectionKeys connection={c} />
						</SlTabPanel>
					</>
				)}
			</SlTabGroup>
		</div>
	);
};

export default ConnectionSettings;
