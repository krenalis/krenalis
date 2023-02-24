import { useState, useEffect, useContext } from 'react';
import './OAuth.css';
import PrimaryBackground from '../PrimaryBackground/PrimaryBackground';
import { AppContext } from '../../context/AppContext';
import { SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

const OAuth = () => {
	let [hasError, setHasError] = useState(false);
	let [redirectURL, setRedirectURL] = useState('');

	let { API, redirect } = useContext(AppContext);

	useEffect(() => {
		const fetchOAuthToken = async () => {
			let connectorID = localStorage.getItem('addConnectionID');
			let connectionRole = localStorage.getItem('addConnectionRole');
			localStorage.removeItem('addConnectionID');
			localStorage.removeItem('addConnectionRole');
			let url = new URL(document.location);
			let oauthCode = url.searchParams.get('oauthCode');
			let [oauthToken, err] = await API.workspace.oauthToken(Number(connectorID), oauthCode);
			if (err != null) {
				console.error(err);
				setHasError(true);
			}
			setTimeout(() => {
				setRedirectURL(`/admin/connectors/${connectorID}?role=${connectionRole}&oauthToken=${oauthToken}`);
			}, 1000);
		};
		fetchOAuthToken();
	}, []);

	if (hasError) {
		redirect('/admin/oauth/error');
		return;
	}

	if (redirectURL !== '') {
		redirect(redirectURL);
		return;
	}

	return (
		<div className='OAuth'>
			<PrimaryBackground height={300} overlap={130}></PrimaryBackground>
			<div className='loading'>
				<div className='text'>Finalizing the OAuth authentication...</div>
				<SlSpinner
					style={{
						fontSize: '3rem',
						'--track-width': '6px',
					}}
				/>
			</div>
		</div>
	);
};

export default OAuth;
