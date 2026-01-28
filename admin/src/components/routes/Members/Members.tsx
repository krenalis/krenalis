import React, { useState, useContext, useEffect, useRef } from 'react';
import './Members.css';
import AppContext from '../../../context/AppContext';
import ListTile from '../../base/ListTile/ListTile';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import { Member } from '../../../lib/api/types/responses';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlBadge from '@shoelace-style/shoelace/dist/react/badge/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { validateMemberEmail } from '../../../lib/core/member';
import { Link } from '../../base/Link/Link';

const Members = () => {
	const [isLoading, setIsLoading] = useState<boolean>(true);
	const [isRemoveAlertOpen, setIsRemoveAlertOpen] = useState<boolean>(false);
	const [isInviteMemberDialogOpen, setIsInviteMemberDialogOpen] = useState<boolean>(false);
	const [members, setMembers] = useState<Member[]>();

	const { api, handleError, member: loggedMember, logout, publicMetadata } = useContext(AppContext);

	const pendingDeletedMember = useRef<number>(0);

	useEffect(() => {
		const fetchData = async () => {
			let members: Member[];
			try {
				members = await api.members();
			} catch (err) {
				handleError(err);
				setTimeout(() => setIsLoading(false), 300);
				return;
			}
			setMembers(members);

			setTimeout(() => setIsLoading(false), 300);
		};

		if (!isLoading) {
			return;
		}

		fetchData();
	}, [isLoading]);

	const onDeleteMember = (id: number) => {
		pendingDeletedMember.current = id;
		setIsRemoveAlertOpen(true);
	};

	const onDeleteMemberCancel = () => {
		pendingDeletedMember.current = 0;
		setIsRemoveAlertOpen(false);
	};

	const onDeleteMemberConfirmation = async () => {
		try {
			await api.deleteMember(pendingDeletedMember.current);
		} catch (err) {
			if (!(err instanceof NotFoundError)) {
				handleError(err);
				return;
			}
		}
		setIsRemoveAlertOpen(false);
		setMembers(members.filter((member) => member.id !== pendingDeletedMember.current));
		pendingDeletedMember.current = 0;
	};

	const onLogout = async () => {
		await logout();
	};

	if (isLoading) {
		return (
			<div className='members'>
				<div className='members__content'>
					<SlSpinner
						style={
							{
								fontSize: '3rem',
								'--track-width': '6px',
							} as React.CSSProperties
						}
					></SlSpinner>
				</div>
			</div>
		);
	} else {
		return (
			<div className='members'>
				<div className='members__content'>
					<div className='members__title'>
						<p className='members__title-text'>Team members</p>
						{publicMetadata.inviteMembersViaEmail ? (
							<SlButton size='small' variant='primary' onClick={() => setIsInviteMemberDialogOpen(true)}>
								Invite a new team member
							</SlButton>
						) : (
							<Link path={'organization/members/add'}>
								<SlButton size='small' variant='primary' onClick={() => null}>
									Add a new team member
								</SlButton>
							</Link>
						)}
					</div>
					<div className='members__list'>
						{members.map((member) => {
							return (
								<ListTile
									key={member.id}
									className='members__member'
									name={
										<div className='members__member-name'>
											{member.name}
											{member.invitation !== '' && <SlBadge variant='neutral'>Invited</SlBadge>}
											{member.invitation === 'Expired' && (
												<SlBadge variant='danger'>Invitation expired</SlBadge>
											)}
										</div>
									}
									description={member.email}
									icon={
										<SlAvatar
											image={
												member.avatar
													? `data:${member.avatar.mimeType};base64, ${member.avatar.image}`
													: ''
											}
										/>
									}
									action={
										<div className='members__member-pipelines'>
											{member.email === loggedMember.email && (
												<SlButton
													className='members__member-logout'
													size='small'
													onClick={onLogout}
												>
													Logout
												</SlButton>
											)}
											{member.id === loggedMember.id && (
												<Link path={'organization/members/current'}>
													<SlButton className='members__member-edit' size='small'>
														Edit
													</SlButton>
												</Link>
											)}
											{members.length > 1 && (
												<SlButton
													size='small'
													variant='danger'
													onClick={() => onDeleteMember(member.id)}
												>
													Delete
												</SlButton>
											)}
										</div>
									}
								/>
							);
						})}
					</div>
				</div>
				<AlertDialog
					variant='danger'
					isOpen={isRemoveAlertOpen}
					onClose={onDeleteMemberCancel}
					title='Delete the team member?'
					actions={
						<>
							<SlButton onClick={onDeleteMemberCancel}>Cancel</SlButton>
							<SlButton variant='danger' onClick={onDeleteMemberConfirmation}>
								Delete
							</SlButton>
						</>
					}
				>
					If you delete the team member they will no longer have access to this account.
				</AlertDialog>
				<InviteMemberDialog
					isOpen={isInviteMemberDialogOpen}
					setIsOpen={setIsInviteMemberDialogOpen}
					setIsLoadingMembers={setIsLoading}
				/>
			</div>
		);
	}
};

interface InviteMemberDialogProps {
	isOpen: boolean;
	setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
	setIsLoadingMembers: React.Dispatch<React.SetStateAction<boolean>>;
}

const InviteMemberDialog = ({ isOpen, setIsOpen, setIsLoadingMembers }: InviteMemberDialogProps) => {
	const [email, setEmail] = useState<string>('');
	const [isSaving, setIsSaving] = useState<boolean>(false);
	const [error, setError] = useState<string>('');

	const { handleError, api } = useContext(AppContext);

	const inputRef = useRef<any>();

	useEffect(() => {
		if (isOpen) {
			setTimeout(() => {
				inputRef.current.focus();
			}, 100);
		}
	}, [isOpen]);

	const onUpdateEmail = (e) => {
		const value = e.target.value;
		setEmail(value);
	};

	const onInviteMember = async (e: any) => {
		e.preventDefault();
		setError('');
		setIsSaving(true);
		try {
			validateMemberEmail(email);
		} catch (err) {
			setTimeout(() => {
				setIsSaving(false);
				setError(err.message);
			}, 300);
			return;
		}
		try {
			await api.inviteMember(email);
		} catch (err) {
			if (err instanceof UnprocessableError && err.code !== 'CannotSendEmails') {
				setTimeout(() => {
					setIsSaving(false);
					setError(err.message);
				}, 300);
			} else {
				setTimeout(() => {
					setIsSaving(false);
					setIsOpen(false);
					setTimeout(() => {
						setEmail('');
						handleError(err);
					}, 150);
				}, 300);
			}
			return;
		}
		setTimeout(() => {
			setIsSaving(false);
			setIsOpen(false);
			setTimeout(() => {
				setEmail('');
				setIsLoadingMembers(true);
			}, 300);
		}, 300);
	};

	return (
		<SlDialog
			className='members__invite-dialog'
			label='Invite new team member'
			open={isOpen}
			onSlAfterHide={() => setIsOpen(false)}
		>
			<div className='members__invite-dialog-description'>
				An invitation to create a new team member account will be sent to the email address provided.
			</div>
			<form onSubmit={onInviteMember}>
				<SlInput ref={inputRef} label='Email' type='email' value={email} onSlInput={onUpdateEmail} required />
				{error && (
					<div className='members__invite-dialog-error'>
						<SlIcon slot='icon' name='exclamation-octagon' />
						{error}
					</div>
				)}
				<SlButton loading={isSaving} className='members__invite' type='submit' variant='primary'>
					Invite
				</SlButton>
			</form>
		</SlDialog>
	);
};

export default Members;
