import React, { useState, useContext, useRef, ReactNode } from 'react';
import './Action.css';
import ActionHeader from './ActionHeader';
import ActionTransformation from './ActionTransformation';
import ActionFile from './ActionFile';
import ActionQuery from './ActionQuery';
import ActionFilters from './ActionFilters';
import ActionExportMode from './ActionExportMode';
import ActionExportOnDuplicates from './ActionExportOnDuplicates';
import ActionMatching from './ActionMatching';
import ActionTable from './ActionTable';
import { useAction } from './useAction';
import ConnectionContext from '../../../context/ConnectionContext';
import { FullscreenContext } from '../../../context/FullscreenContext';
import ActionContext from '../../../context/ActionContext';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';

const Action = ({ actionType: providedActionType, action: providedAction }) => {
	const [mode, setMode] = useState<'mappings' | 'transformation' | ''>('');
	const [isSaveButtonLoading, setIsSaveButtonLoading] = useState<boolean>(false);

	const { connection } = useContext(ConnectionContext);
	const { closeFullscreen } = useContext(FullscreenContext)!;

	const transformationSectionRef = useRef<ReactNode>();

	const onClose = () => {
		closeFullscreen();
	};

	const {
		isEditing,
		isImport,
		isTransformationFunctionSupported,
		action,
		values,
		setValues,
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
	} = useAction(connection, providedActionType, providedAction, setIsSaveButtonLoading);

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
				mode,
				setMode,
				connection,
				action,
				setAction,
				saveAction,
				values,
				setValues,
				actionType,
				setActionType,
				isEditing,
				isImport,
				isTransformationFunctionSupported,
				onClose,
				transformationSectionRef,
				isTransformationHidden,
				isTransformationDisabled,
				isSaveButtonLoading,
				setIsQueryChanged,
				setIsFileChanged,
				setIsFileConnectorLoading,
				isFileConnectorLoading,
				setIsFileConnectorChanged,
				isFileConnectorChanged,
				setIsTableChanged,
				isSaveHidden,
				setIsSaveHidden,
			}}
		>
			<div className='action'>
				<ActionHeader onClose={onClose} />
				<div className='action__body'>
					{actionType!.fields.includes('Filter') && !isFileStorageImport && <ActionFilters />}
					{actionType!.fields.includes('Query') && <ActionQuery />}
					{actionType!.fields.includes('File') && <ActionFile />}
					{actionType!.fields.includes('Table') && <ActionTable />}
					{actionType!.fields.includes('ExportMode') && <ActionExportMode />}
					{actionType!.fields.includes('Matching') && <ActionMatching />}
					{actionType!.fields.includes('ExportOnDuplicates') && <ActionExportOnDuplicates />}
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
