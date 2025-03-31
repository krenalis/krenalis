import React, { useState, useEffect, useRef, ReactNode } from 'react';
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
import { Execution, Member } from '../../../lib/api/types/responses';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { TransformedMember, transformMember } from '../../../lib/core/member';
import { FeedbackButtonRef } from '../../base/FeedbackButton/FeedbackButton';
import { sleep } from '../../../utils/sleep';
import { Link } from '../../base/Link/Link';
import { hasFilters } from '../../../lib/core/action';
import { formatNumber } from '../../../utils/formatNumber';

const FILTER_STEP = 2;

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

	const executeActionButtonRefs = useRef<{
		[key: number]: React.RefObject<FeedbackButtonRef>;
	}>({});

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
						c.terms,
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
			let isOnLogin = location.pathname === UI_BASE_PATH || location.pathname == UI_BASE_PATH.replace(/\/$/, '');
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

	const executeAction = async (connection: TransformedConnection, actionID: number) => {
		executeActionButtonRefs.current[actionID]?.current?.load();
		let executionID: number;
		try {
			executionID = await api.workspaces.connections.executeAction(actionID);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				executeActionButtonRefs.current[actionID]?.current?.error(err.message);
				return;
			}
			executeActionButtonRefs.current[actionID]?.current?.stop();
			handleError(err);
			return;
		}

		let execution: Execution | null = null;
		while (execution == null) {
			await sleep(500);
			try {
				execution = await api.workspaces.connections.execution(executionID);
			} catch (err) {
				handleError(err);
				return;
			}
			if (execution.endTime == null) {
				execution = null;
			}
		}

		let link = `connections/${connection.id}/overview`;
		if (execution.error) {
			link += `?failed-execution-action=${actionID}`;
		}
		const overviewLink = (
			<div className='connection-actions__link-to-overview'>
				Go to{' '}
				<Link path={link}>
					<span className='connection-actions__link'>Metrics</span>
				</Link>{' '}
				for details.
			</div>
		);

		if (execution.error !== '') {
			executeActionButtonRefs.current[actionID]?.current?.error(
				<>
					{execution.error}
					{overviewLink}
				</>,
			);
			return;
		}

		const passed = execution.passed[5];
		const failed = execution.failed.filter((_, i) => i !== FILTER_STEP).reduce((sum, n) => sum + n, 0);

		const action = connection.actions.find((a) => a.id === actionID);

		let filteredItem: ReactNode;
		if (hasFilters(connection, action.target)) {
			const filtered = execution.failed[FILTER_STEP];
			filteredItem = <li>{formatNumber(filtered)} filtered out</li>;
		}

		const infoMessage = (
			<div className='connection-actions__execution-info'>
				<div className='connection-actions__execution-info-title'>
					{connection.isSource ? 'Import' : 'Export'} completed
				</div>
				<ul>
					<li>
						{formatNumber(passed)} {passed === 1 ? 'user identity' : 'user identities'}{' '}
						{connection.isSource ? 'imported' : 'exported'}
					</li>
					{filteredItem}
					<li>
						{failed === 0
							? 'No errors occurred'
							: `${formatNumber(failed)} not ${connection.isSource ? 'imported' : 'exported'} due to errors`}
					</li>
				</ul>
				{overviewLink}
			</div>
		);
		executeActionButtonRefs.current[actionID]?.current?.info(infoMessage);
	};

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
		executeAction,
		executeActionButtonRefs,
	};
};

export { useApp };
