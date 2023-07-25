import { useState, useEffect, useContext } from 'react';
import './OAuth.css';
import { AppContext } from '../../../context/providers/AppProvider';
import { SlSpinner, SlIcon, SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const errorMessageByOauthErrorCode = {
	invalid_request:
		'An internal error has occurred. Please try again later and if the issue persists contact our support.',
	unauthorized_client:
		'It looks like something has been misconfigured in your Chichi instance. Please contact your Chichi administrator to solve the issue.',
	access_denied: 'Permission to complete the authentication has not been given.',
	unsupported_response_type:
		'An internal error has occurred. Please try again later and if the issue persists contact our support.',
	invalid_scope:
		"The account with which you are logged in on [app-placeholder] doesn't have the permission to complete the operation.",
	server_error: '[app-placeholder] is temporarely unavailable. Try again later.',
	temporarily_unavailable: '[app-placeholder] is temporarely unavailable. Try again later.',
};

const OAuth = () => {
	const [errorMessage, setErrorMessage] = useState('');
	const [redirectURL, setRedirectURL] = useState('');

	const { api, redirect, connectors } = useContext(AppContext);

	useEffect(() => {
		const fetchOAuthToken = async () => {
			const connectorID = localStorage.getItem('addConnectionID');
			const connector = connectors.find((c) => c.id === Number(connectorID));
			const url = new URL(document.location);
			const oauthError = url.searchParams.get('error');
			if (oauthError != null && oauthError !== '') {
				const errorDescription = url.searchParams.get('error_description');
				const errorURI = url.searchParams.get('error_uri');
				const error = `${oauthError}${
					errorDescription != null && errorDescription !== '' ? `\nDescription: ${errorDescription}\n` : ''
				}${errorURI != null && errorURI !== '' ? `\nURI: ${errorURI}\n` : ''}`;
				console.error(error);
				const message = errorMessageByOauthErrorCode[oauthError].replace('[app-placeholder]', connector.name);
				setTimeout(() => {
					setErrorMessage(message);
				}, 1000);
				return;
			}
			const oauthCode = url.searchParams.get('code');
			if (oauthCode == null || oauthCode === '') {
				setErrorMessage(`${connector.name} didn't respond with a valid authentication code.`);
				return;
			}

			const connectionRole = localStorage.getItem('addConnectionRole');
			localStorage.removeItem('addConnectionID');
			localStorage.removeItem('addConnectionRole');
			const [oauthToken, err] = await api.workspace.oauthToken(Number(connectorID), oauthCode);
			if (err != null) {
				console.error(err);
				setErrorMessage(
					'An internal error has occurred. Please try again later and if the issue persists contact our support.'
				);
				return;
			}
			setTimeout(() => {
				setRedirectURL(`connectors/${connectorID}?role=${connectionRole}&oauthToken=${oauthToken}`);
			}, 1000);
		};
		fetchOAuthToken();
	}, []);

	const onGoToConnectionsMapClick = () => {
		redirect(`connections`);
	};

	if (redirectURL !== '') {
		redirect(redirectURL);
		return;
	}

	return (
		<div className='oauth'>
			{errorMessage !== '' ? (
				<div className='error'>
					<SlIcon name='exclamation-circle-fill'></SlIcon>
					<div className='text'>{errorMessage}</div>
					<SlButton variant='default' onClick={onGoToConnectionsMapClick}>
						Go to connections map
					</SlButton>
				</div>
			) : (
				<div className='loading'>
					<div className='text'>Finalizing the OAuth authentication...</div>
					<SlSpinner
						style={{
							fontSize: '3rem',
							'--track-width': '6px',
						}}
					/>
				</div>
			)}
		</div>
	);
};

export default OAuth;
