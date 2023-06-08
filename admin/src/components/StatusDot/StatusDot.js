import './StatusDot.css';
import { SlTooltip, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const StatusDot = ({ statusText, statusVariant }) => {
	return (
		<div className='StatusDot'>
			{statusText != null ? (
				<SlTooltip content={statusText}>
					<div className='hoverArea'>
						<SlIcon className={statusVariant} name='circle-fill'></SlIcon>
					</div>
				</SlTooltip>
			) : (
				<SlIcon className={statusVariant} name='circle-fill'></SlIcon>
			)}
		</div>
	);
};

export default StatusDot;
