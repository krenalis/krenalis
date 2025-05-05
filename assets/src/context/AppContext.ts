import React, { createContext, ReactNode } from 'react';
import API from '../lib/api/api';
import TransformedConnector from '../lib/core/connector';
import TransformedConnection from '../lib/core/connection';
import { Status } from '../components/routes/App/App.types';
import { Warehouse } from '../components/routes/App/App.types';
import Workspace from '../lib/api/types/workspace';
import { TransformedMember } from '../lib/core/member';
import { SlAlert } from '@shoelace-style/shoelace';
import { FeedbackButtonRef } from '../components/base/FeedbackButton/FeedbackButton';

interface AppContext {
	api: API;
	handleError: (err: Error | string) => void;
	showStatus: (status: Status) => void;
	showNotFound: () => void;
	setTitle: React.Dispatch<React.SetStateAction<ReactNode>>;
	redirect: (url: string) => void;
	member: TransformedMember;
	setIsLoadingMember: React.Dispatch<React.SetStateAction<boolean>>;
	connectors: TransformedConnector[];
	connections: TransformedConnection[];
	setIsLoadingConnections: React.Dispatch<React.SetStateAction<boolean>>;
	workspaces: Workspace[];
	setIsLoadingWorkspaces: React.Dispatch<React.SetStateAction<boolean>>;
	warehouse: Warehouse;
	selectedWorkspace: number;
	setSelectedWorkspace: React.Dispatch<React.SetStateAction<number>>;
	setIsLoadingState: React.Dispatch<React.SetStateAction<boolean>>;
	isFullscreen: boolean;
	title: ReactNode;
	logout: () => void;
	setIsLoggedIn: React.Dispatch<React.SetStateAction<boolean>>;
	toastRef: React.MutableRefObject<SlAlert>;
	executeAction: (connection: TransformedConnection, actionID: number) => Promise<void>;
	executeActionButtonRefs: React.MutableRefObject<{
		[key: number]: React.RefObject<FeedbackButtonRef>;
	}>;
	isPasswordless: boolean;
	setIsPasswordless: React.Dispatch<React.SetStateAction<boolean>>;
}

const appContext = createContext<AppContext>({} as AppContext);

export default appContext;
