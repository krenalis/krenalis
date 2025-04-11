import React, { FormEvent, useContext, useState } from 'react';
import './ResetPassword.css';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import appContext from '../../../context/AppContext';
import { Link } from '../../base/Link/Link';

const ResetPassword = () => {
	const [email, setEmail] = useState<string>('');
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [isEmailSent, setIsEmailSent] = useState<boolean>(false);

	const { api, handleError } = useContext(appContext);

	const onEmailChange = (e: any) => {
		setEmail(e.currentTarget.value);
	};

	const onReset = async (e: FormEvent) => {
		e.preventDefault();
		setIsLoading(true);
		try {
			await api.sendMemberPasswordReset(email);
		} catch (err) {
			setIsLoading(false);
			handleError(err);
			return;
		}
		setTimeout(() => {
			setIsLoading(false);
			setIsEmailSent(true);
		}, 500);
	};

	return (
		<div className='reset-password'>
			<div className='reset-password__container'>
				{isEmailSent ? (
					<>
						<div className='reset-password__email-sent'>
							<SlIcon className='reset-password__email-sent-icon' name='check2-circle' />
							<div className='reset-password__email-sent-title'>Request received</div>
							<div className='reset-password__email-sent-text'>
								If the provided email exists, we will send you a link to reset your password
							</div>
						</div>
						<Link path='' className='reset-password__email-sent-back-to-login'>
							Back to login
						</Link>
					</>
				) : (
					<>
						<div className='reset-password__heading'>
							<h1>Reset password</h1>
						</div>
						<form className='reset-password__form' onSubmit={onReset}>
							<SlInput
								type='email'
								className='reset-password__email'
								inputMode='email'
								onSlInput={onEmailChange}
								name='email'
								value={email}
								placeholder='Your email'
								required
							/>
							<Link path='' className='reset-password__back-to-login'>
								Back to login
							</Link>
							<SlButton
								className='reset-password__button'
								type='submit'
								variant='primary'
								loading={isLoading}
							>
								Send reset email
							</SlButton>
						</form>
					</>
				)}
			</div>
		</div>
	);
};

export { ResetPassword };
