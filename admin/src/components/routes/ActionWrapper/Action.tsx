import React, { useState, useContext, useRef, ReactNode } from 'react';
import './Action.css';
import ActionHeader from './ActionHeader';
import ActionMapping from './ActionMapping';
import ActionPath from './ActionPath';
import ActionQuery from './ActionQuery';
import ActionFilters from './ActionFilters';
import ActionExportMode from './ActionExportMode';
import ActionMatchingProperties from './ActionMatchingProperties';
import ActionTable from './ActionTable';
import useActionData from '../../../hooks/useActionData';
import { ConnectionContext } from '../../../context/providers/ConnectionProvider';
import { FullscreenContext } from '../../../context/FullscreenContext';
import appContext from '../../../context/AppContext';
import ActionContext from '../../../context/ActionContext';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';

const Action = ({ actionType: providedActionType, action: providedAction }) => {
	const [mode, setMode] = useState<string>('');
	const [isSaveButtonLoading, setIsSaveButtonLoading] = useState<boolean>(false);
	const [isFileChanged, setIsFileChanged] = useState<boolean>(false);
	const [isTableChanged, setIsTableChanged] = useState<boolean>(false);
	const [isQueryChanged, setIsQueryChanged] = useState<boolean>(false);

	const { workspaces, selectedWorkspace } = useContext(appContext);
	const { connection } = useContext(ConnectionContext);
	const { closeFullscreen } = useContext(FullscreenContext)!;

	const mappingSectionRef = useRef<ReactNode>();

	const onClose = () => {
		closeFullscreen();
	};

	const workspace = workspaces.find((w) => w.ID === selectedWorkspace);
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
		((connection.type === 'Database' || connection.type === 'File') &&
			actionType!.InputSchema == null &&
			!isEditing) ||
		(connection.type === 'Database' &&
			connection.role === 'Destination' &&
			actionType!.OutputSchema == null &&
			!isEditing);
	const hasQueryError =
		connection.type === 'Database' && connection.role === 'Source' && actionType!.InputSchema == null && isEditing;
	const hasRecordsError =
		connection.type === 'File' && connection.role === 'Destination' && actionType!.InputSchema == null && isEditing;
	const hasTableError = connection.type === 'Database' && actionType!.OutputSchema == null && isEditing;
	const isMappingSectionDisabled =
		hasQueryError ||
		isQueryChanged ||
		hasRecordsError ||
		(isFileChanged && isImport) ||
		hasTableError ||
		isTableChanged;

	let disabledReason = '';
	if (hasQueryError) {
		disabledReason =
			'Mappings are disabled since the query returned an error. Fix the query before proceeding to mappings.';
	} else if (hasRecordsError) {
		disabledReason =
			'Mappings are disabled due to an error in the file fetch. Please resolve the file information issue before proceeding with the mappings.';
	} else if (hasTableError) {
		disabledReason = `Mappings are disabled because the table couldn't be retrieved. Please resolve this issue before proceeding with the mappings.`;
	} else if (connection.type === 'Database' && connection.role === 'Source') {
		disabledReason =
			'Mappings are disabled since the query has been modified. Please confirm the query or revert the changes before proceeding with mappings.';
	} else if (connection.type === 'Database' && connection.role === 'Destination') {
		disabledReason =
			'Mappings are disabled since the table name has been modified. Please confirm the table name or revert the changes before proceeding with mappings.';
	} else {
		disabledReason =
			'Mappings are disabled since the file information has been modified . Please confirm the new information or revert the changes before proceeding with mappings.';
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
				setIsTableChanged,
				isSaveHidden,
				setIsSaveHidden,
			}}
		>
			<div className='action'>
				<ActionHeader />
				<div className='body'>
					{actionType!.Fields.includes('Filter') && <ActionFilters />}
					{actionType!.Fields.includes('Query') && <ActionQuery />}
					{actionType!.Fields.includes('Path') && <ActionPath />}
					{actionType!.Fields.includes('Table') && <ActionTable />}
					{actionType!.Fields.includes('ExportMode') && <ActionExportMode />}
					{actionType!.Fields.includes('MatchingProperties') && <ActionMatchingProperties />}
					{actionType!.Fields.includes('Mapping') && !mustComputeSchema && (
						<ActionMapping ref={mappingSectionRef} />
					)}
				</div>
			</div>
		</ActionContext.Provider>
	);
};

export default Action;
