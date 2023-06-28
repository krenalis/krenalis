import './StatusDot.css';
import { SlTooltip, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const StatusDot = ({ status }) => {
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
