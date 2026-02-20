import React, { useContext, useState, useEffect, useRef, useMemo, useLayoutEffect } from 'react';
import './Member.css';
import appContext from '../../../context/AppContext';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import * as icons from '../../../constants/icons';
import { MemberAvatar, MemberToSet } from '../../../lib/api/types/responses';
import { toBase64 } from '../../../utils/toBase64';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { validateMemberToSet } from '../../../lib/core/member';
import { Link } from '../../base/Link/Link';
import { useLocation } from 'react-router-dom';
import { IS_PASSWORDLESS_KEY } from '../../../constants/storage';

const Member = () => {
	const [avatar, setAvatar] = useState<MemberAvatar | null>(null);
	const [name, setName] = useState<string>('');
	const [email, setEmail] = useState<string>('');
	const [password, setPassword] = useState<string | null>(null);
	const [password2, setPassword2] = useState<string | null>(null);
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [isSaving, setIsSaving] = useState<boolean>(false);
	const [error, setError] = useState<string>('');

	const location = useLocation();

	const {
		api,
		member,
		handleError,
		showStatus,
		setIsLoadingMember,
		setTitle,
		redirect,
		isPasswordless,
		setIsPasswordless,
		publicMetadata,
	} = useContext(appContext);

	const nameInputRef = useRef<any>();
	const fileInputRef = useRef<any>();
	const passwordInputRef = useRef<any>();

	const isUpdate = useMemo(() => {
		return location.pathname.endsWith('current');
	}, [location]);

	useLayoutEffect(() => {
		if (!isUpdate) {
			setIsLoading(true);
		}
	}, []);

	useEffect(() => {
		if (isUpdate) {
			// Editing the currently logged-in member. This flow updates the
			// member's own profile data.
			setTitle(member.name);
			setAvatar(member.avatar);
			setName(member.name);
			setEmail(member.email);
		} else {
			// Adding a new member directly, without email invitation.
			if (publicMetadata.inviteMembersViaEmail) {
				// If email invitation is required, this flow is not allowed.
				// Redirect to members list.
				handleError('Email invitation is required');
				redirect('organization/members');
				return;
			}
			setTimeout(() => {
				setIsLoading(false);
				setTimeout(() => {
					nameInputRef.current?.focus();
				}, 100);
			}, 300);
		}
	}, [isUpdate]);

	const onUpdateAvatar = async (e) => {
		setError('');
		const f: File = Array.from(e.target.files)[0] as File;
		if (f == null) {
			return;
		}
		if (f.type !== 'image/jpeg' && f.type !== 'image/png') {
			e.target.value = '';
			setTimeout(() => {
				setError('image must be in jpeg or png format');
			}, 300);
			return;
		}
		if (f.size > 200 * 1024) {
			e.target.value = '';
			setTimeout(() => {
				setError('image must be smaller than 200KB');
			}, 300);
			return;
		}
		const base64: string = await toBase64(f);
		setAvatar({ image: base64, mimeType: f.type });
	};

	const onDeleteAvatar = (e) => {
		e.preventDefault();
		e.stopPropagation();
		setError('');
		fileInputRef.current.value = '';
		setAvatar(null);
	};

	const onUpdateName = (e) => {
		const value = e.target.value;
		setName(value);
	};

	const onUpdateEmail = (e) => {
		const value = e.target.value;
		setEmail(value);
	};

	const onPasswordEnable = () => {
		setPassword('');
		setTimeout(() => {
			passwordInputRef.current?.focus();
		}, 100);
	};

	const onUpdatePassword = (e) => {
		const value = e.target.value;
		setPassword(value);
	};

	const onUpdatePassword2 = (e) => {
		const value = e.target.value;
		setPassword2(value);
	};

	const onSave = async (e: any) => {
		e.preventDefault();
		setError('');
		setIsSaving(true);
		const memberToSet: MemberToSet = {
			name: name,
			email: email,
			image: avatar ? avatar.image : null,
		};
		if (password != null) {
			memberToSet.password = password;
		}
		try {
			validateMemberToSet(memberToSet, true, password != null ? true : false, isUpdate ? null : password2);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setError(err.message);
			}, 300);
			return;
		}
		try {
			if (isUpdate) {
				await api.updateMember(memberToSet);
			} else {
				await api.addMember(memberToSet);
			}
		} catch (err) {
			if (err instanceof UnprocessableError) {
				// Note that EmailInvitationRequired can only be returned when
				// calling 'api.addMember', not 'api.updateMember'.
				if (err.code === 'EmailInvitationRequired') {
					handleError('Email invitation is required');
					redirect('organization/members');
					return;
				}
				setTimeout(() => {
					setIsSaving(false);
					setError(err.message);
				}, 300);
			} else if (err instanceof NotFoundError) {
				setTimeout(() => {
					setIsLoadingMember(true);
				}, 300);
			} else {
				setTimeout(() => {
					setIsSaving(false);
					handleError(err);
				}, 300);
			}
			return;
		}
		if (password != null && isPasswordless) {
			// The user has updated their password, so they are no
			// longer in passwordless mode.
			localStorage.removeItem(IS_PASSWORDLESS_KEY);
			setIsPasswordless(false);
		}
		setTimeout(() => {
			setIsSaving(false);
			setIsLoadingMember(true);
			showStatus({ variant: 'success', icon: icons.OK, text: 'Team member information saved successfully' });
			redirect('organization/members');
		}, 300);
	};

	return (
		<div className='member'>
			<div className={`member__content${isUpdate ? ' member__content--update' : ''}`}>
				{isLoading ? (
					<SlSpinner
						style={
							{
								fontSize: '3rem',
								'--track-width': '6px',
							} as React.CSSProperties
						}
					/>
				) : (
					<>
						<div className='member__title'>{isUpdate ? 'Your profile' : 'Add a new team member'}</div>
						<form onSubmit={onSave}>
							<div className='member__name'>
								<SlInput
									ref={nameInputRef}
									label='Name'
									name='name'
									value={name}
									onSlInput={onUpdateName}
									required
								/>
							</div>
							<div className='member__email'>
								<SlInput
									label='Email'
									type='email'
									name='email'
									value={email}
									onSlInput={onUpdateEmail}
									required
								/>
							</div>
							<div className='member__password'>
								<SlInput
									ref={passwordInputRef}
									type='password'
									label='Password'
									name='password'
									disabled={isUpdate && password === null}
									required={!isUpdate || password !== null}
									onSlInput={onUpdatePassword}
									value={isUpdate && password === null ? '••••••••••••••••' : password}
									password-toggle
								/>
								{isUpdate && password === null && (
									<SlButton onClick={onPasswordEnable}>Change</SlButton>
								)}
							</div>
							{!isUpdate && (
								<div className='member__confirm-password'>
									<SlInput
										type='password'
										label='Confirm password'
										name='confirm-password'
										required={true}
										onSlInput={onUpdatePassword2}
										value={password2}
										password-toggle
									/>
								</div>
							)}
							<div className='member__avatar'>
								<div className='member__avatar-label'>Avatar</div>
								<div className='member__avatar-box'>
									<div className='member__avatar-buttons'>
										<div
											className='member__add-avatar'
											onClick={() => fileInputRef.current?.click()}
										>
											Upload
										</div>
										{avatar && (
											<div className='member__remove-avatar' onClick={onDeleteAvatar}>
												Delete
											</div>
										)}
									</div>
									<SlAvatar image={avatar ? `data:${avatar.mimeType};base64, ${avatar.image}` : ''} />
									<input
										ref={fileInputRef}
										type='file'
										accept='image/jpeg, image/png'
										onChange={onUpdateAvatar}
									/>
								</div>
							</div>
							{error && (
								<div className='member__error'>
									<SlIcon slot='icon' name='exclamation-octagon' />
									{error}
								</div>
							)}
							<div className='member__buttons'>
								<Link path='organization/members'>
									<SlButton className='member__cancel-button'>Cancel</SlButton>
								</Link>
								<SlButton
									className='member__save-button'
									variant='primary'
									loading={isSaving}
									type='submit'
								>
									{isUpdate ? 'Save' : 'Add'}
								</SlButton>
							</div>
						</form>
					</>
				)}
			</div>
		</div>
	);
};

export default Member;
