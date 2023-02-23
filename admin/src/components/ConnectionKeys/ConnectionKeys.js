import { useState, useEffect, useContext } from 'react';
import './ConnectionKeys.css';
import Flex from '../Flex/Flex';
import { AppContext } from '../../context/AppContext';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import statuses from '../../constants/statuses';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionKeys = ({ connection: c }) => {
	let [keys, setKeys] = useState([]);

	const { API, showStatus, showError, redirect } = useContext(AppContext);

	useEffect(() => {
		const fetchKeys = async () => {
			let [keys, err] = await API.connections.keys(c.ID);
			if (err) {
				if (err instanceof NotFoundError) {
					redirect('/admin/connections');
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
		let [key, err] = await API.connections.generateKey(c.ID);
		if (err) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
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
		let ks = [...keys, key];
		setKeys(ks);
	};

	const onRevokeKey = async (key) => {
		let [, err] = await API.connections.revokeKey(c.ID, key);
		if (err) {
			if (err instanceof NotFoundError) {
				redirect('/admin/connections');
				showStatus(statuses.connectionDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError && err.code !== 'KeyNotExist') {
				if (err.code === 'UniqueKey') {
					showStatus(statuses.uniqueKey);
				}
				return;
			}
			if (err.code !== 'KeyNotExist') {
				showError(err);
				return;
			}
			// if the error code is 'KeyNotExist', let the key be removed from
			// the UI without showing errors.
		}
		let ks = [];
		for (let k of keys) {
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
								<SlIcon slot='prefix' name='x' />
								Revoke
							</SlButton>
						</Flex>
					);
				})}
			</div>
			<SlButton variant='neutral' onClick={onAddKey}>
				<SlIcon slot='prefix' name='plus' />
				Generate new Key
			</SlButton>
		</>
	);
};

export default ConnectionKeys;
