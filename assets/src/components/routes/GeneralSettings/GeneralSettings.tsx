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
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import { UnprocessableError } from '../../../lib/api/errors';
import { ObjectType } from '../../../lib/api/types/types';
import { ComboBoxInput, ComboBoxList } from '../../base/ComboBox/ComboBox';
import { getSchemaComboboxItems } from '../../helpers/getSchemaComboBoxItems';
import { flattenSchema } from '../../../lib/helpers/transformedAction';
import { checkDisplayedProperty } from './GeneralSettings.helpers';

const GeneralSettings = () => {
	const [userSchema, setUserSchema] = useState<ObjectType>();
	const [name, setName] = useState<string>('');
	const [useEuropeRegion, setUseEuropeRegion] = useState<boolean>(false);
	const [image, setImage] = useState<string>('');
	const [firstName, setFirstName] = useState<string>('');
	const [lastName, setLastName] = useState<string>('');
	const [information, setInformation] = useState<string>('');
	const [isDeleteConfirmationDialogOpen, setIsDeleteConfirmationDialogOpen] = useState<boolean>(false);

	const deleteButtonRef = useRef<any>();
	const userSchemaListRef = useRef<any>();

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
		const ws = workspaces.find((workspace) => workspace.ID === selectedWorkspace);
		setName(ws.Name);
		setUseEuropeRegion(ws.PrivacyRegion === 'Europe');
		setImage(ws.DisplayedProperties.Image);
		setFirstName(ws.DisplayedProperties.FirstName);
		setLastName(ws.DisplayedProperties.LastName);
		setInformation(ws.DisplayedProperties.Information);
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

	const userSchemaComboboxItems = useMemo(() => getSchemaComboboxItems(userSchema), [userSchema]);

	const flatUserSchema = useMemo(() => flattenSchema(userSchema), [userSchema]);

	const imageError = useMemo<string>(() => checkDisplayedProperty(image, flatUserSchema), [flatUserSchema, image]);
	const firstNameError = useMemo<string>(
		() => checkDisplayedProperty(firstName, flatUserSchema),
		[flatUserSchema, firstName],
	);
	const lastNameError = useMemo<string>(
		() => checkDisplayedProperty(lastName, flatUserSchema),
		[flatUserSchema, lastName],
	);
	const informationError = useMemo<string>(
		() => checkDisplayedProperty(information, flatUserSchema),
		[flatUserSchema, information],
	);

	const onNameChange = (e) => setName(e.target.value);

	const onUseEuropeRegionChange = () => setUseEuropeRegion(!useEuropeRegion);

	const updateProperty = (name: string, value: string) => {
		switch (name) {
			case 'image':
				setImage(value);
				break;
			case 'firstName':
				setFirstName(value);
				break;
			case 'lastName':
				setLastName(value);
				break;
			case 'information':
				setInformation(value);
		}
	};

	const onUpdateDisplayedProperty = (e) => {
		updateProperty(e.target.name, e.target.value);
	};

	const onSelectDisplayedProperty = (input, v) => {
		const n = input.name;
		updateProperty(n, v);
	};

	const onSave = async () => {
		const privacyRegion = useEuropeRegion ? 'Europe' : '';
		const displayedProperties = {
			Image: image,
			FirstName: firstName,
			LastName: lastName,
			Information: information,
		};
		try {
			await api.workspaces.update(name, privacyRegion, displayedProperties);
		} catch (err) {
			handleError(err);
			return;
		}
		showStatus({ variant: 'success', icon: icons.OK, text: 'Workspace updated succesfully' });
		setIsLoadingWorkspaces(true);
	};

	const onDelete = () => setIsDeleteConfirmationDialogOpen(true);

	const onDeleteConfirmation = async () => {
		deleteButtonRef.current!.load();
		try {
			await api.workspaces.delete();
		} catch (err) {
			if (err instanceof UnprocessableError) {
				if (err.code === 'CurrentlyConnected') {
					setTimeout(() => {
						deleteButtonRef.current!.stop();
						setIsDeleteConfirmationDialogOpen(false);
						handleError('You must disconnect the data warehouse first');
					}, CONFIRM_ANIMATION_DURATION);
					return;
				}
			}
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
				value={name}
				onSlChange={onNameChange}
			/>
			<SlCheckbox
				className='general-settings__use-europe-region'
				checked={useEuropeRegion}
				onSlChange={onUseEuropeRegionChange}
			>
				Use the European Privacy Region for this workspace
			</SlCheckbox>
			<div className='general-settings__displayed-properties'>
				<div className='general-settings__displayed-properties-title'>Displayed user properties</div>
				<div className='general-settings__displayed-properties-description'>
					The properties of the user schema shown in the user pages
				</div>
				<div className='general-settings__displayed-properties-fields'>
					<ComboBoxInput
						className='general-settings__displayed-first-name'
						label='First name'
						comboBoxListRef={userSchemaListRef}
						onInput={onUpdateDisplayedProperty}
						value={firstName}
						name='firstName'
						error={firstNameError && firstNameError}
						caret={true}
					/>
					<ComboBoxInput
						className='general-settings__displayed-last-name'
						label='Last name'
						comboBoxListRef={userSchemaListRef}
						onInput={onUpdateDisplayedProperty}
						value={lastName}
						name='lastName'
						error={lastNameError && lastNameError}
						caret={true}
					/>
					<ComboBoxInput
						className='general-settings__displayed-information'
						label='Additional line'
						comboBoxListRef={userSchemaListRef}
						onInput={onUpdateDisplayedProperty}
						value={information}
						name='information'
						error={informationError && informationError}
						caret={true}
					/>
					<ComboBoxInput
						className='general-settings__displayed-image'
						label='Image'
						comboBoxListRef={userSchemaListRef}
						onInput={onUpdateDisplayedProperty}
						value={image}
						name='image'
						error={imageError && imageError}
						caret={true}
					/>
				</div>
				<ComboBoxList
					ref={userSchemaListRef}
					items={userSchemaComboboxItems}
					onSelect={onSelectDisplayedProperty}
				/>
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
