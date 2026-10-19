import React, { useId } from 'react';
import './StatusDot.css';
import SlTooltip from '@awesome.me/webawesome/dist/react/tooltip/index.js';
import SlIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import { ConnectionStatus } from '../../../lib/core/connection';

interface StatusDotProps {
	status: ConnectionStatus;
	placement?:
		| 'top'
		| 'top-start'
		| 'top-end'
		| 'right'
		| 'right-start'
		| 'right-end'
		| 'bottom'
		| 'bottom-start'
		| 'bottom-end'
		| 'left'
		| 'left-start'
		| 'left-end';
}

const StatusDot = ({ status, placement }: StatusDotProps) => {
	const hoverAreaID = useId();
	return (
		<div className='status-dot'>
			{status.text != null ? (
				<>
					<SlTooltip for={hoverAreaID} placement={placement ? placement : 'top'}>
						{status.text}
					</SlTooltip>
					<div id={hoverAreaID} className='status-dot__hover-area'>
						<SlIcon
							className={`status-dot__icon status-dot__icon--${status.variant}`}
							name='circle-fill'
						></SlIcon>
					</div>
				</>
			) : (
				<SlIcon className={`status-dot__icon status-dot__icon--${status.variant}`} name='circle-fill'></SlIcon>
			)}
		</div>
	);
};

export default StatusDot;
