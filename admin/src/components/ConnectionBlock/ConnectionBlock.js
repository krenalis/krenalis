import './ConnectionBlock.css';
import UnknownLogo from '../UnknownLogo/UnknownLogo';
import LittleLogo from '../LittleLogo/LittleLogo';
import FlexContainer from '../FlexContainer/FlexContainer';
import getConnectionStatusInfos from '../../utils/getConnectionStatusInfos';
import { NavLink } from 'react-router-dom';
import { SlTooltip, SlIcon } from '@shoelace-style/shoelace/dist/react/index.js';

const ConnectionBlock = ({ connection: c }) => {
	let logo;
	if (c.LogoURL === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo url={c.LogoURL} alternativeText={`${c.Name}'s logo`} />;
	}

	let { text: statusText, variant: statusVariant } = getConnectionStatusInfos(c);

	return (
		<div className='ConnectionBlock' id={`${c.ID}`}>
			<FlexContainer alignItems='center' justifyContent='space-between' gap={20}>
				<FlexContainer alignItems='center' gap={10}>
					{logo}
					<div className='name'>{c.Name}</div>
				</FlexContainer>
				<SlTooltip content={statusText}>
					<div className='hoverArea'>
						<SlIcon className={statusVariant} name='circle-fill'></SlIcon>
					</div>
				</SlTooltip>
			</FlexContainer>
			<NavLink to={`/admin/connections/${c.ID}`}></NavLink>
		</div>
	);
};

export default ConnectionBlock;
