import React, { ReactNode, useContext } from 'react';
import './Header.css';
import IconWrapper from '../../shared/IconWrapper/IconWrapper';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import AppContext from '../../../context/AppContext';
import { TransformedMember } from '../../../lib/helpers/transformedMember';

interface HeaderProps {
	title: ReactNode;
	member: TransformedMember;
}

const Header = ({ title, member }: HeaderProps) => {
	const { redirect } = useContext(AppContext);

	const onOrganizationClick = () => {
		redirect('organization');
	};

	const onMemberClick = () => {
		redirect(`members/current`);
	};

	return (
		<div className='header'>
			<div className='title'>
				<span>{title}</span>
			</div>
			<div className='account'>
				<IconWrapper name='bell' moat={true}></IconWrapper>
				<IconWrapper name='question-lg' moat={true}></IconWrapper>
				<IconWrapper name='building' onClick={onOrganizationClick} moat={true}></IconWrapper>
				<SlAvatar
					className='accountAvatar'
					initials={member.Initials}
					image={member.Avatar ? `data:${member.Avatar.MimeType};base64, ${member.Avatar.Image}` : ''}
					onClick={onMemberClick}
				/>
			</div>
		</div>
	);
};

export default Header;
