import React, { createContext, ReactNode } from 'react';
import API from '../lib/api/api';
import TransformedConnector from '../lib/helpers/transformedConnector';
import TransformedConnection from '../lib/helpers/transformedConnection';
import { Status, Warehouse } from '../types/internal/app';
import Workspace from '../types/external/workspace';
import { TransformedMember } from '../lib/helpers/transformedMember';
import { SlAlert } from '@shoelace-style/shoelace';

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
}

const appContext = createContext<AppContext>({} as AppContext);

export default appContext;
