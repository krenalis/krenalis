import React, { useState, useEffect, useContext } from 'react';
import Flex from '../../shared/Flex/Flex';
import { AppContext } from '../../../context/providers/AppProvider';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import TransformedConnection from '../../../lib/helpers/transformedConnection';

interface KeysProps {
	connection: TransformedConnection;
}

const Keys = ({ connection: c }: KeysProps) => {
	const [keys, setKeys] = useState<string[]>([]);

	const { api, showStatus, showError, redirect } = useContext(AppContext);

	useEffect(() => {
		const fetchKeys = async () => {
			let keys: string[];
			try {
				keys = await api.workspace.connections.keys(c.id);
			} catch (err) {
				if (err instanceof NotFoundError) {
					redirect('connections');
					showStatus(statuses.connectionDoesNotExistAnymore);
					return;
				}
				showError(err);
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
			key = await api.workspace.connections.generateKey(c.id);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'TooManyKeys') {
					showStatus(statuses.tooManyKeys);
				}
				return;
			}
			showError(err);
			return;
		}
		const ks = [...keys, key];
		setKeys(ks);
	};

	const onRevokeKey = async (key: string) => {
		try {
			await api.workspace.connections.revokeKey(c.id, key);
		} catch (err) {
			if (err instanceof NotFoundError) {
				redirect('connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError && err.code !== 'KeyNotExists') {
				if (err.code === 'UniqueKey') {
					showStatus(statuses.uniqueKey);
				}
				return;
			}
			if (err.code !== 'KeyNotExists') {
				showError(err);
				return;
			}
			// if the error code is 'KeyNotExists', const the key be removed from
			// the UI without showing errors.
		}
		const ks: string[] = [];
		for (const k of keys) {
			if (k !== key) ks.push(k);
		}
		setKeys(ks);
	};

	return (
		<>
			<div className='keys'>
				{keys.map((key) => {
					return (
						<Flex alignItems='center' gap={30}>
							<div className='key'>{key}</div>
							<SlButton variant='danger' onClick={() => onRevokeKey(key)}>
								Revoke
							</SlButton>
						</Flex>
					);
				})}
			</div>
			<SlButton variant='neutral' onClick={onAddKey}>
				Generate new Key
			</SlButton>
		</>
	);
};

export default Keys;
