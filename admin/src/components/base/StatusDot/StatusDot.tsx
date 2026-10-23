import React, { useId } from 'react';
import './StatusDot.css';
import WaTooltip from '@awesome.me/webawesome/dist/react/tooltip/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
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
					<WaTooltip for={hoverAreaID} placement={placement ? placement : 'top'}>
						{status.text}
					</WaTooltip>
					<div id={hoverAreaID} className='status-dot__hover-area'>
						<WaIcon
							className={`status-dot__icon status-dot__icon--${status.variant}`}
							name='circle-fill'
						></WaIcon>
					</div>
				</>
			) : (
				<WaIcon className={`status-dot__icon status-dot__icon--${status.variant}`} name='circle-fill'></WaIcon>
			)}
		</div>
	);
};

export default StatusDot;
