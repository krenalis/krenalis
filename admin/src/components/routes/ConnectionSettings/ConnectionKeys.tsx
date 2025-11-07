import React, { useState, useEffect, useContext } from 'react';
import Flex from '../../base/Flex/Flex';
import AppContext from '../../../context/AppContext';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';
import TransformedConnection from '../../../lib/core/connection';

interface KeysProps {
	connection: TransformedConnection;
}

const ConnectionKeys = ({ connection: c }: KeysProps) => {
	const [keys, setKeys] = useState<string[]>([]);

	const { api, handleError, redirect, setIsLoadingConnections } = useContext(AppContext);

	useEffect(() => {
		const fetchKeys = async () => {
			let keys: string[];
			try {
				keys = await api.workspaces.connections.eventWriteKeys(c.id);
			} catch (err) {
				if (err instanceof NotFoundError) {
					redirect('connections');
					handleError('The connection does not exist anymore');
					return;
				}
				handleError(err);
				return;
			}
			setKeys(keys);
			return;
		};
		fetchKeys();
	}, []);

	const onAddKey = async () => {
		let key: string;
		try {
			key = await api.workspaces.connections.createEventWriteKey(c.id);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				handleError('The connection does not exist anymore');
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'TooManyKeys') {
					handleError('The maximum number of event write keys has been reached');
					return;
				}
			}
			handleError(err);
			return;
		}
		const ks = [...keys, key];
		setKeys(ks);
		setIsLoadingConnections(true);
	};

	const onDeleteWriteKey = async (key: string) => {
		try {
			await api.workspaces.connections.deleteEventWriteKey(c.id, key);
		} catch (err) {
			if (err instanceof NotFoundError) {
				// let the key be removed from the UI without showing errors.
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'ConnectionUniqueKey') {
					handleError(err.message);
					return;
				}
			}
			handleError(err);
			return;
		}
		const ks: string[] = [];
		for (const k of keys) {
			if (k !== key) ks.push(k);
		}
		setKeys(ks);
		setIsLoadingConnections(true);
	};

	return (
		<>
			<div className='connection-settings__keys'>
				{keys.map((key) => {
					return (
						<Flex key={key} alignItems='center' gap={30}>
							<div className='connection-settings__key-copy'>
								<SlInput readonly value={key} filled />
								<SlCopyButton value={key} />
							</div>
							<SlButton variant='danger' onClick={() => onDeleteWriteKey(key)}>
								Revoke
							</SlButton>
						</Flex>
					);
				})}
			</div>
			<SlButton variant='neutral' onClick={onAddKey}>
				Generate new key
			</SlButton>
			<div className='connection-settings__keys-endpoint'>
				<div className='connection-settings__keys-endpoint-copy'>
					<SlInput readonly label='Endpoint' value={`${window.location.origin}/api/v1/events`} filled />
					<SlCopyButton value={`${window.location.origin}/api/v1/events`} />
				</div>
			</div>
		</>
	);
};

export default ConnectionKeys;
