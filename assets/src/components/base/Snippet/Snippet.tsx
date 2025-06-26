import React, { useContext, useEffect, useMemo, useState } from 'react';
import './Snippet.css';
import AppContext from '../../../context/AppContext';
import { NotFoundError } from '../../../lib/api/errors';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';

import SyntaxHighlight from '../SyntaxHighlight/SyntaxHighlight';
import Section from '../../base/Section/Section';

interface SnippetProps {
	connectorName: string;
	connectionID: number;
}

interface SnippetFile {
	SNIPPET: string;
	INSTALL_COMMAND?: string;
	DOCUMENTATION_LINK: string;
}

const Snippet = ({ connectorName, connectionID }: SnippetProps) => {
	const [keys, setKeys] = useState<string[]>([]);
	const [eventURL, setEventURL] = useState<string>();
	const [javaScriptSDKURL, setJavaScriptSDKURL] = useState<string>();
	const [snippet, setSnippet] = useState<string>();
	const [installCommand, setInstallCommand] = useState<string>();
	const [documentationLink, setDocumentationLink] = useState<string>();

	const { api, handleError, redirect } = useContext(AppContext);

	useEffect(() => {
		import(`../../../constants/snippets/${connectorName.toLowerCase().replace(/\./g, '')}.ts`).then(
			(file: SnippetFile) => {
				setSnippet(file.SNIPPET);
				setInstallCommand(file.INSTALL_COMMAND);
				setDocumentationLink(file.DOCUMENTATION_LINK);
			},
		);
	}, [connectorName]);

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

	// Retrieve the JavaScript SDK URL.
	useEffect(() => {
		const fetchJavaScriptSDKURL = async () => {
			let javaScriptSDKURL: string;
			try {
				javaScriptSDKURL = await api.javaScriptSDKURL();
			} catch (err) {
				setTimeout(() => {
					handleError(err);
				}, 300);
				return;
			}
			setJavaScriptSDKURL(javaScriptSDKURL);
			return;
		};
		fetchJavaScriptSDKURL();
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

	const completeSnippet = useMemo<string>(() => {
		if (snippet == null) {
			return '';
		}
		const s1 = snippet.replace('"writekey"', JSON.stringify(keys[0]));
		const s2 = s1.replace('"endpoint"', JSON.stringify(eventURL));
		let s3 = s2;
		if (connectorName === 'Javascript') {
			s3 = s2.replace('"javaScriptSDKURL"', JSON.stringify(javaScriptSDKURL));
		}
		return s3;
	}, [connectorName, snippet, keys, eventURL]);

	let applicationType = 'server';
	if (connectorName === 'Android' || connectorName === 'Apple') {
		applicationType = 'app';
	} else if (connectorName === 'JavaScript') {
		applicationType = 'website';
	}

	let language = 'html';
	if (connectorName === 'Python') {
		language = 'python';
	} else if (connectorName === 'Go') {
		language = 'go';
	} else if (connectorName === '.NET') {
		language = 'csharp';
	} else if (connectorName === 'Android') {
		language = 'kotlin';
	} else if (connectorName === 'Java') {
		language = 'java';
	} else if (connectorName === 'Node.js') {
		language = 'javascript';
	}

	return (
		<Section
			title={`Add Meergo to your ${applicationType}`}
			className='connection-actions__instructions'
			description={
				<div className='connection-actions__instructions-text'>
					Copy this snippet and paste it into your {applicationType} to receive events
					<a target='_blank' href={documentationLink}>
						See documentation
					</a>
				</div>
			}
			annotated={true}
		>
			<div className={`snippet${installCommand != null ? ' snippet--command' : ''}`}>
				{installCommand != null && (
					<>
						<SyntaxHighlight
							className='syntax-highlight--install-command'
							language={connectorName === 'Java' || connectorName === 'Android' ? 'markdown' : 'bash'}
							icon={connectorName === 'Java' || connectorName === 'Android' ? 'info-circle' : 'terminal'}
						>
							{installCommand}
						</SyntaxHighlight>
						{connectorName !== 'Java' && connectorName !== 'Android' && (
							<SlCopyButton value={installCommand} />
						)}
					</>
				)}
				<SyntaxHighlight className='syntax-highlight--snippet' language={language}>
					{completeSnippet}
				</SyntaxHighlight>
				<SlCopyButton value={completeSnippet} />
			</div>
		</Section>
	);
};

export { Snippet };
