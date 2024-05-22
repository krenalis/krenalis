import React, { useState, useEffect, ReactNode } from 'react';
import Flex from '../../base/Flex/Flex';
import Arrow from '../../base/Arrow/Arrow';
import StatusDot from '../../base/StatusDot/StatusDot';
import { ArrowAnchor } from '../../base/Arrow/Arrow.types';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import TransformedConnection from '../../../lib/core/connection';
import { Link } from '../../base/Link/Link';

interface ConnectionBlockProps {
	connection: TransformedConnection;
	isNew: boolean;
}

const ConnectionBlock = ({ connection: c, isNew }: ConnectionBlockProps) => {
	const [isHovered, setIsHovered] = useState<boolean>(false);
	const [arrow, setArrow] = useState<ReactNode>();

	useEffect(() => {
		// Must wait for the block to be painted and styled before proceding
		// with the render of the arrow.
		let arrowStart: string, arrowEnd: string, arrowStartAnchor: ArrowAnchor, arrowEndAnchor: ArrowAnchor;
		if (c.isFile) {
			arrowStart = `${c.id}`;
			arrowEnd = `${c.storage}`;
			arrowStartAnchor = c.isSource ? 'right' : 'left';
			arrowEndAnchor = c.isSource ? 'left' : 'right';
		} else {
			arrowStart = `${c.id}`;
			arrowEnd = 'central-logo';
			arrowStartAnchor = c.isSource ? 'right' : 'left';
			arrowEndAnchor = c.isSource ? 'left' : 'right';
		}
		const arrow = (
			<Arrow
				start={arrowStart}
				end={arrowEnd}
				startAnchor={arrowStartAnchor}
				endAnchor={arrowEndAnchor}
				color={isHovered ? '#4f46e5' : undefined}
				dashness={isHovered ? { strokeLen: 5, nonStrokeLen: 5, animation: c.isSource ? 2 : -2 } : false}
				data-is-hovered={isHovered}
				isNew={isNew}
			/>
		);
		setTimeout(() => {
			setArrow(arrow);
		}, 0);
	}, [c, isHovered]);

	const onMouseEnter = () => {
		setIsHovered(true);
	};

	const onMouseLeave = () => {
		setIsHovered(false);
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
							{getConnectorLogo(c.connector.icon)}
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
