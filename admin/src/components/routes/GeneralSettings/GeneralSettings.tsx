import React, { useState, useContext, useLayoutEffect, useRef } from 'react';
import './GeneralSettings.css';
import * as icons from '../../../constants/icons';
import DangerZone from '../../base/DangerZone/DangerZone';
import FeedbackButton from '../../base/FeedbackButton/FeedbackButton';
import { CONFIRM_ANIMATION_DURATION } from '../PipelineWrapper/Pipeline.constants';
import appContext from '../../../context/AppContext';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import ConfirmByTyping from '../../base/ConfirmByTyping/ConfirmByTyping';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';

const GeneralSettings = () => {
	const [name, setName] = useState<string>('');
	const [isDeleteConfirmationDialogOpen, setIsDeleteConfirmationDialogOpen] = useState<boolean>(false);
	const [deleteConfirmationInput, setDeleteConfirmationInput] = useState<string>('');

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
		setTitle,
	} = useContext(appContext);

	useLayoutEffect(() => {
		setTitle('Settings / General');
	}, [setTitle]);

	useLayoutEffect(() => {
		const ws = workspaces.find((workspace) => workspace.id === selectedWorkspace);
		if (ws == null) {
			return;
		}
		setName(ws.name);
	}, [selectedWorkspace, workspaces]);

	const onNameInput = (e) => setName(e.target.value);

	const onSave = async () => {
		try {
			await api.workspaces.update(name);
		} catch (err) {
			handleError(err);
			return;
		}
		showStatus({ variant: 'success', icon: icons.OK, text: 'Workspace updated successfully' });
		setIsLoadingWorkspaces(true);
	};

	const isDeleteConfirmed = deleteConfirmationInput === name;

	const onDelete = () => {
		setDeleteConfirmationInput('');
		setIsDeleteConfirmationDialogOpen(true);
	};

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
			setSelectedWorkspace('');
			setIsLoadingState(true);
		}, CONFIRM_ANIMATION_DURATION);
	};

	const onCancelDeletion = () => {
		setIsDeleteConfirmationDialogOpen(false);
		setDeleteConfirmationInput('');
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
				title={<span>Delete the workspace?</span>}
				actions={
					<>
						<SlButton onClick={onCancelDeletion}>Cancel</SlButton>
						<FeedbackButton
							ref={deleteButtonRef}
							className='general-settings__alert-deletion-button'
							variant='danger'
							onClick={onDeleteConfirmation}
							animationDuration={CONFIRM_ANIMATION_DURATION}
							disabled={!isDeleteConfirmed}
						>
							Delete workspace
						</FeedbackButton>
					</>
				}
			>
				<ul className='general-settings__grid-alert-workspace-ul'>
					<li>The workspace will be permanently removed.</li>
					<li>Data stored in the connected data warehouse will NOT be deleted.</li>
					<li>
						The data warehouse will not be able to connect to another Krenalis workspace unless its contents
						are manually cleared.
					</li>
				</ul>
				<ConfirmByTyping
					confirmText={name}
					value={deleteConfirmationInput}
					onInput={setDeleteConfirmationInput}
				/>
			</AlertDialog>
		</div>
	);
};

export default GeneralSettings;
