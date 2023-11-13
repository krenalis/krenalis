import React, { useState, useContext } from 'react';
import ActionContext from '../../../context/ActionContext';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';

interface ActionHeaderProps {
	onClose: () => void;
}

const ActionHeader = ({ onClose }: ActionHeaderProps) => {
	const [isNameEditable, setIsNameEditable] = useState(false);

	const {
		connection,
		action,
		actionType,
		setAction,
		saveAction,
		isMappingSectionDisabled,
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
		await saveAction();
		onClose();
	};

	return (
		<div className='header'>
			<div className='title'>
				<div className='actionTitle'>
					{getConnectorLogo(connection.connector.icon)}
					<div className='actionName'>
						{isNameEditable ? (
							<span className='name'>
								<SlInput
									className='nameInput'
									value={action != null ? action.Name : actionType.Name}
									onSlInput={onUpdateName}
								></SlInput>
								<SlIconButton
									name='check-lg'
									label='Confirm'
									onClick={() => setIsNameEditable(false)}
								/>
							</span>
						) : (
							<span className='name'>
								{action != null ? action.Name : actionType.Name}
								<SlIconButton name='pencil' label='Edit' onClick={() => setIsNameEditable(true)} />
							</span>
						)}
					</div>
					{!isNameEditable && <div className='actionTypeDescription'>{actionType.Description}</div>}
				</div>
			</div>
			<div className={`headerButtons${isSaveHidden ? ' hidden' : ''}`}>
				<SlButton variant='default' onClick={onClose}>
					Cancel
				</SlButton>
				<SlButton
					className='saveAction'
					variant='primary'
					disabled={isMappingSectionDisabled}
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
