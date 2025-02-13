import React, { useContext, useState, useEffect, useMemo } from 'react';
import { NotFoundError } from '../../../lib/api/errors';
import ConnectionContext from '../../../context/ConnectionContext';
import AppContext from '../../../context/AppContext';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';
import { SNIPPET } from '../../../constants/javascriptSnippet';
import EditorWrapper from '../../base/EditorWrapper/EditorWrapper';

const ConnectionSnippet = () => {
	const [keys, setKeys] = useState<string[]>([]);

	const { api, handleError, redirect } = useContext(AppContext);
	const { connection: c } = useContext(ConnectionContext);

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
	}, [c]);

	const snippet = useMemo<string>(() => {
		const r1 = SNIPPET.replace('"writekey"', `"${keys[0]}"`);
		const r2 = r1.replace('"endpoint"', `"${window.location.origin}/api/v1/events"`);
		return r2;
	}, [SNIPPET, keys]);

	return (
		<>
			<div className='connection-settings__snippet-copy'>
				<div>Embed the snippet in your website to start sending events:</div>
				<EditorWrapper
					name='snippetEditor'
					language='html'
					height={180}
					value={snippet}
					isReadOnly={true}
					hideGutter={true}
				></EditorWrapper>
				<SlCopyButton value={snippet}></SlCopyButton>
			</div>
		</>
	);
};

export default ConnectionSnippet;
