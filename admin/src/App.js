import { useState, useRef } from 'react';
import './App.css';
import Toast from './components/Toast/Toast';
import API from './api/api';
import { AppContext } from './context/AppContext';
import * as variants from './constants/variants';
import * as icons from './constants/icons';
import { Outlet } from 'react-router-dom';
import { useNavigate } from 'react-router';
import '@shoelace-style/shoelace/dist/themes/light.css';
import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path.js';
import { BadRequestError } from './api/errors';

setBasePath('https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.0.0-beta.85/dist/');

const App = () => {
	let [status, setStatus] = useState(null);

	const toastRef = useRef();
	const navigate = useNavigate();

	const showStatus = ([variant, icon, text]) => {
		setStatus({ variant: variant, icon: icon, text: text });
		return toastRef.current.toast();
	};

	const showError = (err) => {
		if (err instanceof BadRequestError) {
			console.error(`Bad Request: ${err.message}`);
			let message = '';
			if (err.message !== '') {
				message = `[debug mode] Bad Request: ${err.message}`;
			} else {
				message = 'Unexpected error. Contact the administrator for more information.';
			}
			setStatus({
				variant: variants.DANGER,
				icon: icons.EXCLAMATION,
				text: message,
			});
		} else {
			setStatus({ variant: variants.DANGER, icon: icons.EXCLAMATION, text: err });
		}
		toastRef.current.toast();
	};

	const showNotFound = () => {
		return navigate('/admin/not-found');
	};

	const redirect = (url) => {
		return navigate(url);
	};

	// TODO(@Andrea): find a way not to hardcode the URL directly into the
	// javascript.
	let api = new API('https://localhost:9090');

	return (
		<AppContext.Provider
			value={{
				API: api,
				showStatus: showStatus,
				showError: showError,
				showNotFound: showNotFound,
				redirect: redirect,
			}}
		>
			<div className='App'>
				<Outlet />
				<Toast reactRef={toastRef} status={status} />
			</div>
		</AppContext.Provider>
	);
};

export default App;
