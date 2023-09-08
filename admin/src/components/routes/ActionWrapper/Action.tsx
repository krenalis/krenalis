import React, { useState, useContext, useRef, ReactNode } from 'react';
import './Action.css';
import ActionHeader from './ActionHeader';
import ActionMapping from './ActionMapping';
import ActionPath from './ActionPath';
import ActionQuery from './ActionQuery';
import ActionFilters from './ActionFilters';
import ActionExportMode from './ActionExportMode';
import ActionMatchingProperties from './ActionMatchingProperties';
import ActionIdentifiers from './ActionIdentifiers';
import useActionData from '../../../hooks/useActionData';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { FullscreenContext } from '../../../context/FullscreenContext';
import appContext from '../../../context/AppContext';
import ActionContext from '../../../context/ActionContext';
import { SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';

const Action = ({ actionType: providedActionType, action: providedAction }) => {
	const [mode, setMode] = useState<string>('');
	const [isSaveButtonLoading, setIsSaveButtonLoading] = useState<boolean>(false);
	const [isFileChanged, setIsFileChanged] = useState<boolean>(false);
	const [isQueryChanged, setIsQueryChanged] = useState<boolean>(false);

	const { workspace } = useContext(appContext);
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
	} = useActionData(onClose, connection, providedActionType, providedAction, setIsSaveButtonLoading, workspace);

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

	const mustComputeSchema =
		(connection.type === 'Database' || connection.type === 'File') && actionType!.InputSchema == null && !isEditing;
	const hasQueryError = connection.type === 'Database' && actionType!.InputSchema == null && isEditing;
	const hasRecordsError = connection.type === 'File' && actionType!.InputSchema == null && isEditing;
	const isMappingSectionDisabled = hasQueryError || isQueryChanged || hasRecordsError || (isFileChanged && isImport);

	let disabledReason = '';
	if (hasQueryError) {
		disabledReason =
			'Mappings are disabled since the query returned an error. Fix the query before proceding to mappings.';
	} else if (hasRecordsError) {
		disabledReason =
			'Mappings are disabled since the file fetch returned an error. Fix the file informations before proceding to mappings.';
	} else if (connection.type === 'Database') {
		disabledReason =
			'Mappings are disabled since you have modified the query. Confirm the query or undo the changes before proceding to mappings';
	} else {
		disabledReason =
			'Mappings are disabled since you have modified the file informations. Confirm the new informations or undo the changes before proceding to mappings';
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
				isMappingSectionDisabled,
				disabledReason,
				isSaveButtonLoading,
				setIsQueryChanged,
				setIsFileChanged,
			}}
		>
			<div className='action'>
				<ActionHeader />
				<div className='body'>
					{actionType!.Fields.includes('Filter') && <ActionFilters />}
					{actionType!.Fields.includes('Query') && <ActionQuery />}
					{actionType!.Fields.includes('Path') && <ActionPath />}
					{actionType!.Fields.includes('ExportMode') && <ActionExportMode />}
					{actionType!.Fields.includes('MatchingProperties') && <ActionMatchingProperties />}
					{actionType!.Fields.includes('Identifiers') && <ActionIdentifiers />}
					{actionType!.Fields.includes('Mapping') && !mustComputeSchema && (
						<ActionMapping ref={mappingSectionRef} />
					)}
				</div>
			</div>
		</ActionContext.Provider>
	);
};

export default Action;
