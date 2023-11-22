import React, { useState, useEffect, useRef, ReactNode } from 'react';
import './App.css';
import Toast from './components/shared/Toast/Toast';
import Sidebar from './components/layout/Sidebar/Sidebar';
import Header from './components/layout/Header/Header';
import Login from './components/routes/Login/Login';
import Workspaces from './components/routes/Workspaces/Workspaces';
import * as variants from './constants/variants';
import * as icons from './constants/icons';
import { Status } from './types/internal/app';
import { FULLSCREEN_PATTERNS } from './lib/helpers/navigation';
import { adminBasePath } from './constants/path';
import AppContext from './context/AppContext';
import { Outlet } from 'react-router-dom';
import { useNavigate, useLocation, matchPath } from 'react-router-dom';
import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path.js';
import SlAlert from '@shoelace-style/shoelace/dist/components/alert/alert';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import '@shoelace-style/shoelace/dist/themes/light.css';
import { useApp } from './hooks/useApp';

setBasePath('https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.9.0/dist/');

const App = () => {
	const [account, setAccount] = useState<number | null>(null);
	const [isFullscreen, setIsFullscreen] = useState<boolean>(false);
	const [status, setStatus] = useState<Status | null>(null);
	const [title, setTitle] = useState<ReactNode>('');

	const toastRef = useRef<SlAlert | null>(null);
	const navigate = useNavigate();
	const location = useLocation();

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
			setIsLoadingState(true);
			return;
		}
		return navigate(`${adminBasePath}${url}`);
	};

	const {
		isLoadingState,
		setIsLoadingState,
		isLoggedIn,
		setIsLoggedIn,
		connectors,
		connections,
		setIsLoadingConnections,
		warehouse,
		workspaces,
		setIsLoadingWorkspaces,
		selectedWorkspace,
		setSelectedWorkspace,
		api,
	} = useApp(showError, redirect, location);

	useEffect(() => {
		// Determine whether the current route spans the entire viewport or
		// includes a sidebar, and set the `isFullscreen` state variable to
		// ensure proper centering of fixed elements.
		if (isLoadingState) {
			setIsFullscreen(true);
			return;
		}
		for (const pattern of FULLSCREEN_PATTERNS) {
			const match = matchPath(pattern, location.pathname);
			if (match != null) {
				setTimeout(() => setIsFullscreen(true), 200);
				return;
			}
		}
		setTimeout(() => setIsFullscreen(false), 200);
	}, [location, isLoadingState]);

	let content: ReactNode;
	if (isLoadingState) {
		content = (
			<SlSpinner
				className='globalSpinner'
				style={
					{
						fontSize: '5rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			/>
		);
	} else if (isLoggedIn) {
		if (selectedWorkspace === 0) {
			content = (
				<Workspaces
					setSelectedWorkspace={setSelectedWorkspace}
					workspaces={workspaces}
					api={api}
					showError={showError}
					redirect={redirect}
					setIsLoadingState={setIsLoadingState}
				/>
			);
		} else {
			content = (
				<AppContext.Provider
					value={{
						api,
						showError,
						showStatus,
						showNotFound,
						redirect,
						setTitle,
						account,
						workspaces,
						setIsLoadingWorkspaces,
						warehouse,
						selectedWorkspace,
						setSelectedWorkspace,
						connectors,
						connections,
						setIsLoadingConnections,
						setIsLoadingState,
					}}
				>
					<div className='app'>
						<Sidebar
							setIsLoggedIn={setIsLoggedIn}
							workspaces={workspaces}
							warehouse={warehouse}
							selectedWorkspace={selectedWorkspace}
							setSelectedWorkspace={setSelectedWorkspace}
						/>
						<Header title={title} />
						<Outlet />
					</div>
				</AppContext.Provider>
			);
		}
	} else {
		content = (
			<Login
				setIsLoggedIn={setIsLoggedIn}
				setIsLoadingState={setIsLoadingState}
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
