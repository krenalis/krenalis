import React from 'react';
import './StatusDot.css';
import { SlTooltip, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';
import { Status } from '../../../types/app';

interface StatusDotProps {
	status: Status;
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
