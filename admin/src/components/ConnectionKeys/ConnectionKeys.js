import { useState, useEffect } from 'react';
import './ConnectionKeys.css';
import FlexContainer from '../FlexContainer/FlexContainer';
import call from '../../utils/call';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionKeys = ({ connection: c, onError }) => {
	let [keys, setKeys] = useState([]);

	useEffect(() => {
		const fetchKeys = async () => {
			let [keys, err] = await call(`/api/connections/${c.ID}/keys`, 'GET');
			if (err) {
				onError(err);
				return;
			}
			setKeys(keys);
			return;
		};
		fetchKeys();
	}, []);

	const onRevokeKey = async (key) => {
		let [, err] = await call(`/api/connections/${c.ID}/keys/${key}`, 'DELETE');
		if (err !== null) {
			onError(err);
			return;
		}
		let ks = [];
		for (let k of keys) {
			if (k !== key) ks.push(k);
		}
		setKeys(ks);
	};

	const onAddKey = async () => {
		let [key, err] = await call(`/api/connections/${c.ID}/keys`, 'POST');
		if (err !== null) {
			onError(err);
			return;
		}
		let ks = [...keys, key];
		setKeys(ks);
	};

	return (
		<>
			<div className='keys'>
				{keys.map((key) => {
					return (
						<FlexContainer alignItems='center' gap={30}>
							<div className='key'>{key}</div>
							<SlButton variant='danger' onClick={() => onRevokeKey(key)}>
								<SlIcon slot='prefix' name='x' />
								Revoke
							</SlButton>
						</FlexContainer>
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
