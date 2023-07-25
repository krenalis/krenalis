import { useState, useContext } from 'react';
import './ConnectionSettings.css';
import Form from './Form';
import Deletion from './Deletion';
import Enabling from './Enabling';
import Keys from './Keys';
import Storage from './Storage';
import Snippet from './Snippet';
import { AppContext } from '../../../context/providers/AppProvider';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { SlTab, SlTabGroup, SlTabPanel } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionSettings = () => {
	const [isDeleted, setIsDeleted] = useState(false);

	const { redirect } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	if (isDeleted) {
		redirect('connections');
		return;
	}

	return (
		<div className='connectionSettings'>
			<SlTabGroup className='settings' placement='start'>
				{(c.type === 'Website' || c.type === 'Mobile') && c.role === 'Source' && (
					<>
						<SlTab slot='nav' panel='snippet'>
							Snippet
						</SlTab>
						<SlTabPanel name='snippet'>
							<div className='panelTitle'>Snippet</div>
							<Snippet />
						</SlTabPanel>
					</>
				)}

				{c.hasSettings && (
					<>
						<SlTab slot='nav' panel='connection'>
							Connection
						</SlTab>
						<SlTabPanel name='connection'>
							<div className='panelTitle'>Connection</div>
							<Form connection={c} />
						</SlTabPanel>
					</>
				)}

				{c.type === 'Server' && c.role === 'Source' && (
					<>
						<SlTab slot='nav' panel='apiKeys'>
							API Keys
						</SlTab>
						<SlTabPanel name='apiKeys'>
							<div className='panelTitle'>API Keys</div>
							<Keys connection={c} />
						</SlTabPanel>
					</>
				)}

				{c.type === 'File' && (
					<>
						<SlTab slot='nav' panel='storage'>
							Storage
						</SlTab>
						<SlTabPanel name='storage'>
							<div className='panelTitle'>Storage</div>
							<Storage connection={c} />
						</SlTabPanel>
					</>
				)}

				<SlTab slot='nav' panel='enabling'>
					Enabling
				</SlTab>
				<SlTabPanel name='enabling'>
					<div className='panelTitle'>Enabling</div>
					<Enabling connection={c} />
				</SlTabPanel>

				<SlTab slot='nav' panel='deletion'>
					Deletion
				</SlTab>
				<SlTabPanel name='deletion'>
					<div className='panelTitle'>Deletion</div>
					<Deletion connection={c} onDelete={() => setIsDeleted(true)} />
				</SlTabPanel>
			</SlTabGroup>
		</div>
	);
};

export default ConnectionSettings;
