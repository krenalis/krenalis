import { useState } from 'react';
import './Login.css';
import { SlButton, SlInput } from '@shoelace-style/shoelace/dist/react/index.js';

const Login = ({ setIsLoggedIn, api, showStatus, showError, setAccount }) => {
	const [email, setEmail] = useState('');
	const [password, setPassword] = useState('');
	const [isLoading, setIsLoading] = useState(false);

	const onLogin = async (e) => {
		e.preventDefault();
		setIsLoading(true);
		const [[accountID, authError], err] = await api.login(email, password);
		if (err !== null) {
			setIsLoading(false);
			showError(err);
			return;
		}
		if (authError) {
			setIsLoading(false);
			if (authError === 'AuthenticationFailed') {
				showStatus(['danger', 'lock', 'Your email or password are incorrect']);
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
						onSlChange={onEmailChange}
						name='email'
						value={email}
						placeholder='Your email'
						required
					/>
					<SlInput
						type='password'
						id='passwordInput'
						onSlChange={onPaswordChange}
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
