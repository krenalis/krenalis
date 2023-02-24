import { useState, useContext } from 'react';
import './ConnectionSettings.css';
import ConnectionForm from '../../components/ConnectionForm/ConnectionForm';
import ConnectionDeletion from '../../components/ConnectionDeletion/ConnectionDeletion';
import ConnectionEnabling from '../../components/ConnectionEnabling/ConnectionEnabling';
import ConnectionKeys from '../../components/ConnectionKeys/ConnectionKeys';
import ConnectionStorage from '../../components/ConnectionStorage/ConnectionStorage';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { SlTab, SlTabGroup, SlTabPanel } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionSettings = () => {
	let [isDeleted, setIsDeleted] = useState(false);

	let { redirect } = useContext(AppContext);
	let { c, setCurrentConnectionSection } = useContext(ConnectionContext);

	setCurrentConnectionSection('settings');

	if (isDeleted) {
		redirect('/admin/connections');
		return;
	}

	return (
		<div className='ConnectionSettings'>
			<SlTabGroup className='settings' placement='start'>
				{c.HasSettings && (
					<>
						<SlTab slot='nav' panel='connection'>
							Connection
						</SlTab>
						<SlTabPanel name='connection'>
							<div className='panelTitle'>Connection</div>
							<ConnectionForm connection={c} />
						</SlTabPanel>
					</>
				)}
				{c.Type === 'Server' && c.Role === 'Source' && (
					<>
						<SlTab slot='nav' panel='apiKeys'>
							API Keys
						</SlTab>
						<SlTabPanel name='apiKeys'>
							<div className='panelTitle'>API Keys</div>
							<ConnectionKeys connection={c} />
						</SlTabPanel>
					</>
				)}
				{c.Type === 'File' && (
					<>
						<SlTab slot='nav' panel='storage'>
							Storage
						</SlTab>
						<SlTabPanel name='storage'>
							<div className='panelTitle'>Storage</div>
							<ConnectionStorage connection={c} />
						</SlTabPanel>
					</>
				)}
				<SlTab slot='nav' panel='enabling'>
					Enabling
				</SlTab>
				<SlTabPanel name='enabling'>
					<div className='panelTitle'>Enabling</div>
					<ConnectionEnabling connection={c} />
				</SlTabPanel>
				<SlTab slot='nav' panel='deletion'>
					Deletion
				</SlTab>
				<SlTabPanel name='deletion'>
					<div className='panelTitle'>Deletion</div>
					<ConnectionDeletion connection={c} onDelete={() => setIsDeleted(true)} />
				</SlTabPanel>
			</SlTabGroup>
		</div>
	);
};

export default ConnectionSettings;
