import { useState, useEffect, useRef, useLayoutEffect } from 'react';
import './App.css';
import Toast from './components/shared/Toast/Toast';
import Sidebar from './components/layout/Sidebar/Sidebar';
import Header from './components/layout/Header/Header';
import Login from './components/routes/Login/Login';
import API from './lib/api/api';
import * as variants from './constants/variants';
import * as icons from './constants/icons';
import { FULLSCREEN_PATTERNS } from './lib/helpers/navigation';
import { checkSessionCookie } from './lib/helpers/auth';
import { adminBasePath } from './constants/path';
import { BadRequestError } from './lib/api/errors';
import { AppProvider } from './context/providers/AppProvider';
import { Outlet } from 'react-router-dom';
import { useNavigate, useLocation, matchPath } from 'react-router-dom';
import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path.js';
import '@shoelace-style/shoelace/dist/themes/light.css';

setBasePath('https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.5.2/dist/');

const App = () => {
	const [isLoading, setIsLoading] = useState(true);
	const [isLoggedIn, setIsLoggedIn] = useState(false);
	const [isFullscreen, setIsFullscreen] = useState(false);
	const [status, setStatus] = useState(null);
	const [title, setTitle] = useState('');
	const [account, setAccount] = useState(null);

	const toastRef = useRef();
	const navigate = useNavigate();
	const location = useLocation();

	useLayoutEffect(() => {
		const hasSessionCookie = checkSessionCookie();
		if (hasSessionCookie) {
			setIsLoggedIn(true);
		} else {
			setIsLoggedIn(false);
		}
		setIsLoading(false);
	}, [location]);

	useEffect(() => {
		for (const pattern of FULLSCREEN_PATTERNS) {
			const match = matchPath(pattern, location.pathname);
			if (match != null) {
				setTimeout(() => setIsFullscreen(true), 200);
				return;
			}
		}
		setTimeout(() => setIsFullscreen(false), 200);
	}, [location]);

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
		return navigate(`${adminBasePath}not-found`);
	};

	const redirect = (url) => {
		toastRef.current.hide();
		return navigate(`${adminBasePath}${url}`);
	};

	const onLogout = () => {
		document.cookie = 'session=; Max-Age=-99999999; Path=/';
		setIsLoggedIn(false);
		redirect('');
	};

	if (isLoading) {
		return;
	}

	const baseURL = window.location.origin;
	const api = new API(baseURL);

	let content;
	if (isLoggedIn) {
		let isBasePath = location.pathname === adminBasePath;
		if (isBasePath) {
			redirect('connections');
			return;
		}
		content = (
			<AppProvider
				setTitle={setTitle}
				api={api}
				showError={showError}
				showStatus={showStatus}
				showNotFound={showNotFound}
				redirect={redirect}
				account={account}
			>
				<div className='app'>
					<Sidebar onLogout={onLogout} />
					<Header title={title} />
					<Outlet />
				</div>
			</AppProvider>
		);
	} else {
		content = (
			<Login
				setIsLoggedIn={setIsLoggedIn}
				api={api}
				showStatus={showStatus}
				showError={showError}
				setAccount={setAccount}
			/>
		);
	}

	return (
		<>
			{content}
			<div>
				<Toast ref={toastRef} status={status} isFullscreen={isFullscreen} />
			</div>
		</>
	);
};

export default App;
