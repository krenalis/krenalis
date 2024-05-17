import React, { useState, useContext, useRef, ReactNode } from 'react';
import './Action.css';
import ActionHeader from './ActionHeader';
import ActionTransformation from './ActionTransformation';
import ActionFile from './ActionFile';
import ActionQuery from './ActionQuery';
import ActionFilters from './ActionFilters';
import ActionExportMode from './ActionExportMode';
import ActionExportOnDuplicatedUsers from './ActionExportOnDuplicatedUsers';
import ActionMatchingProperties from './ActionMatchingProperties';
import ActionTable from './ActionTable';
import { useAction } from './useActionData';
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
					{actionType!.Fields.includes('Filter') && <ActionFilters />}
					{actionType!.Fields.includes('Query') && <ActionQuery />}
					{actionType!.Fields.includes('File') && <ActionFile />}
					{actionType!.Fields.includes('Table') && <ActionTable />}
					{actionType!.Fields.includes('ExportMode') && <ActionExportMode />}
					{actionType!.Fields.includes('MatchingProperties') && <ActionMatchingProperties />}
					{actionType!.Fields.includes('ExportOnDuplicatedUsers') && <ActionExportOnDuplicatedUsers />}
					{actionType!.Fields.includes('Transformation') && !isTransformationHidden && (
						<ActionTransformation ref={transformationSectionRef} />
					)}
				</div>
			</div>
		</ActionContext.Provider>
	);
};

export default Action;
