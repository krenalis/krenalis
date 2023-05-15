import { useContext } from 'react';
import './ConnectionBlock.css';
import UnknownLogo from '../UnknownLogo/UnknownLogo';
import LittleLogo from '../LittleLogo/LittleLogo';
import Flex from '../Flex/Flex';
import getConnectionStatusInfos from '../../utils/getConnectionStatusInfos';
import StatusDot from '../StatusDot/StatusDot';
import { AppContext } from '../../context/AppContext';
import { NavLink } from 'react-router-dom';
import { SlTooltip } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionBlock = ({ connection: c, isNew }) => {
	let { text: statusText, variant: statusVariant } = getConnectionStatusInfos(c);

	let { connectors } = useContext(AppContext);
	let connector = connectors.find((connector) => connector.ID === c.Connector);
	let logo;
	if (connector.Icon === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo icon={connector.Icon} />;
	}

	return (
		<div className={`ConnectionBlock${isNew ? ' new' : ''}`} id={`${c.ID}`}>
			<Flex alignItems='center' justifyContent='space-between' gap={20}>
				<Flex alignItems='center' gap={10}>
					{logo}
					<div className='name'>{c.Name}</div>
				</Flex>
				<SlTooltip content={statusText}>
					<StatusDot statusText={statusText} statusVariant={statusVariant} />
				</SlTooltip>
			</Flex>
			<NavLink to={`/admin/connections/${c.ID}/actions`}></NavLink>
		</div>
	);
};

export default ConnectionBlock;
