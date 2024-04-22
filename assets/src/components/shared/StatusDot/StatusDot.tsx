import React from 'react';
import './StatusDot.css';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import { ConnectionStatus } from '../../../lib/helpers/transformedConnection';

interface StatusDotProps {
	status: ConnectionStatus;
}

const StatusDot = ({ status }: StatusDotProps) => {
	return (
		<div className='statusDot'>
			{status.text != null ? (
				<SlTooltip content={status.text}>
					<div className='hoverArea'>
						<SlIcon className={status.variant} name='circle-fill'></SlIcon>
					</div>
				</SlTooltip>
			) : (
				<SlIcon className={status.variant} name='circle-fill'></SlIcon>
			)}
		</div>
	);
};

export default StatusDot;
