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

interface ConnectionBlockProps {
	connection: TransformedConnection;
	isNew: boolean;
}

const ConnectionBlock = ({ connection: c, isNew }: ConnectionBlockProps) => {
	const [isHovered, setIsHovered] = useState<boolean>(false);
	const [arrow, setArrow] = useState<ReactNode>();

	const { connections } = useContext(appContext);
	const { hoveredConnection, setHoveredConnection, isUserDbHovered, isEventDbHovered } =
		useContext(connectionMapContext);

	useEffect(() => {
		// Must wait for the block to be painted and styled before proceding
		// with the render of the arrow.

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

		const isConnected = c.actionsCount > 0;
		const isActive = c.relations(connections).length > 0;

		const hovered =
			isHovered ||
			c.relations(connections).includes(hoveredConnection) ||
			(isUserDbHovered && c.relations(connections).includes('dwh-user')) ||
			(isEventDbHovered && c.relations(connections).includes('dwh-event'));
		const isHighlighted = hovered && isConnected;

		const isSomethingHovered = hoveredConnection != null || isUserDbHovered || isEventDbHovered;
		const isHidden = !isConnected || (isSomethingHovered && !isHighlighted);

		const arrow = (
			<Arrow
				start={arrowStart}
				end={arrowEnd}
				startAnchor={arrowStartAnchor}
				endAnchor={arrowEndAnchor}
				color={isHighlighted && isActive ? '#4f46e5' : undefined}
				strokeWidth={1}
				dashness={
					isHighlighted && isActive
						? { strokeLen: 5, nonStrokeLen: 5, animation: c.isSource ? 2 : -2 }
						: false
				}
				data-is-hovered={isHighlighted && isActive}
				isNew={isNew}
				isHidden={isHidden}
				showTail={showTail && isConnected}
				showHead={showHead && isConnected}
				useCircleShape={true}
			/>
		);

		setTimeout(() => {
			setArrow(arrow);
		}, 0);
	}, [c, isHovered, hoveredConnection, isUserDbHovered, isEventDbHovered]);

	const onMouseEnter = () => {
		setIsHovered(true);
		setHoveredConnection(c.id);
	};

	const onMouseLeave = () => {
		setIsHovered(false);
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
					data-is-hovered={isHovered}
				>
					<div className='connection-block__content'>
						<Flex alignItems='center' gap={10}>
							<LittleLogo code={c.connector.code} />
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
