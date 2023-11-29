import React, { useState, useContext, useLayoutEffect, useRef } from 'react';
import './GeneralSettings.css';
import * as icons from '../../../constants/icons';
import DangerZone from '../../shared/DangerZone/DangerZone';
import FeedbackButton from '../../shared/FeedbackButton/FeedbackButton';
import { CONFIRM_ANIMATION_DURATION } from '../ActionWrapper/Action.constants';
import appContext from '../../../context/AppContext';
import AlertDialog from '../../shared/AlertDialog/AlertDialog';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import { UnprocessableError } from '../../../lib/api/errors';
import { Variant } from '../../../types/internal/app';

const GeneralSettings = () => {
	const [name, setName] = useState<string>('');
	const [useEuropeRegion, setUseEuropeRegion] = useState<boolean>(false);
	const [isDeleteConfirmationDialogOpen, setIsDeleteConfirmationDialogOpen] = useState<boolean>(false);

	const deleteButtonRef = useRef<any>();

	const {
		api,
		showError,
		showStatus,
		redirect,
		workspaces,
		setIsLoadingWorkspaces,
		selectedWorkspace,
		setSelectedWorkspace,
		setIsLoadingState,
	} = useContext(appContext);

	useLayoutEffect(() => {
		const workspace = workspaces.find((workspace) => workspace.ID === selectedWorkspace);
		setName(workspace.Name);
		if (workspace.PrivacyRegion === 'Europe') {
			setUseEuropeRegion(true);
		} else {
			setUseEuropeRegion(false);
		}
	}, [selectedWorkspace]);

	const onNameChange = (e) => setName(e.target.value);

	const onUseEuropeRegionChange = () => setUseEuropeRegion(!useEuropeRegion);

	const onUpdate = async () => {
		const privacyRegion = useEuropeRegion ? 'Europe' : '';
		try {
			await api.workspaces.update(name, privacyRegion);
		} catch (err) {
			showError(err);
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
					const onClick = () => redirect('settings/data-warehouse');
					const status = {
						variant: 'danger' as Variant,
						icon: icons.EXCLAMATION,
						text: 'You must disconnect the data warehouse first',
						action: {
							name: 'Disconnect data warehouse...',
							onClick: onClick,
						},
					};
					setTimeout(() => {
						deleteButtonRef.current!.stop();
						setIsDeleteConfirmationDialogOpen(false);
						showStatus(status);
					}, CONFIRM_ANIMATION_DURATION);
					return;
				}
			}
			setTimeout(() => {
				deleteButtonRef.current!.stop();
				setIsDeleteConfirmationDialogOpen(false);
				showError(err);
			}, CONFIRM_ANIMATION_DURATION);
			return;
		}
		deleteButtonRef.current!.confirm();
		setTimeout(() => {
			setIsLoadingState(true);
			setSelectedWorkspace(0);
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
			<SlButton className='general-settings__update-workspace-button' variant='primary' onClick={onUpdate}>
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
							className='general-settings__deletion-button'
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
