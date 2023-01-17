import './ConnectionHeading.css';
import getConnectionStatusInfos from '../../utils/getConnectionStatusInfos';
import { SlBadge } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionHeading = ({ connection: c }) => {
	let { text: statusText, variant: statusVariant } = getConnectionStatusInfos(c);

	return (
		<div className='ConnectionHeading'>
			<div className='title'>
				{c.LogoURL !== '' && <img className='littleLogo' src={c.LogoURL} alt={`${c.Name}'s logo`} />}
				<div className='text'>{c.Name}</div>
			</div>
			<div className='badges'>
				<SlBadge className='type' variant='neutral'>
					{c.Type}
				</SlBadge>
				<SlBadge className='role' variant='neutral'>
					{c.Role}
				</SlBadge>
				<SlBadge className={`status ${statusVariant}`}>{statusText}</SlBadge>
			</div>
		</div>
	);
};

export default ConnectionHeading;
