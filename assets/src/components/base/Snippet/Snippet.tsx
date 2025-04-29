import React, { useContext, useEffect, useMemo, useState } from 'react';
import './Snippet.css';
import AppContext from '../../../context/AppContext';
import { NotFoundError } from '../../../lib/api/errors';
import { SNIPPET } from '../../../constants/javascriptSnippet';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';
import SyntaxHighlight from '../SyntaxHighlight/SyntaxHighlight';

interface SnippetProps {
	connectionID: number;
}

const Snippet = ({ connectionID }: SnippetProps) => {
	const [keys, setKeys] = useState<string[]>([]);
	const [eventURL, setEventURL] = useState<string>();
	const [cdnURL, setCDNURL] = useState<string>();

	const { api, handleError, redirect } = useContext(AppContext);

	// Retrieve the CDN URL.
	useEffect(() => {
		const fetchCDNURL = async () => {
			let cdnURL: string;
			try {
				cdnURL = await api.cdnURL();
			} catch (err) {
				setTimeout(() => {
					handleError(err);
				}, 300);
				return;
			}
			setCDNURL(cdnURL);
			return;
		};
		fetchCDNURL();
	}, []);

	// Retrieve the event URL.
	useEffect(() => {
		const fetchEventURL = async () => {
			let eventURL: string;
			try {
				eventURL = await api.eventURL();
			} catch (err) {
				setTimeout(() => {
					handleError(err);
				}, 300);
				return;
			}
			setEventURL(eventURL);
			return;
		};
		fetchEventURL();
	}, []);

	useEffect(() => {
		const fetchKeys = async () => {
			let keys: string[];
			try {
				keys = await api.workspaces.connections.eventWriteKeys(connectionID);
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
	}, [connectionID]);

	const snippet = useMemo<string>(() => {
		const r1 = SNIPPET.replace('"writekey"', `"${keys[0]}"`);
		const r2 = r1.replace('"endpoint"', `"${eventURL}"`);
		const r3 = r2.replace('"/javascript-sdk/dist/meergo.min.js"', `"${cdnURL}/javascript-sdk/dist/meergo.min.js"`);
		return r3;
	}, [SNIPPET, keys, eventURL]);

	return (
		<div className='snippet'>
			<SyntaxHighlight language='html'>{snippet}</SyntaxHighlight>
			<SlCopyButton value={snippet} />
		</div>
	);
};

export { Snippet };
