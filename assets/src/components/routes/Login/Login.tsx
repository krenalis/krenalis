import React, { FormEvent, useState, useContext, useEffect } from 'react';
import './Login.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import AppContext from '../../../context/AppContext';
import { Link } from '../../base/Link/Link';
import { useSearchParams } from 'react-router-dom';
import * as icons from '../../../constants/icons';

const Login = () => {
	const [email, setEmail] = useState<string>('');
	const [password, setPassword] = useState<string>('');
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const { api, handleError, showStatus, setIsLoadingState, setIsLoggedIn, logout } = useContext(AppContext);

	const [searchParms, setSearchParams] = useSearchParams();

	useEffect(() => {
		const status = searchParms.get('status');
		if (status == null) {
			return;
		}
		showStatus({ variant: 'success', icon: icons.OK, text: 'You can now log in with your new password' });
		setSearchParams(new URLSearchParams()); // reset the search params.
	}, []);

	useEffect(() => {
		const removeCookieAndLogout = async () => {
			try {
				// remove the session cookie.
				await api.logout();
			} catch (err) {
				handleError(err);
				return;
			}
			// ensure user is logged out even when they navigate to this route
			// via the back/forward button of the browser.
			logout();
		};
		removeCookieAndLogout();
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

	const onPaswordChange = (e) => {
		setPassword(e.currentTarget.value);
	};

	return (
		<div className='login'>
			<div className='login__container'>
				<div className='login__heading'>
					<h1>Sign-in to your account</h1>
				</div>
				<form className='login__form' onSubmit={onLogin}>
					{/* Using standard inputs instead of Shoelace inputs as a workaround for Shoelace issue #269 */}
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
						onInput={onPaswordChange}
						name='password'
						value={password}
						placeholder='Your password'
						minLength={8}
						required
					/>
					<Link path='reset-password' className='login__reset-password'>
						Forgot your password?
					</Link>
					<SlButton className='login__button' type='submit' variant='primary' loading={isLoading}>
						Login
					</SlButton>
				</form>
			</div>
		</div>
	);
};

export default Login;
