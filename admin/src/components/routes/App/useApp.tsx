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
import { RESET_PASSWORD_PATH, UI_BASE_PATH } from '../../../constants/paths';
import { Connection } from '../../../lib/api/types/connection';
import Workspace from '../../../lib/api/types/workspace';
import { Warehouse } from './App.types';
import { WarehouseResponse } from '../../../lib/api/types/warehouse';
import { Execution, Member, PublicMetadata } from '../../../lib/api/types/responses';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { FeedbackButtonRef } from '../../base/FeedbackButton/FeedbackButton';
import { sleep } from '../../../utils/sleep';
import { Link } from '../../base/Link/Link';
import { hasFilters } from '../../../lib/core/pipeline';
import { formatNumber } from '../../../utils/formatNumber';
import * as Sentry from '@sentry/react';
import { scrubURL } from '../../../lib/telemetry/scrubURL';
import { PipelineTarget } from '../../../lib/api/types/pipeline';
import { IS_PASSWORDLESS_KEY, storageKeysToBeRemoved, WORKSPACE_ID_KEY } from '../../../constants/storage';

const FILTER_STEP = 2;

const useApp = (
	handleError: (err: Error | string) => void,
	redirect: (url: string) => void,
	logout: () => void,
	location: Location,
	setIsLoggedIn: React.Dispatch<React.SetStateAction<boolean>>,
) => {
	const [isLoadingState, setIsLoadingState] = useState<boolean>(true);
	const [member, setMember] = useState<Member | null>();
	const [isLoadingMember, setIsLoadingMember] = useState<boolean>(false);
	const [connectors, setConnectors] = useState<TransformedConnector[] | null>(null);
	const [connections, setConnections] = useState<TransformedConnection[] | null>(null);
	const [isLoadingConnections, setIsLoadingConnections] = useState<boolean>(false);
	const [warehouse, setWarehouse] = useState<Warehouse | null>(null);
	const [workspaces, setWorkspaces] = useState<Workspace[] | null>(null);
	const [isLoadingWorkspaces, setIsLoadingWorkspaces] = useState<boolean>(false);
	const [isPasswordless, setIsPasswordless] = useState<boolean>(localStorage.getItem(IS_PASSWORDLESS_KEY) != null);
	const [selectedWorkspace, setSelectedWorkspace] = useState<number>(Number(localStorage.getItem(WORKSPACE_ID_KEY)));
	const [publicMetadata, setPublicMetadata] = useState<PublicMetadata>();

	let api = new API(window.location.origin, selectedWorkspace);

	const executePipelineButtonRefs = useRef<{
		[key: number]: React.RefObject<FeedbackButtonRef>;
	}>({});

	const executePipelineDropdownButtonRefs = useRef<{
		[key: number]: React.RefObject<FeedbackButtonRef>;
	}>({});

	const feedbackObserverRef = useRef<MutationObserver | null>(null);

	useEffect(() => {
		const loadAppState = async () => {
			// Retrieve the public metadata.
			let publicMetadata: PublicMetadata;
			try {
				publicMetadata = await api.publicMetadata();
			} catch (err) {
				handleError(err);
				setIsLoadingState(false);
				return;
			}

			// Redirect the user to the external URL if it doesn't match the
			// current URL.
			{
				const c = new URL(window.location.href);
				const e = new URL(publicMetadata.externalURL);
				if (c.protocol !== e.protocol || c.host !== e.host || c.port !== e.port) {
					// Clear the browser storage.
					sessionStorage.clear();
					for (const key of storageKeysToBeRemoved) {
						localStorage.removeItem(key);
					}
					// Redirect to the external URL, restarting the application.
					c.protocol = e.protocol;
					c.host = e.host;
					c.port = e.port;
					window.location.replace(c.toString());
					return;
				}
			}

			setPublicMetadata(publicMetadata);

			if (publicMetadata.telemetryLevel == 'errors' || publicMetadata.telemetryLevel == 'all') {
				// Initialize the Sentry SDK.
				Sentry.init({
					dsn: 'https://4bc227ec8dc487e9bae1f3aea7f3ede1@o4509282180136960.ingest.de.sentry.io/4509292547211344',
					tunnel: '/api/v1/sentry/errors',
					// Setting this option to true will send default PII
					// data to Sentry. For example, automatic IP address
					// collection on events.
					// TODO: is it okay to set it to false? See https://github.com/meergo/meergo/issues/1517.
					sendDefaultPii: false,
					beforeSend: (event) => {
						const [scrubbedURL, extras] = scrubURL(event.request.url, false);
						event.request.url = scrubbedURL;
						event.extra = {
							...event.extra,
							...extras,
						};
						return event;
					},
					beforeBreadcrumb: (breadcrumb) => {
						if (breadcrumb.category === 'fetch') {
							const [scrubbedURL] = scrubURL(breadcrumb.data.url, true);
							breadcrumb.data.url = scrubbedURL;
						}
						return breadcrumb;
					},
				});
				// Add the installation ID as tag.
				Sentry.setTag('meergo_installation_id', publicMetadata.installationID);
			}

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

			// if the API call succeeds without errors, it confirms that the
			// user is logged in.
			setIsLoggedIn(true);

			const isDeleted = workspaces != null && ws.length < workspaces.length;
			if (selectedWorkspace === 0) {
				if (ws.length === 1 && !isDeleted) {
					// the user has only one workspace, so it can be
					// automatically selected.
					setSelectedWorkspace(ws[0].id);
					api = new API(window.location.origin, ws[0].id);
				} else {
					if (ws.length === 0) {
						// the user must create a workspace.
						redirect('workspaces/create');
					} else {
						// the user must choose a workspace.
						redirect('workspaces');
					}
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
						c.code,
						c.label,
						c.type,
						c.categories,
						c.asSource,
						c.asDestination,
						c.identityIDLabel,
						c.hasSheets,
						c.fileExtension,
						c.oauth,
						c.terms,
						c.strategies,
					),
				);
			}
			setConnectors(transformedConnectors);

			let member: Member;
			try {
				member = await api.member();
			} catch (err) {
				if (err instanceof NotFoundError) {
					handleError('The current logged in team member does not exist anymore');
					setTimeout(() => {
						logout();
						setIsLoadingState(false);
					}, 300);
					return;
				}
				handleError(err);
				return;
			}
			setMember(member);

			// if the user is logged in and has a selected workspace,
			// but they are currently on the login or reset password
			// routes, redirect to the connections route.
			let isOutside =
				location.pathname === UI_BASE_PATH ||
				location.pathname == UI_BASE_PATH.replace(/\/$/, '') ||
				location.pathname.startsWith(RESET_PASSWORD_PATH);
			if (isOutside) {
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
					localStorage.removeItem(WORKSPACE_ID_KEY);
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
					c.health,
					c.storage,
					c.compression,
					c.strategy,
					c.sendingMode,
					getConnectionStatus(c),
					getConnectionDescription(c, connector),
				);
				if (c.linkedConnections) {
					transformedConnection.linkedConnections = c.linkedConnections;
				}
				transformedConnection.pipelines = c.pipelines;
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

		return () => {
			feedbackObserverRef.current?.disconnect();
			feedbackObserverRef.current = null;
		};
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
					c.health,
					c.storage,
					c.compression,
					c.strategy,
					c.sendingMode,
					getConnectionStatus(c),
					getConnectionDescription(c, connector),
				);
				if (c.linkedConnections) {
					transformedConnection.linkedConnections = c.linkedConnections;
				}
				transformedConnection.pipelines = c.pipelines;
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
					handleError('The current logged in team member does not exist anymore');
					setIsLoadingMember(false);
					logout();
					return;
				}
				handleError(err);
				return;
			}
			setMember(m);
		};

		if (isLoadingState || !isLoadingMember) {
			return;
		}

		loadMember();
		setIsLoadingMember(false);
	}, [isLoadingMember]);

	useEffect(() => {
		if (selectedWorkspace === 0) {
			localStorage.removeItem(WORKSPACE_ID_KEY);
		} else {
			localStorage.setItem(WORKSPACE_ID_KEY, String(selectedWorkspace));
		}
	}, [selectedWorkspace]);

	const executePipeline = async (
		connection: TransformedConnection,
		pipelineID: number,
		pipelineTarget: PipelineTarget,
	) => {
		executePipelineButtonRefs.current[pipelineID]?.current?.load();
		executePipelineDropdownButtonRefs.current[pipelineID]?.current?.load();
		let executionID: number;
		try {
			executionID = await api.workspaces.connections.executePipeline(pipelineID);
		} catch (err) {
			if (err instanceof UnprocessableError) {
				executePipelineButtonRefs.current[pipelineID]?.current?.error(err.message);
				executePipelineDropdownButtonRefs.current[pipelineID]?.current?.error(err.message);
				return;
			}
			executePipelineButtonRefs.current[pipelineID]?.current?.stop();
			executePipelineDropdownButtonRefs.current[pipelineID]?.current?.error(err.message);
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

		let link = `connections/${connection.id}/metrics`;
		if (execution.error) {
			link += `?failed-execution-pipeline=${pipelineID}`;
		}
		link += `?target=${pipelineTarget === 'Event' ? 'event' : 'user'}`;
		const metricsLink = (
			<div className='connection-pipelines__link-to-metrics'>
				Go to{' '}
				<Link path={link}>
					<span className='connection-pipelines__link'>Metrics</span>
				</Link>{' '}
				for details.
			</div>
		);

		if (execution.error !== '') {
			executePipelineButtonRefs.current[pipelineID]?.current?.error(
				<>
					{execution.error}
					{metricsLink}
				</>,
			);
			executePipelineDropdownButtonRefs.current[pipelineID]?.current?.error(
				<>
					{execution.error}
					{metricsLink}
				</>,
			);
			return;
		}

		const passed = execution.passed[5];
		const failed = execution.failed.filter((_, i) => i !== FILTER_STEP).reduce((sum, n) => sum + n, 0);

		const pipeline = connection.pipelines.find((p) => p.id === pipelineID);

		let filteredItem: ReactNode;
		if (hasFilters(connection, pipeline.target)) {
			const filtered = execution.failed[FILTER_STEP];
			filteredItem = <li>{formatNumber(filtered)} filtered out</li>;
		}

		const user = connection.isSource ? 'identity' : 'profile';
		const users = connection.isSource ? 'identities' : 'profiles';
		const executed = connection.isSource ? 'imported' : 'exported';

		const infoMessage = (
			<div className='connection-pipelines__execution-info'>
				<div className='connection-pipelines__execution-info-title'>
					{connection.isSource ? 'Import' : 'Export'} completed
				</div>
				<ul>
					<li>
						{formatNumber(passed)} {passed === 1 ? user : users} {executed}
					</li>
					{filteredItem}
					<li>
						{failed === 0 ? 'No errors occurred' : `${formatNumber(failed)} not ${executed} due to errors`}
					</li>
				</ul>
				{metricsLink}
			</div>
		);
		executePipelineButtonRefs.current[pipelineID]?.current?.info(infoMessage);
		executePipelineDropdownButtonRefs.current[pipelineID]?.current?.info(infoMessage);
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
		executePipeline,
		executePipelineButtonRefs,
		executePipelineDropdownButtonRefs,
		isPasswordless,
		setIsPasswordless,
		publicMetadata,
	};
};

export { useApp };
