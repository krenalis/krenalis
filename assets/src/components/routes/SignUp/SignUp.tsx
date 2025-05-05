import React, { useContext, useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import './SignUp.css';
import AppContext from '../../../context/AppContext';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { MemberInvitationResponse } from '../../../lib/api/types/responses';
import { validateMemberToSet } from '../../../lib/core/member';

const SignUp = () => {
	const [invitedEmail, setInvitedEmail] = useState<string>('');
	const [organizationName, setOrganizationName] = useState<string>('');
	const [name, setName] = useState<string>('');
	const [password, setPassword] = useState<string>('');
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [error, setError] = useState<string>('');

	const { token } = useParams();

	const { api, redirect, handleError, setIsLoggedIn, setIsLoadingState, logout } = useContext(AppContext);

	useEffect(() => {
		const logoutAndFetchInvitedMember = async () => {
			try {
				await api.logout();
			} catch (err) {
				handleError(err);
				return;
			}
			logout();
			if (token.length === 0) {
				handleError('Missing invitation token');
				redirect('');
				return;
			}
			let res: MemberInvitationResponse;
			try {
				res = await api.memberInvitation(token);
			} catch (err) {
				if (err instanceof NotFoundError) {
					handleError('This invitation does not exist anymore');
					redirect('');
					return;
				}
				if (err instanceof UnprocessableError) {
					if (err.code === 'InvitationTokenExpired') {
						handleError(
							'This invitation is expired, contact the organization administrator to ask for a new one',
						);
						redirect('');
						return;
					}
				}
				handleError(err);
				return;
			}
			setInvitedEmail(res.email);
			setOrganizationName(res.organization);
		};

		logoutAndFetchInvitedMember();
	}, []);

	const onNameChange = (e) => {
		const value = e.target.value;
		setName(value);
	};

	const onPasswordChange = (e) => {
		const value = e.target.value;
		setPassword(value);
	};

	const onSignUp = async (e: any) => {
		e.preventDefault();
		setError('');
		try {
			validateMemberToSet({ name: name, email: invitedEmail, password: password }, false, true);
		} catch (err) {
			setTimeout(() => {
				setError(err.message);
			}, 300);
			return;
		}
		setIsLoading(true);
		try {
			await api.acceptInvitation(token, name, password);
		} catch (err) {
			setIsLoading(false);
			if (err instanceof NotFoundError) {
				handleError('This invitation does not exist anymore');
				redirect('');
				return;
			}
			if (err instanceof UnprocessableError) {
				if (err.code === 'InvitationTokenExpired') {
					handleError(
						'This invitation is expired, contact the organization administrator to ask for a new one',
					);
					redirect('');
					return;
				}
			}
			handleError(err);
			return;
		}
		let authError: string;
		try {
			[, authError] = await api.login(invitedEmail, password);
		} catch (err) {
			setIsLoading(false);
			handleError(err);
			return;
		}
		if (authError) {
			setIsLoading(false);
			handleError(
				'It was not possible to log you in automatically. Please enter your email and password to log in.',
			);
			redirect('');
			return;
		}
		setTimeout(() => {
			setIsLoggedIn(true);
			setIsLoading(false);
			setIsLoadingState(true);
			redirect('connections');
		}, 300);
	};

	return (
		<div className='signup'>
			<div className='signup__logo'>Logo</div>
			<h1 className='signup__title'>Sign up to {organizationName}</h1>
			<form onSubmit={onSignUp}>
				<SlInput className='signup__email' label='Email' value={invitedEmail} disabled />
				<SlInput className='signup__name' label='Name' value={name} onSlInput={onNameChange} required />
				<SlInput
					type='password'
					className='signup__password'
					label='Password'
					value={password}
					onSlInput={onPasswordChange}
					passwordToggle
					required
				/>
				{error && (
					<div className='signup__error'>
						<SlIcon slot='icon' name='exclamation-octagon' />
						{error}
					</div>
				)}
				<SlButton className='signup__button' variant='primary' type='submit' loading={isLoading}>
					Sign up
				</SlButton>
			</form>
		</div>
	);
};

export default SignUp;
