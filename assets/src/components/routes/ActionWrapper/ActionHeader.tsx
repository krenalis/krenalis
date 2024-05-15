import React, { useState, useContext } from 'react';
import ActionContext from '../../../context/ActionContext';
import AppContext from '../../../context/AppContext';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';

interface ActionHeaderProps {
	onClose: () => void;
}

const ActionHeader = ({ onClose }: ActionHeaderProps) => {
	const [isNameEditable, setIsNameEditable] = useState(false);

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
		isSaveButtonLoading,
		isSaveHidden,
	} = useContext(ActionContext);

	const onUpdateName = (e) => {
		const a = { ...action };
		a.Name = e.currentTarget.value;
		setAction(a);
	};

	const onSave = async () => {
		const err = await saveAction();
		if (err == null) {
			onClose();
		} else {
			handleError(err);
		}
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
								value={action != null ? action.Name : actionType.Name}
								onSlInput={onUpdateName}
							></SlInput>
							<SlIconButton name='check-lg' label='Confirm' onClick={() => setIsNameEditable(false)} />
						</span>
					) : (
						<span>
							{action != null ? action.Name : actionType.Name}
							<SlIconButton name='pencil' label='Edit' onClick={() => setIsNameEditable(true)} />
						</span>
					)}
				</div>
				{!isNameEditable && <div className='action__header-description'>{actionType.Description}</div>}
			</div>
			<div className={`action__header-buttons${isSaveHidden ? ' action__header-buttons--hidden' : ''}`}>
				<SlButton className='action__header-cancel' variant='default' onClick={onClose}>
					Cancel
				</SlButton>
				<SlButton
					className='action__header-save'
					variant='primary'
					disabled={isTransformationHidden || isTransformationDisabled}
					onClick={onSave}
					loading={isSaveButtonLoading}
				>
					{isEditing ? 'Save' : 'Add'}
				</SlButton>
			</div>
		</div>
	);
};

export default ActionHeader;
