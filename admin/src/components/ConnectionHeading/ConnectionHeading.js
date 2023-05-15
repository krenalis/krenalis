import { useContext } from 'react';
import './ConnectionHeading.css';
import { AppContext } from '../../context/AppContext';
import UnknownLogo from '../UnknownLogo/UnknownLogo';
import LittleLogo from '../LittleLogo/LittleLogo';
import getConnectionStatusInfos from '../../utils/getConnectionStatusInfos';
import { SlBadge } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionHeading = ({ connection: c }) => {
	let { connectors } = useContext(AppContext);

	let { text: statusText, variant: statusVariant } = getConnectionStatusInfos(c);

	let connector = connectors.find((connector) => connector.ID === c.Connector);
	let logo;
	if (connector.Icon === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo icon={connector.Icon} />;
	}

	return (
		<div className='ConnectionHeading'>
			<div className='title'>
				{logo}
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
