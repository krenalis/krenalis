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
			isImportUserDbConnectedToHover = hovered.pipelinesInfo.findIndex((p) => p.target === 'User') != -1;
			isImportUserDbHighlighted = hovered.relations(connections).includes('dwh-user');
		} else if (isUserDbHovered) {
			isImportUserDbConnectedToHover =
				connections.findIndex(
					(c) => c.isSource && c.pipelinesInfo.findIndex((p) => p.target === 'User') != -1,
				) != -1;
			isImportUserDbHighlighted =
				connections.findIndex((c) => c.isSource && c.relations(connections).includes('dwh-user')) != -1;
		}

		let isExportUserDbConnectedToHover = false;
		let isExportUserDbHighlighted = false;
		if (hovered != null && hovered.isDestination) {
			isExportUserDbConnectedToHover = hovered.pipelinesInfo.findIndex((p) => p.target === 'User') != -1;
			isExportUserDbHighlighted = hovered.relations(connections).includes('dwh-user');
		} else if (isUserDbHovered) {
			isExportUserDbConnectedToHover =
				connections.findIndex(
					(c) => c.isDestination && c.pipelinesInfo.findIndex((p) => p.target === 'User') != -1,
				) != -1;
			isExportUserDbHighlighted =
				connections.findIndex((c) => c.isDestination && c.relations(connections).includes('dwh-user')) != -1;
		}

		let isEventDbConnectedToHover = false;
		let isEventDbHighlighted = false;
		if (hovered != null && hovered.isSource) {
			isEventDbConnectedToHover = hovered.pipelinesInfo.findIndex((p) => p.target === 'Event') != -1;
			isEventDbHighlighted = hovered.relations(connections).includes('dwh-event');
		} else if (isEventDbHovered) {
			isEventDbConnectedToHover =
				connections.findIndex(
					(c) => c.isSource && c.pipelinesInfo.findIndex((p) => p.target === 'Event') != -1,
				) != -1;
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
									meergo
								</div>
								<div className='connections-map__databases'>
									<Link path='users'>
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
