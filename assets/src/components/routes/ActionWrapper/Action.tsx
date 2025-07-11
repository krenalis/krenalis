import React, { useState, useContext, useRef } from 'react';
import './Action.css';
import ActionHeader from './ActionHeader';
import ActionTransformation from './ActionTransformation';
import ActionFile from './ActionFile';
import ActionQuery from './ActionQuery';
import ActionFilters from './ActionFilters';
import ActionExportMode from './ActionExportMode';
import ActionUpdateOnDuplicates from './ActionUpdateOnDuplicates';
import ActionMatching from './ActionMatching';
import ActionTable from './ActionTable';
import { useAction } from './useAction';
import ConnectionContext from '../../../context/ConnectionContext';
import { FullscreenContext } from '../../../context/FullscreenContext';
import ActionContext from '../../../context/ActionContext';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import Section from '../../base/Section/Section';

const Action = ({ actionType: providedActionType, action: providedAction }) => {
	const [transformationType, setTransformationType] = useState<'mappings' | 'function' | ''>('');
	const [showEmptyMatchingError, setShowEmptyMatchingError] = useState<boolean>(false);

	const { connection } = useContext(ConnectionContext);
	const { closeFullscreen } = useContext(FullscreenContext)!;

	const transformationSectionRef = useRef<any>();
	const matchingSectionRef = useRef<any>();

	const handleEmptyMatchingError = () => {
		setShowEmptyMatchingError(true);
		const top = matchingSectionRef.current.getBoundingClientRect().top;
		matchingSectionRef.current.closest('.fullscreen').scrollBy({
			top: top - 130,
			left: 0,
			behavior: 'smooth',
		});
	};

	const onClose = (cb?: (...args: any) => void) => {
		closeFullscreen(cb);
	};

	const {
		isEditing,
		isImport,
		action,
		settings,
		setSettings,
		isLoading,
		actionType,
		setActionType,
		setAction,
		saveAction,
		isSaveHidden,
		setIsSaveHidden,
		setIsFileChanged,
		setIsFileConnectorLoading,
		isFileConnectorLoading,
		setIsFileConnectorChanged,
		isFileConnectorChanged,
		setIsTableChanged,
		setIsQueryChanged,
		isTransformationHidden,
		isTransformationDisabled,
		selectedInPaths,
		setSelectedInPaths,
		selectedOutPaths,
		setSelectedOutPaths,
		issues,
		setIssues,
		showIssues,
		setShowIssues,
	} = useAction(connection, providedActionType, providedAction);

	if (isLoading) {
		return (
			<div className='action action--loading'>
				<SlSpinner
					style={
						{
							fontSize: '4rem',
							'--track-width': '6px',
						} as React.CSSProperties
					}
				></SlSpinner>
			</div>
		);
	}

	if (action == null || actionType == null) return;

	const isFileStorageImport = connection.isFileStorage && connection.isSource;

	return (
		<ActionContext.Provider
			value={{
				transformationType,
				setTransformationType,
				connection,
				action,
				setAction,
				saveAction,
				settings: settings,
				setSettings: setSettings,
				actionType,
				setActionType,
				isEditing,
				isImport,
				onClose,
				transformationSectionRef,
				handleEmptyMatchingError,
				showEmptyMatchingError,
				isTransformationHidden,
				isTransformationDisabled,
				setIsQueryChanged,
				setIsFileChanged,
				setIsFormatLoading: setIsFileConnectorLoading,
				isFormatLoading: isFileConnectorLoading,
				setIsFormatChanged: setIsFileConnectorChanged,
				isFormatChanged: isFileConnectorChanged,
				setIsTableChanged,
				isSaveHidden,
				setIsSaveHidden,
				selectedInPaths,
				setSelectedInPaths,
				selectedOutPaths,
				setSelectedOutPaths,
				issues,
				setIssues,
				showIssues,
				setShowIssues,
			}}
		>
			<div className='action'>
				<ActionHeader />
				<div className='action__body'>
					{actionType!.fields.includes('Filter') && !isFileStorageImport && <ActionFilters />}
					{actionType!.fields.includes('Query') && <ActionQuery />}
					{actionType!.fields.includes('File') && <ActionFile />}
					{actionType!.fields.includes('TableName') && <ActionTable />}
					{(actionType!.fields.includes('ExportMode') ||
						actionType!.fields.includes('Matching') ||
						actionType!.fields.includes('UpdateOnDuplicates')) && (
						<Section
							title='Export settings'
							description='Select the matching properties that define a match between users, and specify what can be done with users.'
							padded={true}
							className='action__export-settings'
							annotated={true}
						>
							{actionType!.fields.includes('Matching') && <ActionMatching ref={matchingSectionRef} />}
							{actionType!.fields.includes('ExportMode') && <ActionExportMode />}
							{actionType!.fields.includes('UpdateOnDuplicates') && <ActionUpdateOnDuplicates />}
						</Section>
					)}
					{actionType!.fields.includes('Filter') && isFileStorageImport && !isTransformationHidden && (
						<ActionFilters ref={transformationSectionRef} />
					)}
					{actionType!.fields.includes('Transformation') && !isTransformationHidden && (
						<ActionTransformation ref={isFileStorageImport ? null : transformationSectionRef} />
					)}
				</div>
			</div>
		</ActionContext.Provider>
	);
};

export default Action;
