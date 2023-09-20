import React, { useEffect, useState, useRef, ReactNode } from 'react';
import { Connector } from '../../types/external/connector';
import TransformedConnection, {
	getConnectionFullConnector,
	getConnectionStatus,
	getConnectionDescription,
	getStorageFileConnections,
} from '../../lib/helpers/transformedConnection';
import TransformedConnector from '../../lib/helpers/transformedConnector';
import AppContext from '../AppContext';
import { Status } from '../../types/internal/app';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import API from '../../lib/api/api';
import { Connection } from '../../types/external/connection';
import Workspace from '../../types/external/workspace';

interface AppProviderProps {
	api: API;
	showError: (err: Error | string) => void;
	showStatus: (status: Status) => void;
	showNotFound: () => void;
	setTitle: React.Dispatch<React.SetStateAction<ReactNode>>;
	redirect: (url: string) => void;
	account: number | null;
	children: ReactNode;
}

const AppProvider = ({
	api,
	showError,
	showStatus,
	showNotFound,
	setTitle,
	redirect,
	account,
	children,
}: AppProviderProps) => {
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [connectors, setConnectors] = useState<TransformedConnector[] | null>(null);
	const [connections, setConnections] = useState<TransformedConnection[] | null>(null);
	const [workspace, setWorkspace] = useState<Workspace | null>(null);
	const [areConnectionsStale, setAreConnectionsStale] = useState<boolean>(false);
	const [isWorkspaceStale, setIsWorkspaceStale] = useState<boolean>(true);

	const isLoadingTimeoutID = useRef<number>(0);

	useEffect(() => {
		isLoadingTimeoutID.current = window.setTimeout(() => setIsLoading(true), 100);
	}, []);

	useEffect(() => {
		const fetchWorkspace = async () => {
			let workspace: Workspace;
			try {
				workspace = await api.workspace.get();
			} catch (err) {
				showError(err);
				clearTimeout(isLoadingTimeoutID.current);
				return;
			}
			setWorkspace(workspace);
			setIsWorkspaceStale(false);
		};
		if (isWorkspaceStale) {
			fetchWorkspace();
		}
	}, [isWorkspaceStale]);

	useEffect(() => {
		const fetchConnectors = async () => {
			let connectors: Connector[];
			try {
				connectors = await api.connectors.find();
			} catch (err) {
				showError(err);
				clearTimeout(isLoadingTimeoutID.current);
				return;
			}
			const transformedConnectors: TransformedConnector[] = [];
			for (const c of connectors) {
				transformedConnectors.push(
					new TransformedConnector(
						c.ID,
						c.Name,
						c.Type,
						c.HasSheets,
						c.HasSettings,
						c.Icon,
						c.FileExtension,
						c.WebhooksPer,
						c.OAuth,
						c.SourceDescription,
						c.DestinationDescription,
					),
				);
			}
			setConnectors(transformedConnectors);
			setAreConnectionsStale(true);
		};
		fetchConnectors();
	}, []);

	useEffect(() => {
		const fetchConnections = async () => {
			let connections: Connection[];
			try {
				connections = await api.workspace.connections.find();
			} catch (err) {
				setConnections([]);
				showError(err);
				clearTimeout(isLoadingTimeoutID.current);
				return;
			}
			const transformedConnections: TransformedConnection[] = [];
			for (const c of connections) {
				const connector = getConnectionFullConnector(c.Connector, connectors!);
				const transformedConnection = new TransformedConnection(
					c.ID,
					c.Name,
					c.Type,
					c.Role,
					connector,
					c.HasSettings,
					c.Enabled,
					c.ActionsCount,
					c.Health,
					c.Storage,
					getConnectionStatus(c),
					getConnectionDescription(c, connector),
				);
				transformedConnections.push(transformedConnection);
			}
			for (const c of transformedConnections) {
				if (c.isStorage) {
					c.linkedFiles = getStorageFileConnections(c.id, transformedConnections);
				}
			}
			setConnections(transformedConnections);
			setAreConnectionsStale(false);
		};
		if (areConnectionsStale) {
			fetchConnections();
		}
	}, [areConnectionsStale]);

	useEffect(() => {
		if (connectors == null || connections == null) {
			return;
		}
		if (isLoading) {
			setTimeout(() => setIsLoading(false), 300);
		} else {
			clearTimeout(isLoadingTimeoutID.current);
		}
	}, [connectors, connections]);

	if (isLoading) {
		return (
			<SlSpinner
				className='globalSpinner'
				style={
					{
						fontSize: '5rem',
						'--track-width': '6px',
					} as React.CSSProperties
				}
			/>
		);
	}

	if (connectors == null || connections == null || workspace == null) {
		return null;
	}

	return (
		<AppContext.Provider
			value={{
				api,
				showError,
				showStatus,
				showNotFound,
				setTitle,
				redirect,
				account,
				connectors,
				connections,
				workspace,
				setAreConnectionsStale,
				setIsWorkspaceStale,
			}}
		>
			{children}
		</AppContext.Provider>
	);
};

export { AppProvider, AppContext };
