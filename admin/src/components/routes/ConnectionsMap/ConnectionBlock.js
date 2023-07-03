import { useContext, useState, useEffect } from 'react';
import Flex from '../../common/Flex/Flex';
import Arrow from '../../common/Arrow/Arrow';
import StatusDot from '../../common/StatusDot/StatusDot';
import { AppContext } from '../../../providers/AppProvider';
import { SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionBlock = ({ connection: c, isNew }) => {
	const [isHovered, setIsHovered] = useState(false);
	const [arrow, setArrow] = useState(null);

	const { redirect } = useContext(AppContext);

	useEffect(() => {
		// Must wait for the block to be painted and styled before proceding
		// with the render of the arrow.
		let arrowStart, arrowEnd, arrowStartAnchor, arrowEndAnchor;
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
				color={isHovered ? '#4f46e5' : null}
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
				<Flex alignItems='center' justifyContent='space-between' gap={20}>
					<Flex alignItems='center' gap={10}>
						{c.logo}
						<div className='name'>{c.name}</div>
					</Flex>
					<SlTooltip content={c.status.text}>
						<StatusDot status={c.status} />
					</SlTooltip>
				</Flex>
			</div>
			{arrow}
		</>
	);
};

export default ConnectionBlock;
