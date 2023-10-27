import React, { useContext, useState, useEffect } from 'react';
import { NotFoundError } from '../../../lib/api/errors';
import statuses from '../../../constants/statuses';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { AppContext } from '../../../context/providers/AppProvider';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';

const ConnectionSnippet = () => {
	const [keys, setKeys] = useState<string[]>([]);

	const { api, showStatus, showError, redirect } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

	useEffect(() => {
		const fetchKeys = async () => {
			let keys: string[];
			try {
				keys = await api.workspaces.connections.keys(c.id);
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

	return (
		<>
			<div>You can use one of the API keys of the connection to setup your source:</div>
			<div className='snippetCopy'>
				<SlInput readonly value={keys[0]} />
				<SlCopyButton value={keys[0]} />
			</div>
		</>
	);
};

export default ConnectionSnippet;
