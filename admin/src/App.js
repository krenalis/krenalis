import { useState, useEffect, useRef } from 'react';
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
	let [connectors, setConnectors] = useState(null);

	const appRef = useRef();
	const toastRef = useRef();
	const navigate = useNavigate();

	const showStatus = ([variant, icon, text]) => {
		toastRef.current.hide();
		setTimeout(() => {
			setStatus({ variant: variant, icon: icon, text: text });
			toastRef.current.toast();
		}, 300);
	};

	const showError = (err) => {
		toastRef.current.hide();
		setTimeout(() => {
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
		}, 300);
	};

	const showNotFound = () => {
		return navigate('/admin/not-found');
	};

	const redirect = (url) => {
		return navigate(url);
	};

	// add the 'isFullscreen' class to the body.
	const updateIsFullScreen = (isFullScreen) => {
		let body = appRef.current.closest('body');
		if (isFullScreen) {
			body.classList.add('isFullscreen');
		} else {
			body.classList.remove('isFullscreen');
		}
	};

	let baseURL = window.location.origin;
	let api = new API(baseURL);

	useEffect(() => {
		const fetchConnectors = async () => {
			let [connectors, err] = await api.connectors.find();
			if (err != null) {
				showError(err);
				return;
			}
			setConnectors(connectors);
		};
		fetchConnectors();
	}, []);

	if (connectors == null) {
		return;
	}

	return (
		<AppContext.Provider
			value={{
				API: api,
				showStatus: showStatus,
				showError: showError,
				showNotFound: showNotFound,
				redirect: redirect,
				updateIsFullScreen: updateIsFullScreen,
				connectors: connectors,
			}}
		>
			<div className='App' ref={appRef}>
				<Outlet />
				<Toast toastRef={toastRef} status={status} />
			</div>
		</AppContext.Provider>
	);
};

export default App;
