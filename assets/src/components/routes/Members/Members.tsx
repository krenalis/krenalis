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
import { TransformedMember, transformMember, validateMemberEmail } from '../../../lib/core/member';
import { Link } from '../../base/Link/Link';

const Members = () => {
	const [isLoadingMembers, setIsLoadingMembers] = useState<boolean>(true);
	const [isRemoveAlertOpen, setIsRemoveAlertOpen] = useState<boolean>(false);
	const [isInviteMemberDialogOpen, setIsInviteMemberDialogOpen] = useState<boolean>(false);
	const [members, setMembers] = useState<TransformedMember[]>();

	const { api, handleError, member: loggedMember } = useContext(AppContext);

	const pendingDeletedMember = useRef<number>(0);

	useEffect(() => {
		const fetchMembers = async () => {
			let members: Member[];
			try {
				members = await api.members();
			} catch (err) {
				handleError(err);
				setTimeout(() => setIsLoadingMembers(false), 300);
				return;
			}
			const transformed: TransformedMember[] = [];
			for (const m of members) {
				transformed.push(transformMember(m));
			}
			setMembers(transformed);
			setTimeout(() => setIsLoadingMembers(false), 300);
		};

		if (!isLoadingMembers) {
			return;
		}

		fetchMembers();
	}, [isLoadingMembers]);

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
		setIsLoadingMembers(true);
	};

	if (isLoadingMembers) {
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
					<Link path='organization'>
						<SlButton className='members__back-button' variant='text'>
							<SlIcon slot='prefix' name='arrow-left' />
							Organization
						</SlButton>
					</Link>
					<div className='members__title'>
						<p className='members__title-text'>Members</p>
						<SlButton size='small' variant='primary' onClick={() => setIsInviteMemberDialogOpen(true)}>
							Invite a new member
						</SlButton>
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
											{member.email === loggedMember.email && (
												<SlBadge variant='neutral'>You</SlBadge>
											)}
											{member.invitation !== '' && <SlBadge variant='neutral'>Invited</SlBadge>}
											{member.invitation === 'Expired' && (
												<SlBadge variant='danger'>Invitation expired</SlBadge>
											)}
										</div>
									}
									description={member.email}
									icon={
										<SlAvatar
											initials={member.initials}
											image={
												member.avatar
													? `data:${member.avatar.mimeType};base64, ${member.avatar.image}`
													: ''
											}
										/>
									}
									action={
										<div className='members__member-actions'>
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
					title='Delete the member?'
					actions={
						<>
							<SlButton onClick={onDeleteMemberCancel}>Cancel</SlButton>
							<SlButton variant='danger' onClick={onDeleteMemberConfirmation}>
								Delete
							</SlButton>
						</>
					}
				>
					If you delete the member they will no longer have access to your organization.
				</AlertDialog>
				<InviteMemberDialog
					isOpen={isInviteMemberDialogOpen}
					setIsOpen={setIsInviteMemberDialogOpen}
					setIsLoadingMembers={setIsLoadingMembers}
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

	const onInviteMember = async () => {
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
			label='Invite new member'
			open={isOpen}
			onSlAfterHide={() => setIsOpen(false)}
		>
			<SlInput ref={inputRef} label='Email' value={email} onSlInput={onUpdateEmail} />
			{error && (
				<div className='members__invite-dialog-error'>
					<SlIcon slot='icon' name='exclamation-octagon' />
					{error}
				</div>
			)}
			<SlButton loading={isSaving} className='members__invite' variant='primary' onClick={onInviteMember}>
				Invite
			</SlButton>
		</SlDialog>
	);
};

export default Members;
