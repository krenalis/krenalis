import './ConnectionHeading.css';
import { SlBadge } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionHeading = ({ connection: cn }) => {
	return (
		<div className='ConnectionHeading'>
			<div className='title'>
				{cn.LogoURL !== '' && <img className='littleLogo' src={cn.LogoURL} alt={`${cn.Name}'s logo`} />}
				<div className='text'>{cn.Name}</div>
			</div>
			<div className='badges'>
				<SlBadge className='type' variant='neutral'>
					{cn.Type}
				</SlBadge>
				<SlBadge className='role' variant='neutral'>
					{cn.Role}
				</SlBadge>
			</div>
		</div>
	);
};

export default ConnectionHeading;
