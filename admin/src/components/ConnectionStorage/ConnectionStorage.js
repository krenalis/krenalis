import { useState, useContext } from 'react';
import './ConnectionStorage.css';
import Flex from '../Flex/Flex';
import { AppContext } from '../../context/AppContext';
import { ConnectionContext } from '../../context/ConnectionContext';
import { ConnectionsContext } from '../../context/ConnectionsContext';
import statuses from '../../constants/statuses';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionStorage = ({ connection: c }) => {
	let [showStorages, setShowStorages] = useState(false);

	let { API, redirect, showError, showStatus } = useContext(AppContext);
	let { setConnection } = useContext(ConnectionContext);
	let { connections } = useContext(ConnectionsContext);

	const onChangeStorage = async (storage) => {
		let [, err] = await API.connections.setStorage(c.ID, storage, '');
		setShowStorages(false);
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'StorageNotExists') {
					showStatus(statuses.storageNotExist);
				}
				return;
			}
			showError(err);
			return;
		}
		let cn = { ...c };
		cn.Storage = storage;
		setConnection(cn);
	};

	const onRemoveStorage = async () => {
		let [, err] = await API.connections.setStorage(c.ID, 0, '');
		if (err !== null) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			showError(err);
			return;
		}
		let cn = { ...c };
		cn.Storage = 0;
		setConnection(cn);
	};

	let storages = [];
	for (let cn of connections) {
		if (cn.Type === 'Storage' && cn.Role === c.Role) {
			storages.push(cn);
		}
	}

	let currentStorage = storages.find((s) => s.ID === c.Storage);
	let dialogStorages = storages.filter((s) => s.ID !== c.Storage);

	return (
		<>
			{currentStorage && (
				<>
					<Flex className='storageContainer' alignItems='center' gap={30}>
						<div className='storage'>{currentStorage.Name}</div>
						<SlButton variant='danger' onClick={onRemoveStorage}>
							Remove
						</SlButton>
					</Flex>
				</>
			)}
			<SlButton variant='neutral' onClick={() => setShowStorages(true)}>
				{c.Storage === 0 ? 'Add a storage' : 'Change the storage'}
			</SlButton>
			<SlDialog
				className='storageDialog'
				open={showStorages}
				style={{ '--width': '600px' }}
				onSlAfterHide={() => setShowStorages(false)}
				label={`Select a storage`}
			>
				{dialogStorages.length === 0 ? (
					<div className='noStorage'>No Storage available</div>
				) : (
					dialogStorages.map((s) => (
						<Flex className='storage' alignItems='center' justifyContent='space-between' gap={20}>
							<div className='name'>{s.Name}</div>
							<SlButton
								variant='primary'
								onClick={async () => {
									await onChangeStorage(s.ID);
								}}
								className='changeStorageButton'
							>
								<SlIcon name='arrow-right' />
							</SlButton>
						</Flex>
					))
				)}
			</SlDialog>
		</>
	);
};

export default ConnectionStorage;
