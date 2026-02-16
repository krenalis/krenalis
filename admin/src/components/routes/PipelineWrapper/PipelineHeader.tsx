import React, { useState, useContext, useEffect } from 'react';
import PipelineContext from '../../../context/PipelineContext';
import AppContext from '../../../context/AppContext';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';
import { PipelineIssues } from './PipelineIssues';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

const PipelineHeader = () => {
	const [isNameEditable, setIsNameEditable] = useState(false);
	const [isFullscreenClosing, setIsFullscreenClosing] = useState(false);
	const [isSaving, setIsSaving] = useState<boolean>(false);

	const { handleError } = useContext(AppContext);

	const {
		connection,
		pipeline,
		pipelineType,
		setPipeline,
		savePipeline,
		isTransformationHidden,
		isTransformationDisabled,
		isFullscreenTransformationOpen,
		setIsFullscreenTransformationOpen,
		isEditing,
		onClose,
		issues,
		showIssues,
	} = useContext(PipelineContext);

	useEffect(() => {
		const button = document.querySelector<HTMLButtonElement>('.pipeline__header-save');
		if (!button) {
			return;
		}

		const handleMouseEnter = () => {
			if (button.disabled) {
				button.addEventListener('mouseleave', handleMouseLeave, { once: true });
			}
		};

		const handleMouseLeave = () => {
			button.disabled = false;
			button.classList.remove('pipeline__header-save--fullscreen-closing-disabled');
		};

		button.addEventListener('mouseenter', handleMouseEnter);

		return () => {
			button.removeEventListener('mouseenter', handleMouseEnter);
			button.removeEventListener('mouseleave', handleMouseLeave);
		};
	}, []);

	const onUpdateName = (e) => {
		const p = { ...pipeline };
		p.name = e.currentTarget.value;
		setPipeline(p);
	};

	const onSave = async () => {
		if (isSaving) {
			// prevent duplicated pipelines caused by multiple clicks.
			return;
		}
		setIsSaving(true);
		const err = await savePipeline();
		setTimeout(() => {
			if (err == null) {
				onClose(() => {
					setTimeout(() => {
						// use a timeout to prevent duplicated pipelines
						// caused by clicks during the closing of the
						// pipeline page.
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

	const onCloseFullscreenTransformation = () => {
		setIsFullscreenTransformationOpen(false);
		setIsFullscreenClosing(true);
		setTimeout(() => {
			setIsFullscreenClosing(false);
		}, 1000);
	};

	return (
		<div className='pipeline__header'>
			<div className='pipeline__header-title'>
				<LittleLogo code={connection.connector.code} path={CONNECTORS_ASSETS_PATH} />
				<div className='pipeline__header-name'>
					{isNameEditable ? (
						<span>
							<SlInput
								className='pipeline__header-name-input'
								value={pipeline != null ? pipeline.name : pipelineType.name}
								onSlInput={onUpdateName}
								maxlength={60}
							></SlInput>
							<SlIconButton name='check-lg' label='Confirm' onClick={() => setIsNameEditable(false)} />
						</span>
					) : (
						<span>
							{pipeline != null ? pipeline.name : pipelineType.name}
							<SlIconButton name='pencil' label='Edit' onClick={() => setIsNameEditable(true)} />
						</span>
					)}
				</div>
				{!isNameEditable && <div className='pipeline__header-description'>{pipelineType.description}</div>}
			</div>
			{issues != null && issues.length > 0 ? (
				<PipelineIssues
					issues={issues}
					type={connection.connector.type}
					role={connection.role}
					show={showIssues}
				/>
			) : (
				<div className='pipeline__header-issues-placeholder' /> // Render an empty div to maintain the grid layout
			)}
			<div className='pipeline__header-buttons'>
				<div
					className={`pipeline__header-buttons-save${isFullscreenTransformationOpen ? ' pipeline__header-buttons-save--hidden' : ''}`}
				>
					<SlButton className='pipeline__header-cancel' variant='default' onClick={onCancel}>
						Cancel
					</SlButton>
					<SlButton
						className={`pipeline__header-save${isFullscreenClosing ? ' pipeline__header-save--fullscreen-closing-disabled' : ''}`}
						variant='primary'
						disabled={isTransformationHidden || isTransformationDisabled || isSaving || isFullscreenClosing}
						onClick={onSave}
						loading={isSaving}
					>
						{isEditing ? 'Save' : 'Add'}
					</SlButton>
				</div>
				<div
					className={`pipeline__header-buttons-close-transformation${!isFullscreenTransformationOpen ? ' pipeline__header-buttons-close-transformation--hidden' : ''}`}
				>
					<button
						className='pipeline__header-close-transformation'
						onClick={onCloseFullscreenTransformation}
						title='Exit full mode'
					>
						×
					</button>
				</div>
			</div>
		</div>
	);
};

export default PipelineHeader;
