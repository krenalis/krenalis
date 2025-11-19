import React, { useState, useEffect, ReactNode, useContext } from 'react';
import Flex from '../../base/Flex/Flex';
import Arrow from '../../base/Arrow/Arrow';
import StatusDot from '../../base/StatusDot/StatusDot';
import { ArrowAnchor } from '../../base/Arrow/Arrow.types';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';
import connectionMapContext from '../../../context/ConnectionMapContext';
import appContext from '../../../context/AppContext';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

interface ConnectionBlockProps {
	connection: TransformedConnection;
	isNew: boolean;
}

const ConnectionBlock = ({ connection: c, isNew }: ConnectionBlockProps) => {
	const [arrow, setArrow] = useState<ReactNode>();

	const { connections } = useContext(appContext);
	const { hoveredConnection, setHoveredConnection, isUserDbHovered, isEventDbHovered } =
		useContext(connectionMapContext);

	useEffect(() => {
		let arrowStart: string,
			arrowEnd: string,
			arrowStartAnchor: ArrowAnchor,
			arrowEndAnchor: ArrowAnchor,
			showTail: boolean = false,
			showHead: boolean = false;
		if (c.isSource) {
			arrowStart = `${c.id}`;
			arrowEnd = 'central-logo';
			arrowStartAnchor = 'right';
			arrowEndAnchor = 'left';
			showTail = true;
		} else {
			arrowStart = 'central-logo';
			arrowEnd = `${c.id}`;
			arrowStartAnchor = 'right';
			arrowEndAnchor = 'left';
			showHead = true;
		}

		const isConnected = c.actionsCount > 0 || c.linkedConnections?.length > 0;
		const hasRelations = c.relations(connections).length > 0;

		const isHovered =
			c.id === hoveredConnection ||
			c.relations(connections).includes(hoveredConnection) ||
			(isUserDbHovered && c.relations(connections).includes('dwh-user')) ||
			(isEventDbHovered && c.relations(connections).includes('dwh-event'));

		const isHighlighted = isHovered && hasRelations;

		const isSomethingHovered = hoveredConnection != null || isUserDbHovered || isEventDbHovered;
		const isHidden =
			!isConnected ||
			(isSomethingHovered &&
				!(isHovered && isConnected) &&
				!c.linkedConnections?.includes(hoveredConnection) &&
				!(isUserDbHovered && c.actionsInfo.findIndex((a) => a.target === 'User') != -1) &&
				!(isEventDbHovered && c.isSource && c.actionsInfo.findIndex((a) => a.target === 'Event') != -1));

		const arrow = (
			<Arrow
				start={arrowStart}
				end={arrowEnd}
				startAnchor={arrowStartAnchor}
				endAnchor={arrowEndAnchor}
				color={isHighlighted ? '#4f46e5' : undefined}
				strokeWidth={1}
				dashness={isHighlighted ? { strokeLen: 5, nonStrokeLen: 5, animation: c.isSource ? 2 : -2 } : false}
				isNew={isNew}
				isHidden={isHidden}
				showTail={showTail && isConnected}
				showHead={showHead && isConnected}
				useCircleShape={true}
			/>
		);

		// Must wait for the block to be painted and styled before proceding
		// with the render of the arrow.
		setTimeout(() => {
			setArrow(arrow);
		}, 0);
	}, [c, hoveredConnection, isUserDbHovered, isEventDbHovered]);

	const onMouseEnter = () => {
		setHoveredConnection(c.id);
	};

	const onMouseLeave = () => {
		setHoveredConnection(null);
	};

	return (
		<>
			<Link path={`connections/${c.id}/actions`}>
				<div
					className={`connection-block${isNew ? ' connection-block--new' : ''}`}
					id={`${c.id}`}
					onMouseEnter={onMouseEnter}
					onMouseLeave={onMouseLeave}
					data-is-hovered={c.id === hoveredConnection}
				>
					<div className='connection-block__content'>
						<Flex alignItems='center' gap={10}>
							<LittleLogo code={c.connector.code} path={CONNECTORS_ASSETS_PATH} />
							<div className='connection-block__name'>{c.name}</div>
						</Flex>
						<StatusDot status={c.status} />
					</div>
				</div>
			</Link>
			{arrow}
		</>
	);
};

export default ConnectionBlock;
