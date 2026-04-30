import React, { useState, useEffect, useRef, ReactNode } from 'react';
import './App.css';
import Toast from '../../base/Toast/Toast';
import * as icons from '../../../constants/icons';
import { Status } from './App.types';
import { FULLSCREEN_PATHS, RESET_PASSWORD_PATH } from '../../../constants/paths';
import { UI_BASE_PATH, SIGN_UP_PATH } from '../../../constants/paths';
import AppContext from '../../../context/AppContext';
import { Outlet } from 'react-router-dom';
import { useNavigate, useLocation, matchPath } from 'react-router-dom';
import { setBasePath } from '@shoelace-style/shoelace/dist/utilities/base-path.js';
import type SlAlert from '@shoelace-style/shoelace/dist/components/alert/alert.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import '@shoelace-style/shoelace/dist/themes/light.css';
import { useApp } from './useApp';
import { UnauthorizedError } from '../../../lib/api/errors';
import * as Sentry from '@sentry/react';
import RootError from '../RootError/RootError';
import { IS_PASSWORDLESS_KEY, IS_DOCKER_KEY } from '../../../constants/storage';
import { AuthKitProvider, useAuth } from '@workos-inc/authkit-react';
import API from '../../../lib/api/api';
import { PublicMetadata } from '../../../lib/api/types/responses';
import '@radix-ui/themes/styles.css';
import '@workos-inc/widgets/styles.css';

setBasePath('/admin/src/shoelace/dist');

const App = ({ onWorkosLogout }: { onWorkosLogout?: () => void } = {}) => {
	const [isFullscreen, setIsFullscreen] = useState<boolean>(false);
	const [status, setStatus] = useState<Status | null>(null);
	const [title, setTitle] = useState<ReactNode>('');
	const [isLoggedIn, setIsLoggedIn] = useState<boolean>(false);

	const toastRef = useRef<SlAlert | null>(null);
	const navigate = useNavigate();
	const location = useLocation();

	const hasWorkOS = onWorkosLogout != null;

	const showStatus = (status: Status) => {
		if (toastRef.current == null) return;
		toastRef.current.hide();
		setTimeout(() => {
			setStatus(status);
			toastRef.current!.toast();
		}, 300);
	};

	const logout = async () => {
		try {
			// remove the session cookie.
			await api.logout();
		} catch (err) {
			handleError(err);
			return;
		}
		localStorage.removeItem(IS_PASSWORDLESS_KEY);
		setIsPasswordless(false);
		if (hasWorkOS) {
			onWorkosLogout();
		}
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
		runPipeline,
		runPipelineButtonRefs,
		runPipelineDropdownButtonRefs,
		isPasswordless,
		setIsPasswordless,
		publicMetadata,
	} = useApp(handleError, redirect, logout, location, setIsLoggedIn);

	useEffect(() => {
		if (!isLoadingState && !isLoggedIn && !isAuthRelatedRoute(location.pathname) && !hasWorkOS) {
			// if the app is initialized but the user is not logged in and they
			// try to access a non-authentication page, redirect them to the
			// login form first.
			//
			// In WorkOS mode the redirect is handled by WorkOSWrapper, so skip
			// it.
			redirect('');
		}
	}, [isLoadingState, isLoggedIn, location]);

	useEffect(() => {
		// Determine whether the current route spans the entire viewport or
		// includes a sidebar. This helps ensure centered positioning of the
		// fixed elements and to control the visibility of specific UI elements,
		// such as the automatically opened tooltip when the user is in
		// passwordless mode.
		for (const path of FULLSCREEN_PATHS) {
			const match = matchPath(path, location.pathname);
			if (match != null) {
				setIsFullscreen(true);
				return;
			}
		}
		setIsFullscreen(false);
	}, [location, isLoadingState]);

	let content: ReactNode;
	if (isLoadingState || (!isLoggedIn && (hasWorkOS || !isAuthRelatedRoute(location.pathname)))) {
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
					runPipeline,
					runPipelineButtonRefs,
					runPipelineDropdownButtonRefs,
					isPasswordless,
					setIsPasswordless,
					publicMetadata,
				}}
			>
				<Outlet />
			</AppContext.Provider>
		);
	}

	return (
		<Sentry.ErrorBoundary fallback={<RootError />}>
			{content}
			<div>
				<Toast ref={toastRef} status={status} isFullscreen={isFullscreen} />
			</div>
		</Sentry.ErrorBoundary>
	);
};

// WorkOSWrapper handles the WorkOS authentication flow. Once the WorkOS user is
// available it exchanges the WorkOS access token for a Krenalis session cookie,
// then renders the app as usual.
const WorkOSWrapper = () => {
	const [isLoggedInViaWorkos, setIsLoggedInViaWorkos] = useState(false);

	const { isLoading, user: workosUser, signIn, signOut, getAccessToken } = useAuth();

	useEffect(() => {
		if (!isLoading && workosUser == null) {
			signIn();
		}
	}, [isLoading, workosUser]);

	useEffect(() => {
		const loginViaWorkos = async () => {
			const api = new API(window.location.origin, 0);
			try {
				const token = await getAccessToken();
				await api.workosLogin(token);
			} catch (err) {
				signOut();
				return;
			}
			localStorage.removeItem(IS_PASSWORDLESS_KEY);
			localStorage.removeItem(IS_DOCKER_KEY);
			setIsLoggedInViaWorkos(true);
		};
		if (workosUser == null || isLoggedInViaWorkos) {
			return;
		}
		loginViaWorkos();
	}, [workosUser]);

	if (!isLoggedInViaWorkos) {
		return (
			<SlSpinner
				className='app-spinner'
				style={{ fontSize: '5rem', '--track-width': '6px' } as React.CSSProperties}
			/>
		);
	}
	return <App onWorkosLogout={signOut} />;
};

const Root = () => {
	const [workosClientID, setWorkosClientID] = useState<string | null>(null);

	useEffect(() => {
		const api = new API(window.location.origin, 0);

		const fetchWorkosClientID = async () => {
			let publicMetadata: PublicMetadata;
			try {
				publicMetadata = await api.publicMetadata();
			} catch (err) {
				console.error('error', err);
				return;
			}
			setWorkosClientID(publicMetadata.workosClientID);
		};

		fetchWorkosClientID();
	}, []);

	if (workosClientID == null) {
		return (
			<SlSpinner
				className='app-spinner'
				style={{ fontSize: '5rem', '--track-width': '6px' } as React.CSSProperties}
			/>
		);
	}

	const hasWorkOS = workosClientID !== '';
	if (hasWorkOS) {
		return (
			<AuthKitProvider clientId={workosClientID} redirectUri={`${window.location.origin}/admin`}>
				<WorkOSWrapper />
			</AuthKitProvider>
		);
	}

	return <App />;
};

const isAuthRelatedRoute = (path: string): boolean => {
	return path === UI_BASE_PATH || path.startsWith(SIGN_UP_PATH) || path.startsWith(RESET_PASSWORD_PATH);
};

export default Root;
