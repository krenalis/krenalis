import { useState, useEffect } from 'react';
import './ConnectionStorage.css';
import FlexContainer from '../FlexContainer/FlexContainer';
import call from '../../utils/call';
import { SlButton, SlIcon, SlDialog } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionStorage = ({ connection: c, onConnectionChange, onError }) => {
	let [storages, setStorages] = useState([]);
	let [showStorages, setShowStorages] = useState(false);

	useEffect(() => {
		const fetchStreams = async () => {
			let [connections, err] = await call('/admin/connections/find', 'GET');
			if (err) {
				onError(err);
				return;
			}
			let storages = [];
			for (let cn of connections) {
				if (cn.Type === 'Storage' && cn.Role === c.Role) {
					storages.push(cn);
				}
			}
			setStorages(storages);
		};
		fetchStreams();
	}, []);

	const onChangeStorage = async (storage) => {
		let [, err] = await call(`/api/connections/${c.ID}/storage/${storage}`, 'PUT');
		if (err !== null) {
			onError(err);
			setShowStorages(false);
			return;
		}
		let cn = { ...c };
		cn.Storage = storage;
		setShowStorages(false);
		onConnectionChange(cn);
	};

	const onRemoveStorage = async () => {
		let [, err] = await call(`/api/connections/${c.ID}/storage/0`, 'PUT');
		if (err !== null) {
			onError(err);
			return;
		}
		let cn = { ...c };
		cn.Storage = 0;
		onConnectionChange(cn);
	};

	let currentStorage = storages.find((s) => s.ID === c.Storage);
	let dialogStorages = storages.filter((s) => s.ID !== c.Storage);

	return (
		<>
			{currentStorage && (
				<>
					<FlexContainer className='storageContainer' alignItems='center' gap={30}>
						<div className='storage'>{currentStorage.Name}</div>
						<SlButton variant='danger' onClick={onRemoveStorage}>
							<SlIcon slot='prefix' name='x' />
							Remove
						</SlButton>
					</FlexContainer>
				</>
			)}
			<SlButton variant='neutral' onClick={() => setShowStorages(true)}>
				<SlIcon slot='prefix' name={c.Storage === 0 ? 'plus' : 'pencil-fill'} />
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
					<div className='noStream'>No Storage available</div>
				) : (
					dialogStorages.map((s) => (
						<FlexContainer className='storage' alignItems='center' justifyContent='space-between' gap={20}>
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
						</FlexContainer>
					))
				)}
			</SlDialog>
		</>
	);
};

export default ConnectionStorage;
