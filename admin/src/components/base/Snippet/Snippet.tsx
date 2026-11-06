import React, { useContext, useEffect, useMemo, useState } from 'react';
import './Snippet.css';
import AppContext from '../../../context/AppContext';
import { NotFoundError } from '../../../lib/api/errors';
import WaCopyButton from '@awesome.me/webawesome/dist/react/copy-button/index.js';
import WaSelect from '@awesome.me/webawesome/dist/react/select/index.js';
import WaOption from '@awesome.me/webawesome/dist/react/option/index.js';

import SyntaxHighlight from '../SyntaxHighlight/SyntaxHighlight';
import Section from '../../base/Section/Section';
import { JAVASCRIPT_SNIPPET_CMP_KEY } from '../../../constants/storage';

// Consent management platforms (CMPs) supported by the JavaScript snippet.
type CMP = 'none' | 'ketch';

// KETCH_SCRIPT_NAME is the name of the script of the Ketch integration. It is
// published on the same CDN, and in the same directory, of the JavaScript SDK.
const KETCH_SCRIPT_NAME = 'krenalis-ketch.min.js';

// storedCMP returns the CMP stored in the localStorage, or 'none'.
const storedCMP = (): CMP => {
	return localStorage.getItem(JAVASCRIPT_SNIPPET_CMP_KEY) === 'ketch' ? 'ketch' : 'none';
};

interface SnippetProps {
	connectorCode: string;
	connectionID: string;
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
	const [cmp, setCMP] = useState<CMP>(storedCMP);

	const { api, handleError, redirect, publicMetadata, connectors } = useContext(AppContext);

	const onChangeCMP = (e: any) => {
		const value: CMP = e.target.value === 'ketch' ? 'ketch' : 'none';
		setCMP(value);
		if (value === 'none') {
			localStorage.removeItem(JAVASCRIPT_SNIPPET_CMP_KEY);
		} else {
			localStorage.setItem(JAVASCRIPT_SNIPPET_CMP_KEY, value);
		}
	};

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
			if (cmp === 'ketch') {
				// Load the Ketch integration before the snippet, and let it load the SDK.
				const ketchScriptURL = new URL(KETCH_SCRIPT_NAME, publicMetadata.javascriptSDKURL).toString();
				s3 = `<script src=${JSON.stringify(ketchScriptURL)}></script>\n` + s3;
				s3 = s3.replace('krenalis.load(', 'withKetch(krenalis).load(');
			}
		}
		return s3;
	}, [connectorCode, snippet, keys, publicMetadata.externalEventURL, publicMetadata.javascriptSDKURL, cmp]);

	const connectorLabelForDocs = useMemo<string>(() => {
		return connectors.find((c) => c.code === connectorCode)?.label ?? connectorCode;
	}, [connectors, connectorCode]);

	let applicationType = 'server';
	if (connectorCode === 'android' || connectorCode === 'ios') {
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
	} else if (connectorCode === 'ios') {
		language = 'swift';
	}

	const hasManualInstallation = connectorCode === 'java' || connectorCode === 'android' || connectorCode === 'ios';

	return (
		<Section
			title={`Add Krenalis to your ${applicationType}`}
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
			{connectorCode === 'javascript' && (
				<div className='snippet__cmp'>
					{cmp === 'ketch' && (
						<span className='snippet__cmp-note'>
							The Ketch Smart Tag must also be added to your website.
						</span>
					)}
					<span className='snippet__cmp-label'>Consent management</span>
					<WaSelect size='s' value={cmp} onChange={onChangeCMP} className='snippet__cmp-select'>
						<WaOption value='none'>None</WaOption>
						<WaOption value='ketch'>Ketch</WaOption>
					</WaSelect>
				</div>
			)}
			<div className={`snippet${installCommand != null ? ' snippet--command' : ''}`}>
				{installCommand != null && (
					<>
						<SyntaxHighlight
							className='syntax-highlight--install-command'
							language={hasManualInstallation ? 'markdown' : 'bash'}
							icon={hasManualInstallation ? 'info-circle' : 'terminal'}
						>
							{installCommand}
						</SyntaxHighlight>
						{!hasManualInstallation && <WaCopyButton value={installCommand} />}
					</>
				)}
				<SyntaxHighlight className='syntax-highlight--snippet' language={language}>
					{completeSnippet}
				</SyntaxHighlight>
				<WaCopyButton value={completeSnippet} />
			</div>
		</Section>
	);
};

export { Snippet };
