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
		setIsFileChanged,
		setIsTableChanged,
		setIsQueryChanged,
		isMappingSectionDisabled,
		disabledReason,
		mustComputeSchema,
	} = useActionData(connection, providedActionType, providedAction, setIsSaveButtonLoading, workspace);

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
				<ActionHeader onClose={onClose} />
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
