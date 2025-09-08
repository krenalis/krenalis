import React, { ReactNode, useContext, useEffect, useRef, useState } from 'react';
import './Header.css';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlDivider from '@shoelace-style/shoelace/dist/react/divider/index.js';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import { Link } from '..//Link/Link';
import appContext from '../../../context/AppContext';
import { useLocation } from 'react-router-dom';
import { Member } from '../../../lib/api/types/responses';

interface HeaderProps {
	title: ReactNode;
	member: Member;
}

const Header = ({ title, member }: HeaderProps) => {
	const [isTooltipOpen, setIsTooltipOpen] = useState<boolean>(false);

	const { isPasswordless, logout, isFullscreen } = useContext(appContext);

	const location = useLocation();

	const dropdownRef = useRef<any>();

	useEffect(() => {
		if (isPasswordless && !isFullscreen) {
			setIsTooltipOpen(true);
		}
	}, []);

	useEffect(() => {
		if (isTooltipOpen) {
			setIsTooltipOpen(false);
		}
	}, [location]);

	const onLogout = () => {
		closeMenu();
		logout();
	};

	const closeMenu = () => {
		if (dropdownRef.current == null) {
			return;
		}
		dropdownRef.current.hide();
	};

	return (
		<div className='header'>
			<div className='header__title'>
				<span>{title}</span>
			</div>
			<div className='header__account'>
				<SlDropdown distance={17} ref={dropdownRef} open={isTooltipOpen}>
					<SlAvatar
						slot='trigger'
						className='header__account-avatar'
						image={member.avatar ? `data:${member.avatar.mimeType};base64, ${member.avatar.image}` : ''}
					/>
					<SlMenu className='header__account-menu-wrapper'>
						{isPasswordless && (
							<div className='header__passwordless-tooltip'>
								<div className='header__passwordless-tooltip-body' slot='content'>
									<div className='header__passwordless-tooltip-title'>
										You are signed in with default credentials
									</div>
									If you prefer, you can create a new account with your credentials instead of using
									the default ones.
								</div>
								<Link path='organization/members/add'>
									<SlButton
										className='header__passwordless-create-account'
										variant='warning'
										size='small'
									>
										Create my account
									</SlButton>
								</Link>
							</div>
						)}
						<div className='header__account-menu'>
							<div className='header__account-menu-heading'>
								<SlAvatar
									slot='trigger'
									className='header__account-menu-heading-avatar'
									image={
										member.avatar
											? `data:${member.avatar.mimeType};base64, ${member.avatar.image}`
											: ''
									}
								/>
								<div className='header__account-menu-heading-text'>
									<div className='header__account-menu-heading-name'>{member.name}</div>
									<div className='header__account-menu-heading-email'>{member.email}</div>
								</div>
							</div>
							<SlDivider style={{ '--spacing': '6px' } as React.CSSProperties} />
							<Link
								className='header__account-menu-item'
								path='organization/members/current'
								onClick={closeMenu}
							>
								<SlIcon className='header__account-menu-item-icon' name='person' />
								Your profile
							</Link>
							<Link className='header__account-menu-item' path='organization' onClick={closeMenu}>
								<SlIcon className='header__account-menu-item-icon' name='building' />
								Your organization
							</Link>
							<Link
								className='header__account-menu-item header__account-menu-item--indented'
								path='organization/members'
								onClick={closeMenu}
							>
								<SlIcon className='header__account-menu-item-icon' name='people' />
								Members
							</Link>
							<Link
								className='header__account-menu-item header__account-menu-item--indented'
								path='organization/access-keys'
								onClick={closeMenu}
							>
								<SlIcon className='header__account-menu-item-icon' name='key' />
								API and MCP keys
							</Link>
							<SlDivider style={{ '--spacing': '6px' } as React.CSSProperties} />
							<a
								className='header__account-menu-item'
								href='https://github.com/meergo/meergo/issues'
								target='_blank'
								onClick={closeMenu}
							>
								<SlIcon className='header__account-menu-item-icon' name='bug' />
								Report a bug
							</a>
							<a
								className='header__account-menu-item'
								href='https://github.com/meergo/meergo/discussions'
								target='_blank'
								onClick={closeMenu}
							>
								<SlIcon className='header__account-menu-item-icon' name='chat-dots' />
								Ask for help
							</a>
							{!isPasswordless && (
								<>
									<SlDivider style={{ '--spacing': '6px' } as React.CSSProperties} />
									<div className='header__account-menu-item' id='logout-button' onClick={onLogout}>
										<SlIcon className='header__account-menu-item-icon' name='box-arrow-right' />
										Logout
									</div>
								</>
							)}
						</div>
					</SlMenu>
				</SlDropdown>
			</div>
		</div>
	);
};

export default Header;
