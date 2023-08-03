import { useState, useContext } from 'react';
import Flex from '../../shared/Flex/Flex';
import { AppContext } from '../../../context/providers/AppProvider';
import statuses from '../../../constants/statuses';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const Storage = ({ connection: c }) => {
	const [showStorages, setShowStorages] = useState(false);

	const { api, redirect, showError, showStatus, connections, setAreConnectionsStale } = useContext(AppContext);

	const onChangeStorage = async (storage) => {
		try {
			await api.connections.setStorage(c.id, storage, '');
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'StorageNotExists') {
					showStatus(statuses.storageNotExist);
				}
				return;
			}
			setShowStorages(false);
			showError(err);
			return;
		}
		setShowStorages(false);
		const cn = { ...c };
		cn.storage = storage;
		setAreConnectionsStale(true);
	};

	const onRemoveStorage = async () => {
		try {
			await api.connections.setStorage(c.id, 0, '');
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			showError(err);
			return;
		}
		const cn = { ...c };
		cn.storage = 0;
		setAreConnectionsStale(true);
	};

	const storages = [];
	for (const cn of connections) {
		if (cn.type === 'Storage' && cn.role === c.role) {
			storages.push(cn);
		}
	}

	const currentStorage = storages.find((s) => s.id === c.storage);
	const dialogStorages = storages.filter((s) => s.id !== c.storage);

	return (
		<>
			{currentStorage && (
				<>
					<div className='storage'>{currentStorage.name}</div>
					<SlButton variant='danger' className='deleteConnectionButton' onClick={onRemoveStorage}>
						Remove
					</SlButton>
				</>
			)}
			<SlButton variant='neutral' onClick={() => setShowStorages(true)}>
				{c.storage === 0 ? 'Add a storage' : 'Change the storage'}
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
						<Flex className='storageItem' alignItems='center' justifyContent='space-between' gap={20}>
							<div className='name'>{s.name}</div>
							<SlButton
								variant='primary'
								onClick={async () => {
									await onChangeStorage(s.id);
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

export default Storage;
