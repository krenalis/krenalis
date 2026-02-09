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

	const onLogout = async () => {
		closeMenu();
		await logout();
	};

	const closeMenu = () => {
		if (dropdownRef.current == null) {
			return;
		}
		dropdownRef.current.hide();
	};

	return (
		<header>
			<div className='header__logo'>
				<img
					alt='Meergo logo'
					src='data:image/png;base64,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'
				/>
			</div>
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
									<SlButton className='header__passwordless-create-account' size='small'>
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
								className='header__account-menu-item header__account-menu-item--profile'
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
							<Link className='header__account-menu-item' path='organization/members' onClick={closeMenu}>
								<SlIcon className='header__account-menu-item-icon' name='people' />
								Team members
							</Link>
							<Link
								className='header__account-menu-item'
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
		</header>
	);
};

export default Header;
