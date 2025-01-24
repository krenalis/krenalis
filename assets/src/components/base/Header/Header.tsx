import React, { ReactNode } from 'react';
import './Header.css';
import IconWrapper from '../IconWrapper/IconWrapper';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import { TransformedMember } from '../../../lib/core/member';
import { Link } from '..//Link/Link';

interface HeaderProps {
	title: ReactNode;
	member: TransformedMember;
}

const Header = ({ title, member }: HeaderProps) => {
	return (
		<div className='header'>
			<div className='header__title'>
				<span>{title}</span>
			</div>
			<div className='header__account'>
				<IconWrapper name='bell' moat={true}></IconWrapper>
				<IconWrapper name='question-lg' moat={true}></IconWrapper>
				<Link path='organization'>
					<IconWrapper name='building' moat={true}></IconWrapper>
				</Link>
				<Link path='organization/members/current'>
					<SlAvatar
						className='header__account-avatar'
						initials={member.initials}
						image={member.avatar ? `data:${member.avatar.mimeType};base64, ${member.avatar.image}` : ''}
					/>
				</Link>
			</div>
		</div>
	);
};

export default Header;
