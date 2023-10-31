import { createContext } from 'react';
import TransformedConnection from '../lib/helpers/transformedConnection';
import { TransformedAction, TransformedActionType } from '../lib/helpers/transformedAction';

interface ActionContext {
	mode: string;
	setMode: React.Dispatch<React.SetStateAction<string>>;
	connection: TransformedConnection;
	action: TransformedAction;
	setAction: React.Dispatch<React.SetStateAction<TransformedAction | undefined>>;
	saveAction: () => Promise<void>;
	actionType: TransformedActionType;
	setActionType: React.Dispatch<React.SetStateAction<TransformedActionType | undefined>>;
	isEditing: boolean;
	isImport: boolean;
	isTransformationAllowed: boolean;
	onClose: () => void;
	mappingSectionRef: React.MutableRefObject<any>;
	isMappingSectionDisabled: boolean;
	disabledReason: string;
	isSaveButtonLoading: boolean;
	setIsQueryChanged: React.Dispatch<React.SetStateAction<boolean>>;
	setIsFileChanged: React.Dispatch<React.SetStateAction<boolean>>;
	setIsTableChanged: React.Dispatch<React.SetStateAction<boolean>>;
	isSaveHidden: boolean;
	setIsSaveHidden: React.Dispatch<React.SetStateAction<boolean>>;
}

const actionContext = createContext<ActionContext>({} as ActionContext);

export default actionContext;
