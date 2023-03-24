import './OAuthError.css';
import { NavLink } from 'react-router-dom';
import { SlIcon, SlButton } from '@shoelace-style/shoelace/dist/react/index.js';

const OAuthError = () => {
	return (
		<div className='OAuthError'>
			<div className='error'>
				<SlIcon name='exclamation-circle-fill'></SlIcon>
				<div className='text'>Something went wrong during the OAuth authentication</div>
				<SlButton variant='default'>
					<SlIcon slot='suffix' name='arrow-right-circle'></SlIcon>
					Go to connections map
					<NavLink to='/admin/connections'></NavLink>
				</SlButton>
			</div>
		</div>
	);
};

export default OAuthError;
