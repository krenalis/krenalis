import React, { ReactNode, useContext, useEffect, useState } from 'react';
import './Header.css';
import IconWrapper from '../IconWrapper/IconWrapper';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { TransformedMember } from '../../../lib/core/member';
import { Link } from '..//Link/Link';
import appContext from '../../../context/AppContext';
import { useLocation } from 'react-router-dom';

interface HeaderProps {
	title: ReactNode;
	member: TransformedMember;
}

const Header = ({ title, member }: HeaderProps) => {
	const [isTooltipOpen, setIsTooltipOpen] = useState<boolean>(false);

	const { isPasswordless } = useContext(appContext);

	const location = useLocation();

	useEffect(() => {
		if (isPasswordless) {
			setTimeout(() => {
				setIsTooltipOpen(true);
			}, 1000);
		}
	}, []);

	useEffect(() => {
		if (isTooltipOpen) {
			setIsTooltipOpen(false);
		}
	}, [location]);

	const onAddMemberClick = () => {
		setIsTooltipOpen(false);
	};

	return (
		<div className='header'>
			<div className='header__title'>
				<span>{title}</span>
			</div>
			<div className='header__account'>
				<IconWrapper name='bell' moat={true} />
				<IconWrapper name='question-lg' moat={true} />
				<SlTooltip
					style={{ '--max-width': '350px' } as React.CSSProperties}
					className='header__passwordless-tooltip'
					open={isTooltipOpen}
					trigger='manual'
					placement='bottom-end'
				>
					<div className='header__passwordless-tooltip-body' slot='content'>
						<div className='header__passwordless-tooltip-title'>
							You are signed in with default credentials
						</div>
						If you prefer, you can{' '}
						<Link path='organization/members' onClick={onAddMemberClick}>
							create a new member
						</Link>{' '}
						with your personal credentials instead of using the default ones.
						<SlButton
							onClick={() => setIsTooltipOpen(false)}
							className='header__passwordless-tooltip-close'
							size='small'
						>
							Close
						</SlButton>
					</div>
					<Link path='organization'>
						<IconWrapper name='building' moat={true} />
					</Link>
				</SlTooltip>
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
