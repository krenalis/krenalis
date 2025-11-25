import React, { FormEvent, useState, useContext, useEffect } from 'react';
import './Login.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import AppContext from '../../../context/AppContext';
import { Link } from '../../base/Link/Link';
import { useSearchParams } from 'react-router-dom';
import * as icons from '../../../constants/icons';
import { IS_DOCKER_KEY, IS_PASSWORDLESS_KEY } from '../../../constants/storage';

const Login = () => {
	const [email, setEmail] = useState<string>('');
	const [password, setPassword] = useState<string>('');
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [isTryingPasswordlessLogin, setIsTryingPasswordlessLogin] = useState<boolean>(true);

	const { api, handleError, showStatus, setIsLoadingState, setIsLoggedIn, setIsPasswordless, publicMetadata } =
		useContext(AppContext);

	const [searchParams, setSearchParams] = useSearchParams();

	useEffect(() => {
		const status = searchParams.get('status');
		if (status == null) {
			return;
		}
		showStatus({ variant: 'success', icon: icons.OK, text: 'You can now log in with your new password' });
		setSearchParams(new URLSearchParams()); // reset the search params.
	}, []);

	useEffect(() => {
		const tryPasswordlessLogin = async () => {
			let authError: string;
			try {
				[, authError] = await api.login('docker@meergo.com', 'meergo-password', true);
			} catch (err) {
				// Do nothing.
				setIsTryingPasswordlessLogin(false);
				return;
			}
			if (authError == null) {
				// Automatically login the user in passwordless mode.
				setIsLoggedIn(true);
				setIsLoadingState(true);
				localStorage.setItem(IS_PASSWORDLESS_KEY, '1');
				// Give the user the ability to have the warehouse based
				// on the PostgreSQL instance provided by Docker.
				localStorage.setItem(IS_DOCKER_KEY, '1');
				setIsPasswordless(true);
				setIsTryingPasswordlessLogin(false);
				return;
			} else {
				// Reset any previous Docker-based state if the
				// application was launched in a Docker environment but
				// is no longer running in that mode.
				localStorage.removeItem(IS_DOCKER_KEY);
			}
			try {
				[, authError] = await api.login('acme@meergo.com', 'meergo-password', true);
			} catch (err) {
				// Do nothing.
				setIsTryingPasswordlessLogin(false);
				return;
			}
			if (authError == null) {
				// Automatically login the user in passwordless mode.
				setIsLoggedIn(true);
				setIsLoadingState(true);
				localStorage.setItem(IS_PASSWORDLESS_KEY, '1');
				setIsPasswordless(true);
			}
			setIsTryingPasswordlessLogin(false);
		};

		tryPasswordlessLogin();
	}, []);

	const onLogin = async (e: FormEvent) => {
		e.preventDefault();
		setIsLoading(true);
		let authError: string;
		try {
			[, authError] = await api.login(email, password);
		} catch (err) {
			setIsLoading(false);
			handleError(err);
			return;
		}
		if (authError) {
			setIsLoading(false);
			if (authError === 'AuthenticationFailed') {
				handleError('Your email or password are incorrect');
				return;
			}
			return;
		}
		setIsLoggedIn(true);
		setIsLoading(false);
		setIsLoadingState(true);
	};

	const onEmailChange = (e) => {
		setEmail(e.currentTarget.value);
	};

	const onPasswordChange = (e) => {
		setPassword(e.currentTarget.value);
	};

	if (isTryingPasswordlessLogin) {
		return (
			<SlSpinner
				style={
					{
						display: 'block',
						position: 'relative',
						top: '50px',
						margin: 'auto',
						fontSize: '3rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			></SlSpinner>
		);
	}

	return (
		<div className='login'>
			<div className='login__container'>
				<div className='login__heading'>
					<h1>Sign-in to your account</h1>
				</div>
				<form className='login__form' onSubmit={onLogin}>
					{/* Using standard inputs instead of Shoelace inputs as a 
						workaround for Shoelace issue #269 */}
					<input
						type='email'
						className='login__email'
						inputMode='email'
						onInput={onEmailChange}
						name='email'
						value={email}
						placeholder='Your email'
						required
					/>
					<input
						type='password'
						className='login__password'
						onInput={onPasswordChange}
						name='password'
						value={password}
						placeholder='Your password'
						minLength={8}
						required
					/>

					{publicMetadata.canSendMemberPasswordReset && (
						<Link path='reset-password' className='login__reset-password'>
							Forgot your password?
						</Link>
					)}

					{/* This hidden submit input is needed to trigger the form 
						submission when the user presses Enter while focusing 
						one of the fields. The Shoelace button alone doesn’t 
						enable this behavior. */}
					<input type='submit' style={{ display: 'none' }} />

					<SlButton className='login__button' type='submit' variant='primary' loading={isLoading}>
						Login
					</SlButton>
				</form>
			</div>
		</div>
	);
};

export default Login;
