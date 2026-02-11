import React, { useContext, useEffect, useMemo, useState } from 'react';
import './Snippet.css';
import AppContext from '../../../context/AppContext';
import { NotFoundError } from '../../../lib/api/errors';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';

import SyntaxHighlight from '../SyntaxHighlight/SyntaxHighlight';
import Section from '../../base/Section/Section';

interface SnippetProps {
	connectorCode: string;
	connectionID: number;
}

interface SnippetFile {
	SNIPPET: string;
	INSTALL_COMMAND?: string;
	DOCUMENTATION_LINK: string;
}

const Snippet = ({ connectorCode, connectionID }: SnippetProps) => {
	const [keys, setKeys] = useState<string[]>([]);
	const [snippet, setSnippet] = useState<string>();
	const [installCommand, setInstallCommand] = useState<string>();
	const [documentationLink, setDocumentationLink] = useState<string>();

	const { api, handleError, redirect, publicMetadata, connectors } = useContext(AppContext);

	useEffect(() => {
		import(`../../../constants/snippets/${connectorCode.toLowerCase().replace(/\./g, '')}.ts`).then(
			(file: SnippetFile) => {
				setSnippet(file.SNIPPET);
				setInstallCommand(file.INSTALL_COMMAND);
				setDocumentationLink(file.DOCUMENTATION_LINK);
			},
		);
	}, [connectorCode]);

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
		const s2 = s1.replace('"endpoint"', JSON.stringify(publicMetadata.externalEventURL));
		let s3 = s2;
		if (connectorCode === 'javascript') {
			s3 = s2.replace('"javaScriptSDKURL"', JSON.stringify(publicMetadata.javascriptSDKURL));
		}
		return s3;
	}, [connectorCode, snippet, keys, publicMetadata.externalEventURL]);

	const connectorLabelForDocs = useMemo<string>(() => {
		return connectors.find((c) => c.code === connectorCode)?.label ?? connectorCode;
	}, [connectors, connectorCode]);

	let applicationType = 'server';
	if (connectorCode === 'android' || connectorCode === 'apple') {
		applicationType = 'app';
	} else if (connectorCode === 'javascript') {
		applicationType = 'website';
	}

	let language = 'html';
	if (connectorCode === 'python') {
		language = 'python';
	} else if (connectorCode === 'go') {
		language = 'go';
	} else if (connectorCode === 'dotnet') {
		language = 'csharp';
	} else if (connectorCode === 'android') {
		language = 'kotlin';
	} else if (connectorCode === 'java') {
		language = 'java';
	} else if (connectorCode === 'nodejs') {
		language = 'javascript';
	}

	return (
		<Section
			title={`Add Meergo to your ${applicationType}`}
			className='connection-pipelines__instructions'
			description={
				<>
					<span>Copy this snippet and paste it into your {applicationType} to receive events.</span>
					<a target='_blank' rel='noopener' href={documentationLink}>
						{connectorLabelForDocs} SDK documentation
					</a>
				</>
			}
			annotated={true}
		>
			<div className={`snippet${installCommand != null ? ' snippet--command' : ''}`}>
				{installCommand != null && (
					<>
						<SyntaxHighlight
							className='syntax-highlight--install-command'
							language={connectorCode === 'java' || connectorCode === 'android' ? 'markdown' : 'bash'}
							icon={connectorCode === 'java' || connectorCode === 'android' ? 'info-circle' : 'terminal'}
						>
							{installCommand}
						</SyntaxHighlight>
						{connectorCode !== 'java' && connectorCode !== 'android' && (
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
