import './ConnectionBlock.css';
import UnknownLogo from '../UnknownLogo/UnknownLogo';
import LittleLogo from '../LittleLogo/LittleLogo';
import FlexContainer from '../FlexContainer/FlexContainer';
import { NavLink } from 'react-router-dom';

const ConnectionBlock = ({ connection: c }) => {
	let logo;
	if (c.LogoURL === '') {
		logo = <UnknownLogo size={21} />;
	} else {
		logo = <LittleLogo url={c.LogoURL} alternativeText={`${c.Name}'s logo`} />;
	}
	return (
		<div className='ConnectionBlock' id={`${c.ID}`}>
			<FlexContainer alignItems='center' gap={20}>
				{logo}
				<div className='name'>{c.Name}</div>
			</FlexContainer>
			<NavLink to={`/admin/connections/${c.ID}`}></NavLink>
		</div>
	);
};

export default ConnectionBlock;
