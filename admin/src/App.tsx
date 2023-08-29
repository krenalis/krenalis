import React, { useState, useEffect, useRef, ReactNode } from 'react';
import './App.css';
import Toast from './components/shared/Toast/Toast';
import Sidebar from './components/layout/Sidebar/Sidebar';
import Header from './components/layout/Header/Header';
import Login from './components/routes/Login/Login';
import API from './lib/api/api';
import * as variants from './constants/variants';
import * as icons from './constants/icons';
import { Status } from './types/internal/app';
import { FULLSCREEN_PATTERNS } from './lib/helpers/navigation';
import { checkSessionCookie } from './lib/helpers/auth';
import { adminBasePath } from './constants/path';
import { AppProvider } from './context/providers/AppProvider';
import { Outlet } from 'react-router-dom';
import { useNavigate, useLocation, matchPath } from 'react-router-dom';
import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path.js';
import SlAlert from '@shoelace-style/shoelace/dist/components/alert/alert';
import '@shoelace-style/shoelace/dist/themes/light.css';

setBasePath('https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.5.2/dist/');

const App = () => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isLoggedIn, setIsLoggedIn] = useState<boolean>(false);
	const [isFullscreen, setIsFullscreen] = useState<boolean>(false);
	const [status, setStatus] = useState<Status | null>(null);
	const [title, setTitle] = useState<ReactNode>('');
	const [account, setAccount] = useState<number | null>(null);

	const toastRef = useRef<SlAlert | null>(null);
	const navigate = useNavigate();
	const location = useLocation();

	useEffect(() => {
		const hasSessionCookie = checkSessionCookie();
		if (hasSessionCookie) {
			setIsLoggedIn(true);
		} else {
			setIsLoggedIn(false);
		}
		setIsLoading(false);
	}, [location]);

	useEffect(() => {
		if (isLoading) {
			return;
		}
		if (isLoggedIn) {
			let isBasePath = location.pathname === adminBasePath;
			if (isBasePath) {
				redirect('connections');
			}
		} else {
			if (location.pathname !== adminBasePath) {
				redirect('');
			}
		}
	}, [isLoggedIn]);

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

	const showStatus = (status: Status) => {
		if (toastRef.current == null) return;
		toastRef.current.hide();
		setTimeout(() => {
			setStatus(status);
			toastRef.current!.toast();
		}, 300);
	};

	const showError = (err: Error | string) => {
		if (toastRef.current == null) return;
		toastRef.current.hide();
		setTimeout(() => {
			setStatus({
				variant: variants.DANGER,
				icon: icons.EXCLAMATION,
				text: err instanceof Error ? err.message : err,
			});
			toastRef.current!.toast();
		}, 300);
	};

	const showNotFound = () => {
		return navigate(`${adminBasePath}not-found`);
	};

	const redirect = (url: string) => {
		if (toastRef.current) {
			toastRef.current.hide();
		}
		const redirectURL = `${adminBasePath}${url}`;
		if (redirectURL === location.pathname) {
			navigate(0);
			return;
		}
		return navigate(`${adminBasePath}${url}`);
	};

	const onLogout = () => {
		document.cookie = 'session=; Max-Age=-99999999; Path=/';
		setIsLoggedIn(false);
	};

	if (isLoading) {
		return null;
	}

	const baseURL = window.location.origin;
	const api = new API(baseURL);

	let content: ReactNode;
	if (isLoggedIn) {
		content = (
			<AppProvider
				api={api}
				showError={showError}
				showStatus={showStatus}
				showNotFound={showNotFound}
				redirect={redirect}
				setTitle={setTitle}
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
