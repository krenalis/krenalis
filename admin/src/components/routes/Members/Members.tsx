import React, { useState, useContext, useLayoutEffect, useEffect, useRef } from 'react';
import './Members.css';
import AppContext from '../../../context/AppContext';
import ListTile from '../../shared/ListTile/ListTile';
import AlertDialog from '../../shared/AlertDialog/AlertDialog';
import { Member, MemberAvatar, MemberToSet } from '../../../types/external/api';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import SlDialog from '@shoelace-style/shoelace/dist/react/dialog/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import API from '../../../lib/api/api';
import { UnprocessableError } from '../../../lib/api/errors';
import { toBase64 } from '../../../lib/utils/toBase64';
import { TransformedMember, transformMember, validateMemberToSet } from '../../../lib/helpers/transformedMember';

const Members = () => {
	const [isLoadingMembers, setIsLoadingMembers] = useState<boolean>(true);
	const [isRemoveAlertOpen, setIsRemoveAlertOpen] = useState<boolean>(false);
	const [isAddMemberDialogOpen, setIsAddMemberDialogOpen] = useState<boolean>(false);
	const [members, setMembers] = useState<TransformedMember[]>();

	const { setTitle, api, showError, redirect } = useContext(AppContext);

	const pendingDeletedMember = useRef<number>(0);

	useLayoutEffect(() => {
		setTitle('Members');
	}, []);

	useEffect(() => {
		const fetchMembers = async () => {
			let members: Member[];
			try {
				members = await api.members();
			} catch (err) {
				showError(err);
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

	const onEditMember = (id: number) => {
		redirect(`members/${id}`);
	};

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
			showError(err);
			return;
		}
		setIsRemoveAlertOpen(false);
		setIsLoadingMembers(true);
	};

	if (isLoadingMembers) {
		return (
			<div className='members__content'>
				<div className='members'>
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
			<div className='members__content'>
				<div className='members'>
					<div className='members__title'>
						<p className='members__title-text'>Members</p>
						<SlButton size='small' variant='primary' onClick={() => setIsAddMemberDialogOpen(true)}>
							Add a new member
						</SlButton>
					</div>
					<div className='members__list'>
						{members.map((member) => {
							return (
								<ListTile
									key={member.ID}
									className='members__member'
									name={member.Name}
									description={member.Email}
									icon={
										<SlAvatar
											initials={member.Initials}
											image={
												member.Avatar
													? `data:${member.Avatar.MimeType};base64, ${member.Avatar.Image}`
													: ''
											}
										/>
									}
									action={
										<div className='members__member-actions'>
											<SlButton size='small' onClick={() => onEditMember(member.ID)}>
												Edit
											</SlButton>
											<SlButton
												size='small'
												variant='danger'
												onClick={() => onDeleteMember(member.ID)}
											>
												Delete
											</SlButton>
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
					title='Are you sure?'
					actions={
						<>
							<SlButton onClick={onDeleteMemberCancel}>Cancel</SlButton>
							<SlButton variant='danger' onClick={onDeleteMemberConfirmation}>
								Delete
							</SlButton>
						</>
					}
				>
					If you delete the member they will no longer have access to your organization
				</AlertDialog>
				<AddMemberDialog
					isOpen={isAddMemberDialogOpen}
					setIsOpen={setIsAddMemberDialogOpen}
					showError={showError}
					setIsLoadingMembers={setIsLoadingMembers}
					api={api}
				/>
			</div>
		);
	}
};

interface AddMemberDialogProps {
	isOpen: boolean;
	setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
	showError: (err: Error | string) => void;
	setIsLoadingMembers: React.Dispatch<React.SetStateAction<boolean>>;
	api: API;
}

const AddMemberDialog = ({ isOpen, setIsOpen, showError, setIsLoadingMembers, api }: AddMemberDialogProps) => {
	const [avatar, setAvatar] = useState<MemberAvatar | null>(null);
	const [name, setName] = useState<string>('');
	const [email, setEmail] = useState<string>('');
	const [password, setPassword] = useState<string>('');
	const [isSaving, setIsSaving] = useState<boolean>(false);
	const [error, setError] = useState<string>('');

	const fileInputRef = useRef<any>();
	const nameInputRef = useRef<any>();

	useEffect(() => {
		if (isOpen) {
			setTimeout(() => {
				nameInputRef.current.focus();
			}, 100);
		}
	}, [isOpen]);

	const onUpdateAvatar = async (e: any) => {
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
		setAvatar({ Image: base64, MimeType: f.type });
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

	const onUpdatePassword = (e) => {
		const value = e.target.value;
		setPassword(value);
	};

	const onAddMember = async () => {
		setError('');
		setIsSaving(true);
		const memberToSet: MemberToSet = {
			Name: name,
			Image: avatar ? avatar.Image : null,
			Email: email,
			Password: password,
		};
		const err = validateMemberToSet(memberToSet, true);
		if (err !== '') {
			setTimeout(() => {
				setIsSaving(false);
				setError(err);
			}, 300);
			return;
		}
		try {
			await api.addMember(memberToSet);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				setTimeout(() => {
					setIsSaving(false);
					setError(err.message);
				}, 300);
			} else {
				setTimeout(() => {
					setIsSaving(false);
					setIsOpen(false);
					setTimeout(() => {
						showError(err);
					}, 150);
				}, 300);
			}
			return;
		}
		setTimeout(() => {
			setIsSaving(false);
			setIsOpen(false);
			setTimeout(() => {
				setIsLoadingMembers(true);
			}, 300);
		}, 300);
	};

	return (
		<SlDialog
			className='members__add-dialog'
			label='Add new member'
			open={isOpen}
			onSlAfterHide={() => setIsOpen(false)}
		>
			<SlInput ref={nameInputRef} required label='Name' value={name} onSlInput={onUpdateName} />
			<SlInput required label='Email' value={email} onSlInput={onUpdateEmail} />
			<SlInput
				required
				type='password'
				label='Password'
				value={password}
				onSlInput={onUpdatePassword}
				password-toggle
			/>
			<label>
				<div className='members__add-dialog-avatar-label'>Avatar</div>
				<div className='members__add-dialog-avatar-box'>
					<div className='members__dialog-avatar-buttons'>
						<div className='members__add-dialog-add-avatar'>Upload</div>
						{avatar && (
							<div className='members__add-dialog-remove-avatar' onClick={onDeleteAvatar}>
								Delete
							</div>
						)}
					</div>
					<SlAvatar image={avatar ? `data:${avatar.MimeType};base64, ${avatar.Image}` : ''} />
					<input ref={fileInputRef} type='file' accept='image/jpeg, image/png' onChange={onUpdateAvatar} />
				</div>
			</label>
			{error && (
				<div className='members__add-dialog-error'>
					<SlIcon slot='icon' name='exclamation-octagon' />
					{error}
				</div>
			)}
			<SlButton loading={isSaving} className='members__add' variant='primary' onClick={onAddMember}>
				Add
			</SlButton>
		</SlDialog>
	);
};

export default Members;
