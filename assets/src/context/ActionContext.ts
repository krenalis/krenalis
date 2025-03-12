import { createContext } from 'react';
import TransformedConnection from '../lib/core/connection';
import { TransformedAction, TransformedActionType } from '../lib/core/action';
import { ConnectorSettings } from '../lib/api/types/responses';

interface ActionContext {
	transformationType: 'mappings' | 'function' | '';
	setTransformationType: React.Dispatch<React.SetStateAction<'mappings' | 'function' | ''>>;
	connection: TransformedConnection;
	action: TransformedAction;
	setAction: React.Dispatch<React.SetStateAction<TransformedAction | undefined>>;
	saveAction: () => Promise<string | Error | null>;
	settings: ConnectorSettings;
	setSettings: React.Dispatch<React.SetStateAction<ConnectorSettings>>;
	actionType: TransformedActionType;
	setActionType: React.Dispatch<React.SetStateAction<TransformedActionType | undefined>>;
	isEditing: boolean;
	isImport: boolean;
	isTransformationFunctionSupported: boolean;
	onClose: (cb?: (...args: any) => void) => void;
	transformationSectionRef: React.MutableRefObject<any>;
	handleEmptyMatchingError: () => void;
	showEmptyMatchingError: boolean;
	isTransformationHidden: boolean;
	isTransformationDisabled: boolean;
	setIsQueryChanged: React.Dispatch<React.SetStateAction<boolean>>;
	setIsFileChanged: React.Dispatch<React.SetStateAction<boolean>>;
	setIsFormatLoading: React.Dispatch<React.SetStateAction<boolean>>;
	isFormatLoading: boolean;
	setIsFormatChanged: React.Dispatch<React.SetStateAction<boolean>>;
	isFormatChanged: boolean;
	setIsTableChanged: React.Dispatch<React.SetStateAction<boolean>>;
	isSaveHidden: boolean;
	setIsSaveHidden: React.Dispatch<React.SetStateAction<boolean>>;
	selectedInPaths: string[];
	setSelectedInPaths: React.Dispatch<React.SetStateAction<string[]>>;
	selectedOutPaths: string[];
	setSelectedOutPaths: React.Dispatch<React.SetStateAction<string[]>>;
	issues: string[];
	setIssues: React.Dispatch<React.SetStateAction<string[]>>;
	showIssues: boolean;
	setShowIssues: React.Dispatch<React.SetStateAction<boolean>>;
}

const actionContext = createContext<ActionContext>({} as ActionContext);

export default actionContext;
