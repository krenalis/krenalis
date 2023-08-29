import React, { FormEvent, useState } from 'react';
import './Login.css';
import { SlButton, SlInput } from '@shoelace-style/shoelace/dist/react/index.js';
import API from '../../../lib/api/api';
import { Status } from '../../../types/internal/app';

interface LoginProps {
	setIsLoggedIn: React.Dispatch<React.SetStateAction<boolean>>;
	api: API;
	showStatus: (status: Status) => void;
	showError: (err: Error | string) => void;
	setAccount: React.Dispatch<React.SetStateAction<number | null>>;
}

const Login = ({ setIsLoggedIn, api, showStatus, showError, setAccount }: LoginProps) => {
	const [email, setEmail] = useState<string>('');
	const [password, setPassword] = useState<string>('');
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const onLogin = async (e: FormEvent) => {
		e.preventDefault();
		setIsLoading(true);
		let accountID: number, authError: string;
		try {
			[accountID, authError] = await api.login(email, password);
		} catch (err) {
			setIsLoading(false);
			showError(err);
			return;
		}
		if (authError) {
			setIsLoading(false);
			if (authError === 'AuthenticationFailed') {
				showStatus({ variant: 'danger', icon: 'lock', text: 'Your email or password are incorrect' });
				return;
			}
		}
		setIsLoading(false);
		setAccount(accountID);
		setIsLoggedIn(true);
	};

	const onEmailChange = (e) => {
		setEmail(e.currentTarget.value);
	};

	const onPaswordChange = (e) => {
		setPassword(e.currentTarget.value);
	};

	return (
		<div className='login'>
			<div className='container'>
				<div className='heading'>
					<h1>Sign-in to your account</h1>
				</div>
				<form className='loginForm' onSubmit={onLogin}>
					<SlInput
						type='email'
						id='emailInput'
						inputMode='email'
						onSlInput={onEmailChange}
						name='email'
						value={email}
						placeholder='Your email'
						required
					/>
					<SlInput
						type='password'
						id='passwordInput'
						onSlInput={onPaswordChange}
						name='password'
						value={password}
						placeholder='Your password'
						minlength={8}
						passwordToggle={true}
						required
					/>
					<div className='note'>
						<span>Note:</span> sign in with email <span>acme@open2b.com</span> and password{' '}
						<span>foopass2</span>
					</div>
					<SlButton className='loginButton' type='submit' variant='primary' loading={isLoading}>
						Login
					</SlButton>
				</form>
			</div>
		</div>
	);
};

export default Login;
