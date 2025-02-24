import React, { useState, useEffect, useRef, ReactNode } from 'react';
import './App.css';
import Toast from '../../base/Toast/Toast';
import * as icons from '../../../constants/icons';
import { Status } from './App.types';
import { FULLSCREEN_PATHS } from '../../../constants/paths';
import { UI_BASE_PATH, SIGN_UP_PATH } from '../../../constants/paths';
import AppContext from '../../../context/AppContext';
import { Outlet } from 'react-router-dom';
import { useNavigate, useLocation, matchPath } from 'react-router-dom';
import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path.js';
import SlAlert from '@shoelace-style/shoelace/dist/components/alert/alert';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import '@shoelace-style/shoelace/dist/themes/light.css';
import { useApp } from './useApp';
import { UnauthorizedError } from '../../../lib/api/errors';

setBasePath('https://cdn.jsdelivr.net/npm/@shoelace-style/shoelace@2.19.0/dist/');

const App = () => {
	const [isFullscreen, setIsFullscreen] = useState<boolean>(false);
	const [status, setStatus] = useState<Status | null>(null);
	const [title, setTitle] = useState<ReactNode>('');
	const [isLoggedIn, setIsLoggedIn] = useState<boolean>(true);

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

	const logout = () => {
		setSelectedWorkspace(0);
		setIsLoggedIn(false);
	};

	const handleError = (err: Error | string) => {
		if (err instanceof UnauthorizedError) {
			logout();
			return;
		}
		if (toastRef.current == null) return;
		toastRef.current.hide();
		setTimeout(() => {
			setStatus({
				variant: 'danger',
				icon: icons.EXCLAMATION,
				text: err instanceof Error ? err.message : err,
			});
			toastRef.current!.toast();
		}, 300);
	};

	const showNotFound = () => {
		return navigate(`${UI_BASE_PATH}not-found`);
	};

	const redirect = (url: string) => {
		if (toastRef.current) {
			toastRef.current.hide();
		}
		const redirectURL = `${UI_BASE_PATH}${url}`;
		if (redirectURL === location.pathname) {
			setIsLoadingState(true);
			return;
		}
		return navigate(`${UI_BASE_PATH}${url}`);
	};

	const {
		isLoadingState,
		setIsLoadingState,
		member,
		setIsLoadingMember,
		connectors,
		connections,
		setIsLoadingConnections,
		warehouse,
		workspaces,
		setIsLoadingWorkspaces,
		selectedWorkspace,
		setSelectedWorkspace,
		api,
		executeAction,
		executeActionButtonRefs,
	} = useApp(handleError, redirect, logout, location);

	useEffect(() => {
		if (!isLoggedIn && location.pathname !== UI_BASE_PATH && !location.pathname.startsWith(SIGN_UP_PATH)) {
			redirect('');
		}
	}, [isLoggedIn, location]);

	useEffect(() => {
		// Determine whether the current route spans the entire viewport or
		// includes a sidebar, and set the `isFullscreen` state variable to
		// ensure proper centering of fixed elements.
		if (isLoadingState) {
			setIsFullscreen(true);
			return;
		}
		for (const path of FULLSCREEN_PATHS) {
			const match = matchPath(path, location.pathname);
			if (match != null) {
				setTimeout(() => setIsFullscreen(true), 200);
				return;
			}
		}
		setTimeout(() => setIsFullscreen(false), 200);
	}, [location, isLoadingState]);

	let content: ReactNode;
	if (
		isLoadingState ||
		(!isLoggedIn && location.pathname !== UI_BASE_PATH && !location.pathname.startsWith(SIGN_UP_PATH))
	) {
		content = (
			<SlSpinner
				className='app-spinner'
				style={
					{
						fontSize: '5rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			/>
		);
	} else {
		content = (
			<AppContext.Provider
				value={{
					api,
					handleError,
					showStatus,
					showNotFound,
					redirect,
					setTitle,
					member,
					setIsLoadingMember,
					workspaces,
					setIsLoadingWorkspaces,
					warehouse,
					selectedWorkspace,
					setSelectedWorkspace,
					connectors,
					connections,
					setIsLoadingConnections,
					setIsLoadingState,
					isFullscreen,
					title,
					logout,
					setIsLoggedIn,
					toastRef,
					executeAction,
					executeActionButtonRefs,
				}}
			>
				<Outlet />
			</AppContext.Provider>
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
