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
import { SlSpinner } from '@shoelace-style/shoelace/dist/react/index.js';
import API from '../../lib/api/api';
import { Connection } from '../../types/external/connection';

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
	const [areConnectionsStale, setAreConnectionsStale] = useState<boolean>(false);

	const isLoadingTimeoutID = useRef<number>(0);

	useEffect(() => {
		isLoadingTimeoutID.current = setTimeout(() => setIsLoading(true), 100);
	}, []);

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
						c.DestinationDescription
					)
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
				connections = await api.connections.find();
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
					getConnectionDescription(c, connector)
				);
				if (transformedConnection.isStorage) {
					transformedConnection.linkedFiles = getStorageFileConnections(
						transformedConnection.id,
						connections
					);
				}
				transformedConnections.push(transformedConnection);
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

	if (connectors == null || connections == null) {
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
				setAreConnectionsStale,
			}}
		>
			{children}
		</AppContext.Provider>
	);
};

export { AppProvider, AppContext };
