import './IconWrapper.css';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const IconWrapper = ({ name, size, moat }) => {
	return (
		<div className={`iconWrapper${moat ? ' moat' : ''}`} style={{ '--icon-size': size ? `${size}px` : '16px' }}>
			<div className='innerWrapper'>
				<SlIcon name={name}></SlIcon>
			</div>
		</div>
	);
};

export default IconWrapper;
