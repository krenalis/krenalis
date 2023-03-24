import './StatusDot.css';
import { SlTooltip, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const StatusDot = ({ statusText, statusVariant }) => {
	return (
		<div className='StatusDot'>
			<SlTooltip content={statusText}>
				<div className='hoverArea'>
					<SlIcon className={statusVariant} name='circle-fill'></SlIcon>
				</div>
			</SlTooltip>
		</div>
	);
};

export default StatusDot;
