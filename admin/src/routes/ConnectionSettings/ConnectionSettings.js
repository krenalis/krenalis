import { useState, useContext } from 'react';
import './ConnectionSettings.css';
import ConnectionForm from '../../components/ConnectionForm/ConnectionForm';
import ConnectionDeletion from '../../components/ConnectionDeletion/ConnectionDeletion';
import ConnectionEnabling from '../../components/ConnectionEnabling/ConnectionEnabling';
import ConnectionKeys from '../../components/ConnectionKeys/ConnectionKeys';
import ConnectionStream from '../../components/ConnectionStream/ConnectionStream';
import ConnectionStorage from '../../components/ConnectionStorage/ConnectionStorage';
import { AppContext } from '../../context/AppContext';
import { SlTab, SlTabGroup, SlTabPanel } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionSettings = ({ connection: c, onConnectionChange, isSelected }) => {
	let [isDeleted, setIsDeleted] = useState(false);

	let { redirect } = useContext(AppContext);

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
							<ConnectionStorage connection={c} onConnectionChange={onConnectionChange} />
						</SlTabPanel>
					</>
				)}
				{(c.Type === 'Mobile' || c.Type === 'Website' || c.Type === 'Server') && c.Role === 'Source' && (
					<>
						<SlTab slot='nav' panel='stream'>
							Stream
						</SlTab>
						<SlTabPanel name='stream'>
							<div className='panelTitle'>Stream</div>
							<ConnectionStream connection={c} onConnectionChange={onConnectionChange} />
						</SlTabPanel>
					</>
				)}
				<SlTab slot='nav' panel='enabling'>
					Enabling
				</SlTab>
				<SlTabPanel name='enabling'>
					<div className='panelTitle'>Enabling</div>
					<ConnectionEnabling connection={c} onConnectionChange={onConnectionChange} />
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
