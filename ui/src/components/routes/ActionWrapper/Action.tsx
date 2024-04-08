import React, { useState, useContext, useRef, ReactNode } from 'react';
import './Action.css';
import ActionHeader from './ActionHeader';
import ActionMapping from './ActionMapping';
import ActionFile from './ActionFile';
import ActionQuery from './ActionQuery';
import ActionFilters from './ActionFilters';
import ActionExportMode from './ActionExportMode';
import ActionExportOnDuplicatedUsers from './ActionExportOnDuplicatedUsers';
import ActionMatchingProperties from './ActionMatchingProperties';
import ActionTable from './ActionTable';
import { useAction } from '../../../hooks/useActionData';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { FullscreenContext } from '../../../context/FullscreenContext';
import ActionContext from '../../../context/ActionContext';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';

const Action = ({ actionType: providedActionType, action: providedAction }) => {
	const [mode, setMode] = useState<'mappings' | 'transformation' | ''>('');
	const [isSaveButtonLoading, setIsSaveButtonLoading] = useState<boolean>(false);

	const { connection } = useContext(ConnectionContext);
	const { closeFullscreen } = useContext(FullscreenContext)!;

	const mappingSectionRef = useRef<ReactNode>();

	const onClose = () => {
		closeFullscreen();
	};

	const {
		isEditing,
		isImport,
		isTransformationAllowed,
		action,
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
		isMappingHidden,
		isMappingDisabled,
		mappingDisabledReason,
	} = useAction(connection, providedActionType, providedAction, setIsSaveButtonLoading);

	if (isLoading) {
		return (
			<div className='action isLoading'>
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
				actionType,
				setActionType,
				isEditing,
				isImport,
				isTransformationAllowed,
				onClose,
				mappingSectionRef,
				isMappingDisabled,
				mappingDisabledReason,
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
				<div className='body'>
					{actionType!.Fields.includes('Filter') && <ActionFilters />}
					{actionType!.Fields.includes('Query') && <ActionQuery />}
					{actionType!.Fields.includes('File') && <ActionFile />}
					{actionType!.Fields.includes('Table') && <ActionTable />}
					{actionType!.Fields.includes('ExportMode') && <ActionExportMode />}
					{actionType!.Fields.includes('MatchingProperties') && <ActionMatchingProperties />}
					{actionType!.Fields.includes('ExportOnDuplicatedUsers') && <ActionExportOnDuplicatedUsers />}
					{actionType!.Fields.includes('Mapping') && !isMappingHidden && (
						<ActionMapping ref={mappingSectionRef} />
					)}
				</div>
			</div>
		</ActionContext.Provider>
	);
};

export default Action;
