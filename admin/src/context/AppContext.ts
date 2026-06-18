import React, { createContext, ReactNode } from 'react';
import API from '../lib/api/api';
import TransformedConnector from '../lib/core/connector';
import TransformedConnection from '../lib/core/connection';
import { Status } from '../components/routes/App/App.types';
import { Warehouse } from '../components/routes/App/App.types';
import Workspace from '../lib/api/types/workspace';
import type SlAlert from '@shoelace-style/shoelace/dist/components/alert/alert.js';
import { FeedbackButtonRef } from '../components/base/FeedbackButton/FeedbackButton';
import { Member, PublicMetadata } from '../lib/api/types/responses';
import { PipelineTarget } from '../lib/api/types/pipeline';

interface AppContext {
	api: API;
	handleError: (err: Error | string) => void;
	showStatus: (status: Status) => void;
	showNotFound: () => void;
	setTitle: React.Dispatch<React.SetStateAction<ReactNode>>;
	redirect: (url: string) => void;
	member: Member;
	setIsLoadingMember: React.Dispatch<React.SetStateAction<boolean>>;
	connectors: TransformedConnector[];
	connections: TransformedConnection[];
	setConnections: React.Dispatch<React.SetStateAction<TransformedConnection[]>>;
	setIsLoadingConnections: React.Dispatch<React.SetStateAction<boolean>>;
	workspaces: Workspace[];
	setIsLoadingWorkspaces: React.Dispatch<React.SetStateAction<boolean>>;
	warehouse: Warehouse;
	selectedWorkspace: string;
	setSelectedWorkspace: React.Dispatch<React.SetStateAction<string>>;
	setIsLoadingState: React.Dispatch<React.SetStateAction<boolean>>;
	isFullscreen: boolean;
	title: ReactNode;
	logout: () => Promise<void>;
	setIsLoggedIn: React.Dispatch<React.SetStateAction<boolean>>;
	toastRef: React.MutableRefObject<SlAlert>;
	runPipeline: (
		connection: TransformedConnection,
		pipelineID: string,
		pipelineTarget: PipelineTarget,
	) => Promise<void>;
	runPipelineButtonRefs: React.MutableRefObject<{
		[key: string]: React.RefObject<FeedbackButtonRef>;
	}>;
	runPipelineDropdownButtonRefs: React.MutableRefObject<{
		[key: string]: React.RefObject<FeedbackButtonRef>;
	}>;
	isPasswordless: boolean;
	setIsPasswordless: React.Dispatch<React.SetStateAction<boolean>>;
	publicMetadata: PublicMetadata;
	isOrganizationDisabled: boolean;
	setIsOrganizationDisabled: React.Dispatch<React.SetStateAction<boolean>>;
}

const appContext = createContext<AppContext>({} as AppContext);

export default appContext;
