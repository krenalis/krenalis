import React, { useState, useContext } from 'react';
import ActionContext from '../../../context/ActionContext';
import AppContext from '../../../context/AppContext';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';
import { ActionIssues } from './ActionIssues';

const ActionHeader = () => {
	const [isNameEditable, setIsNameEditable] = useState(false);
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { handleError } = useContext(AppContext);

	const {
		connection,
		action,
		actionType,
		setAction,
		saveAction,
		isTransformationHidden,
		isTransformationDisabled,
		isEditing,
		isSaveHidden,
		onClose,
		issues,
		showIssues,
	} = useContext(ActionContext);

	const onUpdateName = (e) => {
		const a = { ...action };
		a.name = e.currentTarget.value;
		setAction(a);
	};

	const onSave = async () => {
		if (isSaving) {
			// prevent duplicated actions caused by multiple clicks.
			return;
		}
		setIsSaving(true);
		const err = await saveAction();
		setTimeout(() => {
			if (err == null) {
				onClose(() => {
					setTimeout(() => {
						// use a timeout to prevent duplicated actions
						// caused by clicks during the closing of the
						// action page.
						setIsSaving(false);
					}, 500);
				});
			} else {
				handleError(err);
				setIsSaving(false);
			}
		}, 200);
	};

	const onCancel = () => {
		onClose();
	};

	return (
		<div className='action__header'>
			<div className='action__header-title'>
				{getConnectorLogo(connection.connector.icon)}
				<div className='action__header-name'>
					{isNameEditable ? (
						<span>
							<SlInput
								className='action__header-name-input'
								value={action != null ? action.name : actionType.name}
								onSlInput={onUpdateName}
							></SlInput>
							<SlIconButton name='check-lg' label='Confirm' onClick={() => setIsNameEditable(false)} />
						</span>
					) : (
						<span>
							{action != null ? action.name : actionType.name}
							<SlIconButton name='pencil' label='Edit' onClick={() => setIsNameEditable(true)} />
						</span>
					)}
				</div>
				{!isNameEditable && <div className='action__header-description'>{actionType.description}</div>}
			</div>
			<ActionIssues issues={issues} type={connection.connector.type} role={connection.role} show={showIssues} />
			<div className={`action__header-buttons${isSaveHidden ? ' action__header-buttons--hidden' : ''}`}>
				<div className='action__header-buttons-save'>
					<SlButton className='action__header-cancel' variant='default' onClick={onCancel}>
						Cancel
					</SlButton>
					<SlButton
						className='action__header-save'
						variant='primary'
						disabled={isTransformationHidden || isTransformationDisabled || isSaving}
						onClick={onSave}
						loading={isSaving}
					>
						{isEditing ? 'Save' : 'Add'}
					</SlButton>
				</div>
			</div>
		</div>
	);
};

export default ActionHeader;
