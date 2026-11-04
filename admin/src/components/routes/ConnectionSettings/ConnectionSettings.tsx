import React, { useState, useContext, useEffect, useMemo } from 'react';
import './ConnectionSettings.css';
import ConnectionGeneralSettings from './ConnectionGeneralSettings';
import ConnectionConnectorSettings from './ConnectionConnectorSettings';
import ConnectionKeys from './ConnectionKeys';
import AppContext from '../../../context/AppContext';
import ConnectionContext from '../../../context/ConnectionContext';
import WaTab from '@awesome.me/webawesome/dist/react/tab/index.js';
import WaSpinner from '@awesome.me/webawesome/dist/react/spinner/index.js';
import WaTabGroup from '@awesome.me/webawesome/dist/react/tab-group/index.js';
import WaTabPanel from '@awesome.me/webawesome/dist/react/tab-panel/index.js';
import { ConnectorUIResponse } from '../../../lib/api/types/responses';
import { debounce } from '../../../utils/debounce';
import { Snippet } from '../../base/Snippet/Snippet';

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

		if (c.connector.code === 'javascript') {
			tabs.push('snippet');
		}

		if (hasUIFields) {
			tabs.push('connection');
		}

		if (c.isSource && c.supportsEventTarget) {
			tabs.push('keys');
		}

		return tabs;
	}, [c, hasUIFields]);

	if (isLoading) {
		return (
			<WaSpinner
				className='connection-settings__spinner'
				style={
					{
						fontSize: '3rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			></WaSpinner>
		);
	}

	const connectorTypeHumanReadable = c.connector.type === 'FileStorage' ? 'File storage' : c.connector.type;

	return (
		<div className={`connection-settings${isSmallViewport ? ' connection-settings--small-viewport' : ''}`}>
			<WaTabGroup placement={isSmallViewport ? 'top' : 'start'}>
				<WaTab slot='nav' panel='general'>
					General
				</WaTab>
				<WaTabPanel name='general'>
					<div className='connection-settings__panel-title'>General</div>
					<ConnectionGeneralSettings connection={c} onDelete={() => setIsDeleted(true)} />
				</WaTabPanel>

				{tabs.includes('snippet') && (
					<>
						<WaTab slot='nav' panel='snippet'>
							Snippet
						</WaTab>
						<WaTabPanel name='snippet'>
							<div className='connection-settings__panel-title'>Snippet</div>
							<div className='connection-settings__snippet-copy'>
								<div>Embed the snippet in your website to start sending events:</div>
								<Snippet connectorCode={c.connector.code} connectionID={c.id} />
							</div>
						</WaTabPanel>
					</>
				)}

				{tabs.includes('connection') && (
					<>
						<WaTab slot='nav' panel='connection'>
							{connectorTypeHumanReadable} settings
						</WaTab>
						<WaTabPanel name='connection'>
							<div className='connection-settings__panel-title'>
								{connectorTypeHumanReadable} settings
							</div>
							<ConnectionConnectorSettings connection={c} />
						</WaTabPanel>
					</>
				)}

				{tabs.includes('keys') && (
					<>
						<WaTab slot='nav' panel='keys'>
							Event write keys
						</WaTab>
						<WaTabPanel name='keys'>
							<div className='connection-settings__panel-title'>Event write keys</div>
							<ConnectionKeys connection={c} />
						</WaTabPanel>
					</>
				)}
			</WaTabGroup>
		</div>
	);
};

export default ConnectionSettings;
