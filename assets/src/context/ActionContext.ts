import { createContext } from 'react';
import TransformedConnection from '../lib/core/connection';
import { TransformedAction, TransformedActionType } from '../lib/core/action';
import { ConnectorValues } from '../lib/api/types/responses';

interface ActionContext {
	mode: 'mappings' | 'transformation' | '';
	setMode: React.Dispatch<React.SetStateAction<'mappings' | 'transformation' | ''>>;
	connection: TransformedConnection;
	action: TransformedAction;
	setAction: React.Dispatch<React.SetStateAction<TransformedAction | undefined>>;
	saveAction: () => Promise<string | Error | null>;
	values: ConnectorValues;
	setValues: React.Dispatch<React.SetStateAction<ConnectorValues>>;
	actionType: TransformedActionType;
	setActionType: React.Dispatch<React.SetStateAction<TransformedActionType | undefined>>;
	isEditing: boolean;
	isImport: boolean;
	isTransformationFunctionSupported: boolean;
	onClose: () => void;
	transformationSectionRef: React.MutableRefObject<any>;
	isTransformationHidden: boolean;
	isTransformationDisabled: boolean;
	isSaveButtonLoading: boolean;
	setIsQueryChanged: React.Dispatch<React.SetStateAction<boolean>>;
	setIsFileChanged: React.Dispatch<React.SetStateAction<boolean>>;
	setIsFileConnectorLoading: React.Dispatch<React.SetStateAction<boolean>>;
	isFileConnectorLoading: boolean;
	setIsFileConnectorChanged: React.Dispatch<React.SetStateAction<boolean>>;
	isFileConnectorChanged: boolean;
	setIsTableChanged: React.Dispatch<React.SetStateAction<boolean>>;
	isSaveHidden: boolean;
	setIsSaveHidden: React.Dispatch<React.SetStateAction<boolean>>;
}

const actionContext = createContext<ActionContext>({} as ActionContext);

export default actionContext;
