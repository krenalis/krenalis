import React, { FormEvent, useContext, useEffect, useState } from 'react';
import './ResetPasswordToken.css';
import appContext from '../../../context/AppContext';
import { useParams } from 'react-router-dom';
import { NotFoundError } from '../../../lib/api/errors';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { validateMemberPassword } from '../../../lib/core/member';
import { debounce } from '../../../utils/debounce';

const ResetPasswordToken = () => {
	const [password, setPassword] = useState<string>('');
	const [password2, setPassword2] = useState<string>('');
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [passwordError, setPasswordError] = useState<string>('');

	const { api, handleError, redirect, logout } = useContext(appContext);

	const { token } = useParams();

	useEffect(() => {
		const logoutAndFetchResetToken = async () => {
			try {
				await api.logout();
			} catch (err) {
				handleError(err);
				return;
			}
			logout();
			if (token.length === 0) {
				handleError('Missing reset password token');
				redirect('');
				return;
			}
			try {
				await api.validateMemberPasswordResetToken(token);
			} catch (err) {
				if (err instanceof NotFoundError) {
					handleError('This reset password request is expired');
					redirect('');
					return;
				}
				handleError(err);
				return;
			}
		};
		logoutAndFetchResetToken();
	}, []);

	const onPasswordInput = (e: any) => {
		setPasswordError('');
		const v = e.target.value;
		setPassword(v);
		try {
			validatePassword(v, password2, false);
		} catch (err) {
			setPasswordError(err.message);
		}
	};

	const onPassword2Input = (e: any) => {
		setPasswordError('');
		const v = e.target.value;
		setPassword2(v);
		try {
			validatePassword(password, v, false);
		} catch (err) {
			setPasswordError(err.message);
		}
	};

	const onChangePassword = async (e: FormEvent) => {
		e.preventDefault();
		setIsLoading(true);
		setPasswordError('');
		try {
			validatePassword(password, password2, true);
		} catch (err) {
			setTimeout(() => {
				setPasswordError(err.message);
				setIsLoading(false);
			}, 300);
			return;
		}
		try {
			await api.changeMemberPasswordByToken(token, password);
		} catch (err) {
			setIsLoading(false);
			if (err instanceof NotFoundError) {
				handleError('This reset password request is expired');
				redirect('');
				return;
			}
			handleError(err);
			return;
		}
		setTimeout(() => {
			setIsLoading(false);
			redirect('?status=password-reset');
		}, 500);
	};

	return (
		<div className='reset-password-token'>
			<div className='reset-password-token__container'>
				<div className='reset-password-token__heading'>
					<h1>Reset password</h1>
				</div>
				<form className='reset-password-token__form' onSubmit={passwordError !== '' ? null : onChangePassword}>
					<SlInput
						type='password'
						className='reset-password-token__password'
						onSlInput={debounce(onPasswordInput, 500)}
						name='password'
						value={password}
						placeholder='Password'
						required
						password-toggle
					/>
					<SlInput
						type='password'
						className='reset-password-token__password-2'
						onSlInput={debounce(onPassword2Input, 500)}
						name='password-2'
						value={password2}
						placeholder='Confirm password'
						password-toggle
					/>
					{passwordError !== '' && (
						<div className='reset-password__password-error'>
							<SlIcon name='exclamation-circle' />
							<span className='reset-password__password-error-text'>{passwordError}</span>
						</div>
					)}
					<SlButton
						className='reset-password-token__button'
						type='submit'
						variant='primary'
						loading={isLoading}
						disabled={passwordError !== ''}
					>
						Reset password
					</SlButton>
				</form>
			</div>
		</div>
	);
};

const validatePassword = (password: string, password2: string, forcePasswordMatch: boolean) => {
	try {
		validateMemberPassword(password);
	} catch (err) {
		throw err;
	}
	if (password2 !== '' || forcePasswordMatch) {
		if (password !== password2) {
			throw new Error('Passwords must match');
		}
	}
};

export { ResetPasswordToken };
