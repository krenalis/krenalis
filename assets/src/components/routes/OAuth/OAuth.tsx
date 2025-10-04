import React, { useState, useEffect, useContext } from 'react';
import './OAuth.css';
import AppContext from '../../../context/AppContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { Link } from '../../base/Link/Link';
import { ADD_CONNECTION_ID_KEY, ADD_CONNECTION_ROLE_KEY, ADD_CONNECTOR_CODE_KEY } from '../../../constants/storage';
import { UI_BASE_PATH } from '../../../constants/paths';

const errorMessageByOauthErrorCode = {
	invalid_request:
		'An internal error has occurred. Please try again later and if the issue persists contact our support.',
	unauthorized_client:
		'It looks like something has been misconfigured in your Meergo instance. Please contact your Meergo administrator to solve the issue.',
	access_denied: 'Permission to complete the authentication has not been given.',
	unsupported_response_type:
		'An internal error has occurred. Please try again later and if the issue persists contact our support.',
	invalid_scope:
		"The account with which you are logged in on [app-placeholder] doesn't have the permission to complete the operation.",
	server_error: '[app-placeholder] is temporarily unavailable. Try again later.',
	temporarily_unavailable: '[app-placeholder] is temporarily unavailable. Try again later.',
};

const OAuth = () => {
	const [errorMessage, setErrorMessage] = useState<string>('');
	const [redirectURL, setRedirectURL] = useState<string>('');

	const { api, redirect, connectors } = useContext(AppContext);

	useEffect(() => {
		const fetchOAuthToken = async () => {
			const connectorCode = localStorage.getItem(ADD_CONNECTOR_CODE_KEY);
			const connector = connectors.find((c) => c.code === connectorCode);
			if (connector == null) {
				console.error(`connector with code ${connectorCode} does not exist`);
				setErrorMessage(
					'Something went wrong, please try again and contact the administrator if the error persist',
				);
				return;
			}
			const url = new URL(document.location.href);
			const oauthError = url.searchParams.get('error');
			if (oauthError != null && oauthError !== '') {
				const errorDescription = url.searchParams.get('error_description');
				const errorURI = url.searchParams.get('error_uri');
				const error = `${oauthError}${
					errorDescription != null && errorDescription !== '' ? `\nDescription: ${errorDescription}\n` : ''
				}${errorURI != null && errorURI !== '' ? `\nURI: ${errorURI}\n` : ''}`;
				console.error(error);
				const message = errorMessageByOauthErrorCode[oauthError].replace('[app-placeholder]', connector.label);
				setTimeout(() => {
					setErrorMessage(message);
				}, 1000);
				return;
			}
			const authCode = url.searchParams.get('code');
			if (authCode == null || authCode === '') {
				setErrorMessage(`${connector.label} didn't respond with a valid authentication code.`);
				return;
			}
			const connectionRole = localStorage.getItem(ADD_CONNECTION_ROLE_KEY);
			localStorage.removeItem(ADD_CONNECTION_ID_KEY);
			localStorage.removeItem(ADD_CONNECTION_ROLE_KEY);
			const redirectURI = new URL(`${api.workspaces.origin}${UI_BASE_PATH}oauth/authorize`);
			if (connector.oauth.disallow127_0_0_1 && redirectURI.hostname === '127.0.0.1') {
				redirectURI.hostname = 'localhost';
			} else if (connector.oauth.disallowLocalhost && redirectURI.hostname === 'localhost') {
				redirectURI.hostname = '127.0.0.1';
			}
			let authToken: string;
			try {
				authToken = await api.workspaces.authToken(connectorCode, authCode, redirectURI.toString());
			} catch (err) {
				console.error(err);
				setErrorMessage(
					'An internal error has occurred. Please try again later and if the issue persists contact our support.',
				);
				return;
			}
			setTimeout(() => {
				setRedirectURL(`connectors/${connectorCode}?role=${connectionRole}&authToken=${authToken}`);
			}, 1000);
		};
		fetchOAuthToken();
	}, []);

	useEffect(() => {
		if (redirectURL !== '') {
			redirect(redirectURL);
		}
	}, [redirectURL]);

	return (
		<div className='oauth'>
			{errorMessage !== '' ? (
				<div className='oauth__error'>
					<SlIcon name='exclamation-circle-fill'></SlIcon>
					<div className='oauth__error-text'>{errorMessage}</div>
					<Link path='connections'>
						<SlButton variant='default'>Go to connections map</SlButton>
					</Link>
				</div>
			) : (
				<div className='oauth__loading'>
					<div className='oauth__loading-text'>Finalizing the OAuth authentication...</div>
					<SlSpinner
						style={
							{
								fontSize: '3rem',
								'--track-width': '6px',
							} as React.CSSProperties
						}
					/>
				</div>
			)}
		</div>
	);
};

export default OAuth;
