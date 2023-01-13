import './UnknownLogo.css';
import { SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const UnknownLogo = ({ size }) => {
	return (
		<div className='UnknownLogo'>
			<SlIcon name='question-lg' style={{ fontSize: `${size}px` }}></SlIcon>
		</div>
	);
};

export default UnknownLogo;
