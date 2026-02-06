import React, { useState, useContext, useEffect, useLayoutEffect, ReactNode } from 'react';
import './ConnectionsMap.css';
import Arrow from '../../base/Arrow/Arrow';
import { getConnectionsBlocks } from './ConnectionsMap.helpers';
import AppContext from '../../../context/AppContext';
import ConnectionMapContext from '../../../context/ConnectionMapContext';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';

const ConnectionsMap = () => {
	const [databaseArrows, setDatabaseArrows] = useState<ReactNode>([]);
	const [hoveredConnection, setHoveredConnection] = useState<number | null>(null);
	const [isUserDbHovered, setIsUserDbHovered] = useState<boolean>(false);
	const [isEventDbHovered, setIsEventDbHovered] = useState<boolean>(false);

	const { connections, setTitle, workspaces, selectedWorkspace } = useContext(AppContext);

	useLayoutEffect(() => {
		setTitle('Connections');
	}, []);

	useEffect(() => {
		// Must wait for the map to be painted and styled before proceding with
		// the render of the database's arrow.
		let hovered: TransformedConnection = null;
		if (hoveredConnection != null) {
			hovered = connections.find((c) => c.id === hoveredConnection);
		}

		let isImportUserDbConnectedToHover = false;
		let isImportUserDbHighlighted = false;
		if (hovered != null && hovered.isSource) {
			isImportUserDbConnectedToHover = hovered.pipelines.findIndex((p) => p.target === 'User') != -1;
			isImportUserDbHighlighted = hovered.relations(connections).includes('dwh-user');
		} else if (isUserDbHovered) {
			isImportUserDbConnectedToHover =
				connections.findIndex((c) => c.isSource && c.pipelines.findIndex((p) => p.target === 'User') != -1) !=
				-1;
			isImportUserDbHighlighted =
				connections.findIndex((c) => c.isSource && c.relations(connections).includes('dwh-user')) != -1;
		}

		let isExportUserDbConnectedToHover = false;
		let isExportUserDbHighlighted = false;
		if (hovered != null && hovered.isDestination) {
			isExportUserDbConnectedToHover = hovered.pipelines.findIndex((p) => p.target === 'User') != -1;
			isExportUserDbHighlighted = hovered.relations(connections).includes('dwh-user');
		} else if (isUserDbHovered) {
			isExportUserDbConnectedToHover =
				connections.findIndex(
					(c) => c.isDestination && c.pipelines.findIndex((p) => p.target === 'User') != -1,
				) != -1;
			isExportUserDbHighlighted =
				connections.findIndex((c) => c.isDestination && c.relations(connections).includes('dwh-user')) != -1;
		}

		let isEventDbConnectedToHover = false;
		let isEventDbHighlighted = false;
		if (hovered != null && hovered.isSource) {
			isEventDbConnectedToHover = hovered.pipelines.findIndex((p) => p.target === 'Event') != -1;
			isEventDbHighlighted = hovered.relations(connections).includes('dwh-event');
		} else if (isEventDbHovered) {
			isEventDbConnectedToHover =
				connections.findIndex((c) => c.isSource && c.pipelines.findIndex((p) => p.target === 'Event') != -1) !=
				-1;
			isEventDbHighlighted =
				connections.findIndex((c) => c.isSource && c.relations(connections).includes('dwh-event')) != -1;
		}

		const isSomethingHovered = hoveredConnection != null || isUserDbHovered || isEventDbHovered;

		setTimeout(() => {
			setDatabaseArrows(
				<>
					<Arrow
						start='central-logo'
						end='users-database'
						startAnchor='bottom'
						endAnchor='top'
						color={isImportUserDbHighlighted ? '#4f46e5' : undefined}
						strokeWidth={1}
						dashness={
							isImportUserDbHighlighted
								? { strokeLen: 5, nonStrokeLen: 5, animation: hovered?.isDestination ? -2 : 2 }
								: false
						}
						isHidden={isSomethingHovered && !isImportUserDbConnectedToHover}
					/>
					<Arrow
						start='users-database'
						end='central-logo'
						startAnchor='right'
						endAnchor='bottom'
						color={isExportUserDbHighlighted ? '#4f46e5' : undefined}
						strokeWidth={1}
						dashness={
							isExportUserDbHighlighted
								? { strokeLen: 5, nonStrokeLen: 5, animation: hovered?.isSource ? -2 : 2 }
								: false
						}
						isHidden={isSomethingHovered && !isExportUserDbConnectedToHover}
					/>
					<Arrow
						start='central-logo'
						end='events-database'
						startAnchor='bottom'
						endAnchor='top'
						color={isEventDbHighlighted ? '#4f46e5' : undefined}
						strokeWidth={1}
						dashness={
							isEventDbHighlighted
								? { strokeLen: 5, nonStrokeLen: 5, animation: hovered?.isDestination ? -2 : 2 }
								: false
						}
						isHidden={isSomethingHovered && !isEventDbConnectedToHover}
					/>
				</>,
			);
		}, 0);
	}, [hoveredConnection, isUserDbHovered, isEventDbHovered]);

	const onUserDbMouseEnter = () => {
		setIsUserDbHovered(true);
	};

	const onUserDbMouseLeave = () => {
		setIsUserDbHovered(false);
	};

	const onEventDbMouseEnter = () => {
		setIsEventDbHovered(true);
	};

	const onEventDbMouseLeave = () => {
		setIsEventDbHovered(false);
	};

	const newConnectionID = Number(new URL(document.location.href).searchParams.get('newConnection'));
	const sources: TransformedConnection[] = [];
	const destinations: TransformedConnection[] = [];
	connections.sort((a, b) => {
		if (a.name < b.name) {
			return -1;
		} else if (a.name > b.name) {
			return 1;
		} else {
			// The names are equal, compare the IDs.
			return a.id < b.id ? -1 : 1;
		}
	});
	for (const c of connections) {
		if (c.role === 'Source') sources.push(c);
		if (c.role === 'Destination') destinations.push(c);
	}
	const sourcesBlocks = getConnectionsBlocks(sources, newConnectionID);
	const destinationsBlocks = getConnectionsBlocks(destinations, newConnectionID);

	const warehouseMode = workspaces.find((w) => w.id === selectedWorkspace).warehouseMode;

	return (
		<ConnectionMapContext.Provider
			value={{ hoveredConnection, setHoveredConnection, isEventDbHovered, isUserDbHovered }}
		>
			<div className='connections-map'>
				<div className='route-content'>
					<div className='connections-map__content'>
						<div className='connections-map__buttons'>
							<Link path={`connectors?role=Source`}>
								<SlButton className='connections-map__add-source' variant='text'>
									<SlIcon slot='suffix' name='plus-circle' />
									Add a new source
								</SlButton>
							</Link>
							<Link path={`connectors?role=Destination`}>
								<SlButton className='connections-map__add-destination' variant='text'>
									<SlIcon slot='suffix' name='plus-circle' />
									Add a new destination
								</SlButton>
							</Link>
						</div>
						<div className='connections-map__map'>
							<div
								className={`connections-map__sources${sourcesBlocks.length === 0 ? ' connections-map__sources--no-connection' : ''}`}
							>
								{sourcesBlocks}
							</div>
							<div className='connections-map__main'>
								<div className='connections-map__central-logo' id='central-logo'>
									<img
										alt='Meergo icon logo'
										src='data:image/png;base64,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'
									/>
								</div>
								<div className='connections-map__databases'>
									<Link path='profile-unification/profiles'>
										<div
											className='connections-map__database connections-map__database--users'
											id='users-database'
											onMouseEnter={onUserDbMouseEnter}
											onMouseLeave={onUserDbMouseLeave}
										>
											{warehouseMode === 'Normal' ? (
												<SlTooltip content='The warehouse is in Normal mode (full read and write access)'>
													<SlIcon name='database' />
												</SlTooltip>
											) : warehouseMode === 'Inspection' ? (
												<SlTooltip content='The warehouse is in Inspection mode (read-only for data inspection)'>
													<SlIcon name='database-lock' />
												</SlTooltip>
											) : (
												<SlTooltip
													content='The warehouse is in Maintenance mode (init and alter schema
											operations only)'
												>
													<SlIcon name='database-gear' />
												</SlTooltip>
											)}
											<div className='connections-map__database-name'>User Profiles</div>
										</div>
									</Link>
									<div
										className='connections-map__database connections-map__database--events'
										id='events-database'
										onMouseEnter={onEventDbMouseEnter}
										onMouseLeave={onEventDbMouseLeave}
									>
										<SlIcon name='database' />
										<div className='connections-map__database-name'>Events</div>
									</div>
								</div>
								{databaseArrows}
							</div>
							<div
								className={`connections-map__destinations${destinationsBlocks.length === 0 ? ' connections-map__destinations--no-connection' : ''}`}
							>
								{destinationsBlocks}
							</div>
						</div>
					</div>
				</div>
			</div>
		</ConnectionMapContext.Provider>
	);
};

export default ConnectionsMap;
