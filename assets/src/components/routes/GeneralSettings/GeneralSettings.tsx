import React, { useState, useContext, useEffect, useLayoutEffect, useRef, useMemo } from 'react';
import './GeneralSettings.css';
import * as icons from '../../../constants/icons';
import DangerZone from '../../base/DangerZone/DangerZone';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import { CONFIRM_ANIMATION_DURATION } from '../ActionWrapper/Action.constants';
import appContext from '../../../context/AppContext';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import { ObjectType } from '../../../lib/api/types/types';
import { getUIPreferencesComboboxItems } from '../../helpers/getSchemaComboboxItems';
import { flattenSchema } from '../../../lib/core/action';
import { Combobox } from '../../base/Combobox/Combobox';
import { UIPreferences } from '../../../lib/api/types/workspace';
import { checkUIPreferences } from './GeneralSettings.helpers';

const GeneralSettings = () => {
	const [userSchema, setUserSchema] = useState<ObjectType>();
	const [name, setName] = useState<string>('');
	const [image, setImage] = useState<string>();
	const [firstName, setFirstName] = useState<string>();
	const [lastName, setLastName] = useState<string>();
	const [extra, setExtra] = useState<string>();
	const [isDeleteConfirmationDialogOpen, setIsDeleteConfirmationDialogOpen] = useState<boolean>(false);

	const deleteButtonRef = useRef<any>();

	const {
		api,
		handleError,
		showStatus,
		workspaces,
		setIsLoadingWorkspaces,
		selectedWorkspace,
		setSelectedWorkspace,
		setIsLoadingState,
	} = useContext(appContext);

	useLayoutEffect(() => {
		const ws = workspaces.find((workspace) => workspace.id === selectedWorkspace);
		setName(ws.name);
		setFirstName(ws.uiPreferences.userProfile.firstName);
		setLastName(ws.uiPreferences.userProfile.lastName);
		setExtra(ws.uiPreferences.userProfile.extra);
		setImage(ws.uiPreferences.userProfile.image);
	}, [selectedWorkspace]);

	useEffect(() => {
		const fetchUserSchema = async () => {
			let schema: ObjectType;
			try {
				schema = await api.workspaces.userSchema();
			} catch (err) {
				handleError(err);
				return;
			}
			setUserSchema(schema);
		};
		fetchUserSchema();
	}, []);

	const userSchemaComboboxItems = useMemo(() => getUIPreferencesComboboxItems(userSchema), [userSchema]);

	const flatUserSchema = useMemo(() => flattenSchema(userSchema), [userSchema]);

	const firstNameError = useMemo<string>(
		() => checkUIPreferences(firstName, flatUserSchema),
		[flatUserSchema, firstName],
	);
	const lastNameError = useMemo<string>(
		() => checkUIPreferences(lastName, flatUserSchema),
		[flatUserSchema, lastName],
	);
	const extraError = useMemo<string>(() => checkUIPreferences(extra, flatUserSchema), [flatUserSchema, extra]);
	const imageError = useMemo<string>(() => checkUIPreferences(image, flatUserSchema), [flatUserSchema, image]);

	const onNameInput = (e) => setName(e.target.value);

	const updateProperty = (name: string, value: string) => {
		switch (name) {
			case 'firstName':
				setFirstName(value);
				break;
			case 'lastName':
				setLastName(value);
				break;
			case 'extra':
				setExtra(value);
				break;
			case 'image':
				setImage(value);
				break;
		}
	};

	const onUpdateUIPreferences = (name: string, value: string) => {
		updateProperty(name, value);
	};

	const onSelectUIPreferences = (name: string, value: string) => {
		updateProperty(name, value);
	};

	const onSave = async () => {
		const uiPreferences: UIPreferences = {
			userProfile: {
				firstName,
				lastName,
				extra,
				image,
			},
		};
		try {
			await api.workspaces.update(name, uiPreferences);
		} catch (err) {
			handleError(err);
			return;
		}
		showStatus({ variant: 'success', icon: icons.OK, text: 'Workspace updated successfully' });
		setIsLoadingWorkspaces(true);
	};

	const onDelete = () => setIsDeleteConfirmationDialogOpen(true);

	const onDeleteConfirmation = async () => {
		deleteButtonRef.current!.load();
		try {
			await api.workspaces.delete();
		} catch (err) {
			setTimeout(() => {
				deleteButtonRef.current!.stop();
				setIsDeleteConfirmationDialogOpen(false);
				handleError(err);
			}, CONFIRM_ANIMATION_DURATION);
			return;
		}
		deleteButtonRef.current!.confirm();
		setTimeout(() => {
			setSelectedWorkspace(0);
			setIsLoadingState(true);
		}, CONFIRM_ANIMATION_DURATION);
	};

	const onCancelDeletion = () => {
		setIsDeleteConfirmationDialogOpen(false);
	};

	return (
		<div className='general-settings'>
			<div className='general-settings__title'>General</div>
			<SlInput
				className='general-settings__name'
				maxlength={100}
				label="Workspace's name"
				name='workspace-name'
				value={name}
				onSlInput={onNameInput}
			/>
			<div className='general-settings__user-profile-properties'>
				<div className='general-settings__user-profile-properties-title'>Displayed user properties</div>
				<div className='general-settings__user-profile-properties-description'>
					The properties of the user schema shown in the user pages
				</div>
				<div className='general-settings__user-profile-properties-fields'>
					{firstName !== undefined && (
						<Combobox
							className='general-settings__user-profile-first-name'
							label='First name'
							onInput={onUpdateUIPreferences}
							onSelect={onSelectUIPreferences}
							items={userSchemaComboboxItems}
							value={firstName}
							name='firstName'
							isExpression={false}
							error={firstNameError !== '' ? firstNameError : ''}
							caret={true}
							controlled={true}
						/>
					)}
					{lastName !== undefined && (
						<Combobox
							className='general-settings__user-profile-last-name'
							label='Last name'
							onInput={onUpdateUIPreferences}
							onSelect={onSelectUIPreferences}
							items={userSchemaComboboxItems}
							value={lastName}
							name='lastName'
							isExpression={false}
							error={lastNameError !== '' ? lastNameError : ''}
							caret={true}
							controlled={true}
						/>
					)}
					{extra !== undefined && (
						<Combobox
							className='general-settings__user-profile-extra'
							label='Additional line'
							onInput={onUpdateUIPreferences}
							onSelect={onSelectUIPreferences}
							items={userSchemaComboboxItems}
							value={extra}
							name='extra'
							isExpression={false}
							error={extraError !== '' ? extraError : ''}
							caret={true}
							controlled={true}
						/>
					)}
					{image !== undefined && (
						<Combobox
							className='general-settings__user-profile-image'
							label='Image'
							onInput={onUpdateUIPreferences}
							onSelect={onSelectUIPreferences}
							items={userSchemaComboboxItems}
							value={image}
							name='image'
							isExpression={false}
							error={imageError !== '' ? imageError : ''}
							caret={true}
							controlled={true}
						/>
					)}
				</div>
			</div>
			<SlButton className='general-settings__save-workspace-button' variant='primary' onClick={onSave}>
				Save
			</SlButton>
			<SlDivider />
			<DangerZone>
				<div className='general-settings__deletion-title'>Delete the workspace</div>
				<div className='general-settings__deletion-description-and-button'>
					<div className='general-settings__deletion-description'>Delete permanently the workspace</div>
					<SlButton className='general-settings__deletion-button' variant='danger' onClick={onDelete}>
						Delete
					</SlButton>
				</div>
			</DangerZone>
			<AlertDialog
				variant='danger'
				isOpen={isDeleteConfirmationDialogOpen}
				onClose={onCancelDeletion}
				title='Are you sure?'
				actions={
					<>
						<SlButton onClick={onCancelDeletion}>Cancel</SlButton>
						<FeedbackButton
							ref={deleteButtonRef}
							className='general-settings__alert-deletion-button'
							variant='danger'
							onClick={onDeleteConfirmation}
							animationDuration={CONFIRM_ANIMATION_DURATION}
						>
							Delete
						</FeedbackButton>
					</>
				}
			>
				<p>If you continue, you will lose all the workspace data</p>
			</AlertDialog>
		</div>
	);
};

export default GeneralSettings;
