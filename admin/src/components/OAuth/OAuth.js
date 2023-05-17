import { useState, useEffect, useContext } from 'react';
import './OAuth.css';
import { AppContext } from '../../context/AppContext';
import { NavLink } from 'react-router-dom';
import { SlSpinner, SlIcon, SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

let errorMessageByOauthErrorCode = {
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
	let [errorMessage, setErrorMessage] = useState('');
	let [redirectURL, setRedirectURL] = useState('');

	let { API, redirect, connectors } = useContext(AppContext);

	useEffect(() => {
		const fetchOAuthToken = async () => {
			let connectorID = localStorage.getItem('addConnectionID');
			let connector = connectors.find((c) => c.ID === Number(connectorID));
			let url = new URL(document.location);
			let oauthError = url.searchParams.get('error');
			if (oauthError != null && oauthError !== '') {
				let errorDescription = url.searchParams.get('error_description');
				let errorURI = url.searchParams.get('error_uri');
				let error = `${oauthError}${
					errorDescription != null && errorDescription !== '' ? `\nDescription: ${errorDescription}\n` : ''
				}${errorURI != null && errorURI !== '' ? `\nURI: ${errorURI}\n` : ''}`;
				console.error(error);
				let message = errorMessageByOauthErrorCode[oauthError].replace('[app-placeholder]', connector.Name);
				setTimeout(() => {
					setErrorMessage(message);
				}, 1000);
				return;
			}
			let oauthCode = url.searchParams.get('code');
			if (oauthCode == null || oauthCode === '') {
				setErrorMessage(`${connector.Name} didn't respond with a valid authentication code.`);
				return;
			}

			let connectionRole = localStorage.getItem('addConnectionRole');
			localStorage.removeItem('addConnectionID');
			localStorage.removeItem('addConnectionRole');
			let [oauthToken, err] = await API.workspace.oauthToken(Number(connectorID), oauthCode);
			if (err != null) {
				console.error(err);
				setErrorMessage(
					'An internal error has occurred. Please try again later and if the issue persists contact our support.'
				);
				return;
			}
			setTimeout(() => {
				setRedirectURL(`/admin/connectors/${connectorID}?role=${connectionRole}&oauthToken=${oauthToken}`);
			}, 1000);
		};
		fetchOAuthToken();
	}, []);

	if (redirectURL !== '') {
		redirect(redirectURL);
		return;
	}

	return (
		<div className='OAuth'>
			{errorMessage !== '' ? (
				<div className='error'>
					<SlIcon name='exclamation-circle-fill'></SlIcon>
					<div className='text'>{errorMessage}</div>
					<SlButton variant='default'>
						Go to connections map
						<NavLink to='/admin/connections'></NavLink>
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
