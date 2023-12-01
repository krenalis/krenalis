import { useState, useEffect } from 'react';
import API from '../lib/api/api';
import TransformedConnector from '../lib/helpers/transformedConnector';
import { Connector } from '../types/external/connector';
import TransformedConnection, {
	getConnectionFullConnector,
	getConnectionStatus,
	getConnectionDescription,
	getStorageFileConnections,
} from '../lib/helpers/transformedConnection';
import { Location } from 'react-router-dom';
import { adminBasePath } from '../constants/path';
import { Connection } from '../types/external/connection';
import Workspace from '../types/external/workspace';
import { Warehouse } from '../types/internal/app';
import { WarehouseResponse } from '../types/external/warehouse';
import { Member } from '../types/external/api';
import { NotFoundError } from '../lib/api/errors';
import { TransformedMember, transformMember } from '../lib/helpers/transformedMember';

const useApp = (showError: (err: Error | string) => void, redirect: (url: string) => void, location: Location) => {
	const [isLoadingState, setIsLoadingState] = useState<boolean>(true);
	const [isLoggedIn, setIsLoggedIn] = useState<boolean>(false);
	const [member, setMember] = useState<TransformedMember | null>();
	const [isLoadingMember, setIsLoadingMember] = useState<boolean>(false);
	const [connectors, setConnectors] = useState<TransformedConnector[] | null>(null);
	const [connections, setConnections] = useState<TransformedConnection[] | null>(null);
	const [isLoadingConnections, setIsLoadingConnections] = useState<boolean>(false);
	const [warehouse, setWarehouse] = useState<Warehouse | null>(null);
	const [workspaces, setWorkspaces] = useState<Workspace[] | null>(null);
	const [isLoadingWorkspaces, setIsLoadingWorkspaces] = useState<boolean>(false);
	const [selectedWorkspace, setSelectedWorkspace] = useState<number>(
		Number(localStorage.getItem('chichi_workspace_id')),
	);

	const api = new API(window.location.origin, selectedWorkspace);

	useEffect(() => {
		const loadAppState = async () => {
			// get the workspaces list.
			let ws: Workspace[];
			try {
				ws = await api.workspaces.list();
			} catch (err) {
				showError(err);
				return;
			}
			setWorkspaces(ws);

			const isDeleted = workspaces != null && ws.length < workspaces.length;
			if (selectedWorkspace === 0 && ws.length === 1 && !isDeleted) {
				setSelectedWorkspace(ws[0].ID);
			}

			// get the connectors list.
			let connectors: Connector[];
			try {
				connectors = await api.connectors.find();
			} catch (err) {
				showError(err);
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
						c.SampleQuery,
						c.WebhooksPer,
						c.OAuth,
						c.SourceDescription,
						c.DestinationDescription,
						c.TermForUsers,
						c.TermForGroups,
					),
				);
			}
			setConnectors(transformedConnectors);

			const memberID = sessionStorage.getItem('chichi-member');
			if (memberID == null) {
				setTimeout(() => {
					setIsLoggedIn(false);
					setIsLoadingState(false);
					if (location.pathname !== adminBasePath) {
						redirect('');
					}
				}, 300);
				return;
			}
			setIsLoggedIn(true);

			let member: Member;
			try {
				member = await api.member(Number(memberID));
			} catch (err) {
				if (err instanceof NotFoundError) {
					showError('The current logged in member does not exist anymore');
					try {
						await api.logout();
					} catch (err) {
						showError(err);
						return;
					}
					setTimeout(() => {
						sessionStorage.removeItem('chichi-member');
						setSelectedWorkspace(0);
						setIsLoggedIn(false);
						setIsLoadingState(false);
						redirect('');
					}, 300);
					return;
				}
				showError(err);
				return;
			}
			setMember(transformMember(member));

			if (selectedWorkspace === 0) {
				// the user must choose a workspace.
				setTimeout(() => {
					setIsLoadingState(false);
				}, 300);
				return;
			}

			// if the user is logged in and has a selected workspace, but they
			// are currently at the base path, redirect to the connections map
			// path.
			let isBasePath = location.pathname === adminBasePath;
			if (isBasePath) {
				redirect('connections');
			}

			// get the connections.
			let connections: Connection[];
			try {
				connections = await api.workspaces.connections.find();
			} catch (err) {
				showError(err);
				return;
			}
			const transformedConnections: TransformedConnection[] = [];
			for (const c of connections) {
				const connector = getConnectionFullConnector(c.Connector, transformedConnectors!);
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
					c.Compression,
					c.WebsiteHost,
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

			// get the warehouse.
			let warehouseResponse: WarehouseResponse;
			try {
				warehouseResponse = await api.workspaces.warehouseSettings();
			} catch (err) {
				setTimeout(() => setIsLoadingState(false), 300);
				setWarehouse(null);
				if (err.code === 'NotConnected') {
					return;
				}
				showError(err);
				return;
			}
			setWarehouse({
				type: warehouseResponse.type,
				settings: warehouseResponse.settings,
			});

			setTimeout(() => setIsLoadingState(false), 300);
		};

		if (!isLoadingState) {
			return;
		}

		loadAppState();
	}, [isLoadingState]);

	useEffect(() => {
		const loadConnection = async () => {
			// get the connections.
			let connections: Connection[];
			try {
				connections = await api.workspaces.connections.find();
			} catch (err) {
				showError(err);
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
					c.Compression,
					c.WebsiteHost,
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
		};

		if (isLoadingState || !isLoadingConnections) {
			return;
		}

		loadConnection();
		setIsLoadingConnections(false);
	}, [isLoadingConnections]);

	useEffect(() => {
		const loadWorkspaces = async () => {
			let ws: Workspace[];
			try {
				ws = await api.workspaces.list();
			} catch (err) {
				showError(err);
				return;
			}
			setWorkspaces(ws);
		};

		if (isLoadingState || !isLoadingWorkspaces) {
			return;
		}

		loadWorkspaces();
		setIsLoadingWorkspaces(false);
	}, [isLoadingWorkspaces]);

	useEffect(() => {
		const loadMember = async () => {
			let m: Member;
			try {
				m = await api.member(member.ID);
			} catch (err) {
				if (err instanceof NotFoundError) {
					showError('The current logged in member does not exist anymore');
					try {
						await api.logout();
					} catch (err) {
						showError(err);
						return;
					}
					sessionStorage.removeItem('chichi-member');
					setSelectedWorkspace(0);
					setIsLoggedIn(false);
					setIsLoadingMember(false);
					redirect('');
					return;
				}
				showError(err);
				return;
			}
			setMember(transformMember(m));
		};

		if (isLoadingState || !isLoadingMember) {
			return;
		}

		loadMember();
		setIsLoadingMember(false);
	}, [isLoadingMember]);

	useEffect(() => {
		if (selectedWorkspace === 0) {
			localStorage.removeItem('chichi_workspace_id');
		} else {
			localStorage.setItem('chichi_workspace_id', String(selectedWorkspace));
		}
	}, [selectedWorkspace]);

	return {
		isLoadingState,
		setIsLoadingState,
		isLoggedIn,
		setIsLoggedIn,
		member,
		setIsLoadingMember,
		connectors,
		connections,
		setIsLoadingConnections,
		warehouse,
		workspaces,
		setIsLoadingWorkspaces,
		selectedWorkspace,
		setSelectedWorkspace,
		api,
	};
};

export { useApp };
