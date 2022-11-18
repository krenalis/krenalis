import React from 'react';
import './Login.css';
import Alert from '../../components/Alert/Alert';
import call from '../../utils/call';
import { Navigate } from 'react-router-dom';
import { SlButton, SlIcon } from '@shoelace-style/shoelace/dist/react';

export default class Login extends React.Component {
	constructor(props) {
		super(props);
		this.state = {
			email: '',
			isLoggedIn: false,
			password: '',
			status: null,
		};
	}

	handleLogin = async () => {
		this.setState({ status: null });
		let [, err] = await call('/admin/', { email: this.state.email, password: this.state.password });
		if (err !== null) {
			if (err === 'AuthenticationFailedError') {
				this.setState({
					status: { variant: 'danger', icon: 'lock', text: 'Your email or password are incorrect' },
				});
				return;
			}
			this.setState({ status: { variant: 'danger', icon: 'exclamation-octagon', text: err } });
			return;
		}
		this.setState({ isLoggedIn: true });
	};

	onInputChange = (e) => {
		let name = e.currentTarget.name;
		let value = e.currentTarget.value;
		this.setState({ [name]: value });
	};

	render() {
		if (this.state.isLoggedIn) {
			return <Navigate to='account/connections-map' />;
		} else {
			return (
				<div className='Login'>
					<div className='container'>
						<div className='heading'>
							<h1>Sign-in to your account</h1>
						</div>
						{this.state.status && <Alert status={this.state.status} />}
						<form className='form' onSubmit={this.handleLogin}>
							<input
								type='text'
								onChange={this.onInputChange}
								name='email'
								value={this.state.text}
								placeholder='Your email'
							/>
							<input
								type='password'
								onChange={this.onInputChange}
								name='password'
								value={this.state.password}
								placeholder='Your password'
							/>
							<div className='note'>
								<span>Note:</span> sign in with email <span>acme@open2b.com</span> and password{' '}
								<span>foopass2</span>
							</div>
							<SlButton className='loginButton' variant='primary' onClick={this.handleLogin}>
								<SlIcon slot='suffix' name='box-arrow-in-right' />
								Login
							</SlButton>
						</form>
					</div>
				</div>
			);
		}
	}
}
