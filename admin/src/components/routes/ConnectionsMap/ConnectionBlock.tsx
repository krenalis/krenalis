import React, { useContext, useState, useEffect, ReactNode } from 'react';
import Flex from '../../shared/Flex/Flex';
import Arrow from '../../shared/Arrow/Arrow';
import StatusDot from '../../shared/StatusDot/StatusDot';
import { ArrowAnchor } from '../../../types/internal/app';
import AppContext from '../../../context/AppContext';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import TransformedConnection from '../../../lib/helpers/transformedConnection';

interface ConnectionBlockProps {
	connection: TransformedConnection;
	isNew: boolean;
}

const ConnectionBlock = ({ connection: c, isNew }: ConnectionBlockProps) => {
	const [isHovered, setIsHovered] = useState<boolean>(false);
	const [arrow, setArrow] = useState<ReactNode>();

	const { redirect } = useContext(AppContext);

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
			arrowEnd = 'centralLogo';
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

	const onClick = () => {
		redirect(`connections/${c.id}/actions`);
	};

	const onMouseEnter = () => {
		setIsHovered(true);
	};

	const onMouseLeave = () => {
		setIsHovered(false);
	};

	return (
		<>
			<div
				className={`connectionBlock${isNew ? ' new' : ''}`}
				id={`${c.id}`}
				onClick={onClick}
				onMouseEnter={onMouseEnter}
				onMouseLeave={onMouseLeave}
				data-is-hovered={isHovered}
			>
				<div className='connectionBlockContent'>
					<Flex alignItems='center' gap={10}>
						{getConnectorLogo(c.connector.icon)}
						<div className='name'>{c.name}</div>
					</Flex>
					<StatusDot status={c.status} />
				</div>
			</div>
			{arrow}
		</>
	);
};

export default ConnectionBlock;
