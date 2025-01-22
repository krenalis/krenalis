import { useState, useEffect } from 'react';
import API from '../../../lib/api/api';
import TransformedConnector from '../../../lib/core/connector';
import { Connector } from '../../../lib/api/types/connector';
import TransformedConnection, {
	getConnectionFullConnector,
	getConnectionStatus,
	getConnectionDescription,
	getFileStorageConnections,
} from '../../../lib/core/connection';
import { Location } from 'react-router-dom';
import { UI_BASE_PATH } from '../../../constants/paths';
import { Connection } from '../../../lib/api/types/connection';
import Workspace from '../../../lib/api/types/workspace';
import { Warehouse } from './App.types';
import { WarehouseResponse } from '../../../lib/api/types/warehouse';
import { Member } from '../../../lib/api/types/responses';
import { NotFoundError } from '../../../lib/api/errors';
import { TransformedMember, transformMember } from '../../../lib/core/member';

const useApp = (
	handleError: (err: Error | string) => void,
	redirect: (url: string) => void,
	logout: () => void,
	location: Location,
) => {
	const [isLoadingState, setIsLoadingState] = useState<boolean>(true);
	const [member, setMember] = useState<TransformedMember | null>();
	const [isLoadingMember, setIsLoadingMember] = useState<boolean>(false);
	const [connectors, setConnectors] = useState<TransformedConnector[] | null>(null);
	const [connections, setConnections] = useState<TransformedConnection[] | null>(null);
	const [isLoadingConnections, setIsLoadingConnections] = useState<boolean>(false);
	const [warehouse, setWarehouse] = useState<Warehouse | null>(null);
	const [workspaces, setWorkspaces] = useState<Workspace[] | null>(null);
	const [isLoadingWorkspaces, setIsLoadingWorkspaces] = useState<boolean>(false);
	const [selectedWorkspace, setSelectedWorkspace] = useState<number>(
		Number(localStorage.getItem('meergo_ui_workspace_id')),
	);

	let api = new API(window.location.origin, selectedWorkspace);

	useEffect(() => {
		const loadAppState = async () => {
			// get the workspaces list.
			let ws: Workspace[];
			try {
				ws = await api.workspaces.list();
			} catch (err) {
				handleError(err);
				setIsLoadingState(false);
				return;
			}
			setWorkspaces(ws);

			const isDeleted = workspaces != null && ws.length < workspaces.length;
			if (selectedWorkspace === 0) {
				if (ws.length === 1 && !isDeleted) {
					setSelectedWorkspace(ws[0].id);
					api = new API(window.location.origin, ws[0].id);
				} else {
					// the user must choose a workspace.
					redirect('workspaces');
					setIsLoadingState(false);
					return;
				}
			}

			// get the connectors list.
			let connectors: Connector[];
			try {
				connectors = await api.connectors.find();
			} catch (err) {
				handleError(err);
				return;
			}
			const transformedConnectors: TransformedConnector[] = [];
			for (const c of connectors) {
				transformedConnectors.push(
					new TransformedConnector(
						c.name,
						c.type,
						c.asSource,
						c.asDestination,
						c.identityIDLabel,
						c.hasSheets,
						c.fileExtension,
						c.requiresAuth,
						c.termForUsers,
						c.termForGroups,
						c.icon,
					),
				);
			}
			setConnectors(transformedConnectors);

			let member: Member;
			try {
				member = await api.member();
			} catch (err) {
				if (err instanceof NotFoundError) {
					handleError('The current logged in member does not exist anymore');
					setTimeout(() => {
						logout();
						setIsLoadingState(false);
					}, 300);
					return;
				}
				handleError(err);
				return;
			}
			setMember(transformMember(member));

			// if the user is logged in and has a selected workspace, but they
			// are currently on the login route, redirect to the connections map
			// path.
			let isOnLogin = location.pathname === UI_BASE_PATH;
			if (isOnLogin) {
				redirect('connections');
			}

			// get the connections.
			let connections: Connection[];
			try {
				connections = await api.workspaces.connections.find();
			} catch (err) {
				if (err instanceof NotFoundError) {
					// the workspace saved in the local storage doesn't exist
					// anymore.
					localStorage.removeItem('meergo_ui_workspace_id');
					redirect('workspaces');
					setIsLoadingState(false);
					return;
				}
				handleError(err);
				return;
			}
			const transformedConnections: TransformedConnection[] = [];
			for (const c of connections) {
				const connector = getConnectionFullConnector(c.connector, transformedConnectors!);
				const transformedConnection = new TransformedConnection(
					c.id,
					c.name,
					connector,
					c.role,
					c.actionsCount,
					c.health,
					c.storage,
					c.compression,
					c.strategy,
					c.websiteHost,
					c.sendingMode,
					getConnectionStatus(c),
					getConnectionDescription(c, connector),
				);
				if (c.linkedConnections) {
					transformedConnection.linkedConnections = c.linkedConnections;
				}
				transformedConnections.push(transformedConnection);
			}
			for (const c of transformedConnections) {
				if (c.isFileStorage) {
					c.linkedFiles = getFileStorageConnections(c.id, transformedConnections);
				}
			}
			// order the connections alphabetically.
			transformedConnections.sort((a, b) => (a.name < b.name ? -1 : 1));
			setConnections(transformedConnections);

			// get the warehouse.
			let warehouseResponse: WarehouseResponse;
			try {
				warehouseResponse = await api.workspaces.warehouse();
			} catch (err) {
				setTimeout(() => setIsLoadingState(false), 300);
				setWarehouse(null);
				handleError(err);
				return;
			}
			setWarehouse({
				name: warehouseResponse.name,
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
				handleError(err);
				return;
			}
			const transformedConnections: TransformedConnection[] = [];
			for (const c of connections) {
				const connector = getConnectionFullConnector(c.connector, connectors!);
				const transformedConnection = new TransformedConnection(
					c.id,
					c.name,
					connector,
					c.role,
					c.actionsCount,
					c.health,
					c.storage,
					c.compression,
					c.strategy,
					c.websiteHost,
					c.sendingMode,
					getConnectionStatus(c),
					getConnectionDescription(c, connector),
				);
				if (c.linkedConnections) {
					transformedConnection.linkedConnections = c.linkedConnections;
				}
				transformedConnections.push(transformedConnection);
			}
			for (const c of transformedConnections) {
				if (c.isFileStorage) {
					c.linkedFiles = getFileStorageConnections(c.id, transformedConnections);
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
				handleError(err);
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
				m = await api.member();
			} catch (err) {
				if (err instanceof NotFoundError) {
					handleError('The current logged in member does not exist anymore');
					setIsLoadingMember(false);
					logout();
					return;
				}
				handleError(err);
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
			localStorage.removeItem('meergo_ui_workspace_id');
		} else {
			localStorage.setItem('meergo_ui_workspace_id', String(selectedWorkspace));
		}
	}, [selectedWorkspace]);

	return {
		isLoadingState,
		setIsLoadingState,
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
