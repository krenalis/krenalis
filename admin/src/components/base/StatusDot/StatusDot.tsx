import React from 'react';
import './StatusDot.css';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
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
	return (
		<div className='status-dot'>
			{status.text != null ? (
				<SlTooltip content={status.text} placement={placement ? placement : 'top'}>
					<div className='status-dot__hover-area'>
						<SlIcon
							className={`status-dot__icon status-dot__icon--${status.variant}`}
							name='circle-fill'
						></SlIcon>
					</div>
				</SlTooltip>
			) : (
				<SlIcon className={`status-dot__icon status-dot__icon--${status.variant}`} name='circle-fill'></SlIcon>
			)}
		</div>
	);
};

export default StatusDot;
