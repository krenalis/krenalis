import React, { useState, useContext, useEffect } from 'react';
import './ConnectionSettings.css';
import ConnectionGeneralSettings from './ConnectionGeneralSettings';
import ConnectionConnectorSettings from './ConnectionConnectorSettings';
import ConnectionKeys from './ConnectionKeys';
import ConnectionSnippet from './ConnectionSnippet';
import AppContext from '../../../context/AppContext';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import SlTab from '@shoelace-style/shoelace/dist/react/tab/index.js';
import SlTabGroup from '@shoelace-style/shoelace/dist/react/tab-group/index.js';
import SlTabPanel from '@shoelace-style/shoelace/dist/react/tab-panel/index.js';
import { isEventConnection } from '../../../lib/helpers/transformedConnection';
import { EventConnections } from './EventConnections';

const ConnectionSettings = () => {
	const [isDeleted, setIsDeleted] = useState<boolean>(false);
	const [isEventConnectionsPanelShown, setIsEventConnectionsPanelShown] = useState<boolean>(false); // used to recompute the event connections grid.

	const { redirect } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	const onTabShow = (e) => {
		if (e.detail.name === 'event-connections') {
			setIsEventConnectionsPanelShown(true);
			return;
		}
		setIsEventConnectionsPanelShown(false);
	};

	useEffect(() => {
		if (isDeleted) {
			redirect('connections');
		}
	}, [isDeleted]);

	return (
		<div className='connectionSettings'>
			<SlTabGroup onSlTabShow={onTabShow} placement='start'>
				<SlTab slot='nav' panel='general'>
					General
				</SlTab>
				<SlTabPanel name='general'>
					<div className='panelTitle'>General</div>
					<ConnectionGeneralSettings connection={c} onDelete={() => setIsDeleted(true)} />
				</SlTabPanel>

				{isEventConnection(c.role, c.type, c.connector.targets) && (
					<>
						<SlTab slot='nav' panel='event-connections'>
							{c.isSource ? 'Event Destinations' : 'Event Sources'}
						</SlTab>
						<SlTabPanel name='event-connections'>
							<div className='panelTitle'>{c.isSource ? 'Event Destinations' : 'Event Sources'}</div>
							<EventConnections connection={c} isShown={isEventConnectionsPanelShown} />
						</SlTabPanel>
					</>
				)}

				{(c.type === 'Website' || c.type === 'Mobile') && c.role === 'Source' && (
					<>
						<SlTab slot='nav' panel='snippet'>
							Snippet
						</SlTab>
						<SlTabPanel name='snippet'>
							<div className='panelTitle'>Snippet</div>
							<ConnectionSnippet />
						</SlTabPanel>
					</>
				)}

				{c.hasUI && (
					<>
						<SlTab slot='nav' panel='connection'>
							{c.type} Settings
						</SlTab>
						<SlTabPanel name='connection'>
							<div className='panelTitle'>{c.type} Settings</div>
							<ConnectionConnectorSettings connection={c} />
						</SlTabPanel>
					</>
				)}

				{(c.type === 'Mobile' || c.type === 'Server' || c.type === 'Website') && c.role === 'Source' && (
					<>
						<SlTab slot='nav' panel='keys'>
							API Keys
						</SlTab>
						<SlTabPanel name='keys'>
							<div className='panelTitle'>API keys</div>
							<ConnectionKeys connection={c} />
						</SlTabPanel>
					</>
				)}
			</SlTabGroup>
		</div>
	);
};

export default ConnectionSettings;
