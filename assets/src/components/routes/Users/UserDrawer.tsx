import React, { useState, useEffect, useContext, Fragment, useMemo } from 'react';
import { useUserDrawer } from './useUserDrawer';
import { UserTab } from './Users.types';
import AppContext from '../../../context/AppContext';
import UsersContext from '../../../context/UsersContext';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import SlTab from '@shoelace-style/shoelace/dist/react/tab/index.js';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import SlTabGroup from '@shoelace-style/shoelace/dist/react/tab-group/index.js';
import SlTabPanel from '@shoelace-style/shoelace/dist/react/tab-panel/index.js';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import getConnectorLogo from '../../helpers/getConnectorLogo';
import toJSDateString from '../../../lib/utils/toJSDateString';
import { Link } from '../../shared/Link/Link';

interface UserDrawerProps {
	selectedUser: number;
	setSelectedUser: React.Dispatch<React.SetStateAction<number>>;
}

const UserDrawer = ({ selectedUser, setSelectedUser }: UserDrawerProps) => {
	const [selectedTab, setSelectedTab] = useState<UserTab>();

	const { connections, workspaces, selectedWorkspace } = useContext(AppContext);
	const { userIDList, fetchUsers, pagination } = useContext(UsersContext);
	const { isLoading, traits, events, identities } = useUserDrawer(selectedUser, selectedTab);

	const workspace = useMemo(
		() => workspaces.find((w) => w.ID === selectedWorkspace),
		[workspaces, selectedWorkspace],
	);

	useEffect(() => {
		let tab: string;
		try {
			tab = localStorage.getItem('chichi_ui_users_tab');
		} catch (err) {
			setSelectedTab('traits');
			return;
		}
		if (tab === 'traits' || tab === 'events' || tab === 'identities') {
			setSelectedTab(tab);
			return;
		}
		setSelectedTab('traits');
	}, [selectedUser]);

	useEffect(() => {
		try {
			localStorage.setItem('chichi_ui_users_tab', selectedTab);
		} catch (err) {
			console.error(`cannot write the user tab preference on local storage: ${err}`);
			return;
		}
	}, [selectedTab]);

	const onNavigate = async (direction: 'previous' | 'next') => {
		const i = userIDList.findIndex((id) => id === selectedUser);
		let newUserID: number;
		if (direction === 'previous') {
			if (i - 1 < 0) {
				// if the index is overflowing the start of the users list.
				if (pagination.last === 1) {
					// if there is only one page of users, select the last user.
					newUserID = userIDList[userIDList.length - 1];
				} else {
					// otherwise fetch the previous page and select the last
					// user.
					const page = pagination.current === 1 ? pagination.last : pagination.current - 1;
					const ids = await fetchUsers(page);
					newUserID = ids[ids.length - 1];
				}
			} else {
				// select the previous user.
				newUserID = userIDList[i - 1];
			}
		} else if (direction === 'next') {
			// if the index is overflowing the end of the users list.
			if (i + 1 >= userIDList.length) {
				if (pagination.last === 1) {
					// if there is only one page of users, select the first user.
					newUserID = userIDList[0];
				} else {
					// otherwise fetch the next page and select the first user.
					const page = pagination.current === pagination.last ? 1 : pagination.current + 1;
					const ids = await fetchUsers(page);
					newUserID = ids[0];
				}
			} else {
				// select the next user.
				newUserID = userIDList[i + 1];
			}
		}
		setSelectedUser(newUserID);
	};

	const onSelectTab = (e) => {
		setSelectedTab(e.detail.name);
	};

	const onClose = () => {
		setSelectedUser(0);
	};

	let userImage: any;
	let userFirstName: any;
	let userLastName: any;
	let userInformation: any;
	if (traits && traits.size > 0) {
		const t = Array.from(traits);
		for (const [key, value] of t) {
			switch (key) {
				case workspace.DisplayedProperties.Image:
					userImage = value;
					break;
				case workspace.DisplayedProperties.FirstName:
					userFirstName = value;
					break;
				case workspace.DisplayedProperties.LastName:
					userLastName = value;
					break;
				case workspace.DisplayedProperties.Information:
					userInformation = value;
					break;
			}
		}
	}

	const spinner = (
		<SlSpinner
			style={
				{
					fontSize: '3rem',
					'--track-width': '6px',
				} as React.CSSProperties
			}
		></SlSpinner>
	);

	return (
		<SlDrawer
			className='user-drawer'
			open={selectedUser !== 0}
			contained
			style={{ '--size': '600px' } as React.CSSProperties}
			onSlHide={onClose}
		>
			<div className='user-drawer__navigation'>
				<SlIconButton name='chevron-left' onClick={() => onNavigate('previous')} />
				<SlIconButton name='chevron-right' onClick={() => onNavigate('next')} />
			</div>
			<div className='user-drawer__top-section'>
				<SlAvatar className='user-drawer__image' image={userImage || ''} />
				<div className='user-drawer__user-properties'>
					<span className='user-drawer__first-name'>{userFirstName || ''}</span>{' '}
					<span className='user-drawer__last-name'>{userLastName || ''}</span>
					<div className='user-drawer__information'>{userInformation || ''}</div>
					{userImage == null && userFirstName == null && userLastName == null && userInformation == null && (
						<div className='user-drawer__customize'>
							You can customize the displayed properties in the{' '}
							<Link path='settings/general'>
								<span className='user-drawer__customize-link'>settings</span>
							</Link>
						</div>
					)}
				</div>
			</div>
			<SlTabGroup onSlTabShow={onSelectTab}>
				<SlTab slot='nav' panel='traits' active={selectedTab === 'traits'}>
					Traits
				</SlTab>
				<SlTab slot='nav' panel='events' active={selectedTab === 'events'}>
					Events
				</SlTab>
				<SlTab slot='nav' panel='identities' active={selectedTab === 'identities'}>
					Identities
				</SlTab>
				<SlTabPanel name='traits'>
					<div className='user-drawer__traits'>
						{isLoading ? (
							spinner
						) : traits && traits.size > 0 ? (
							Array.from(traits).map(([key, value]) => {
								return (
									<Fragment key={key}>
										<span className='user-drawer__trait-key'>{key}:</span>{' '}
										{typeof value === 'object' ? (
											<span className='user-drawer__trait-object'>{JSON.stringify(value)}</span>
										) : (
											<div className='user-drawer__trait-value'>{value}</div>
										)}
									</Fragment>
								);
							})
						) : (
							<div className='user-drawer__no-traits'>No traits associated to this user</div>
						)}
					</div>
				</SlTabPanel>
				<SlTabPanel name='events'>
					<div
						className={`user-drawer__events${selectedTab === 'events' ? ' user-drawer__events--selected' : ''}`}
					>
						{isLoading ? (
							spinner
						) : events && events.length > 0 ? (
							events.map((event) => {
								const source = connections.find((c) => c.id === event.source);
								const logo = getConnectorLogo(source.connector.icon);
								return (
									<div className='user-drawer__event' key={event.sentAt}>
										<div className='user-drawer__event-head'>
											<Link path={`connections/${source.id}/actions`}>
												<div className='user-drawer__event-logo'>{logo}</div>
											</Link>
											<div className='user-drawer__event-type'>{event.type}</div>
										</div>
										<div className='user-drawer__event-sent-at'>
											{new Date(toJSDateString(event.sentAt)).toLocaleString('it-IT', {
												timeZone: 'Europe/Rome',
											})}
										</div>
									</div>
								);
							})
						) : (
							<div className='user-drawer__no-events'>No events associated to this user</div>
						)}
					</div>
				</SlTabPanel>
				<SlTabPanel name='identities'>
					<div
						className={`user-drawer__identities${selectedTab === 'identities' ? ' user-drawer__identities--selected' : ''}`}
					>
						{isLoading ? (
							spinner
						) : identities && identities.length > 0 ? (
							identities.map((identity) => {
								const connection = connections.find((c) => c.id === identity.Connection);
								const logo = getConnectorLogo(connection.connector.icon);
								return (
									<div className='user-drawer__identity' key={identity.LastChangeTime}>
										<Link path={`connections/${connection.id}/actions`}>
											<div className='user-drawer__identity-connection-logo'>{logo}</div>
										</Link>
										<div className='user-drawer__identity-info'>
											<div className='user-drawer__identity-connection-date'>
												<Link path={`connections/${connection.id}/actions`}>
													<div className='user-drawer__identity-connection-name'>
														{connection.name}
													</div>
												</Link>
												<div className='user-drawer__identity-date'>
													{new Date(toJSDateString(identity.LastChangeTime)).toLocaleString(
														'it-IT',
														{
															timeZone: 'Europe/Rome',
														},
													)}
												</div>
											</div>
											<div className='user-drawer__identity-id'>
												{identity.IdentityId.Label}: <code>{identity.IdentityId.Value}</code>
											</div>
											<div className='user-drawer__identity-displayed-property'>
												Displayed property: <code>{identity.DisplayedProperty}</code>
											</div>
											{identity.AnonymousIds !== null && (
												<div className='user-drawer__identity-anonymous-ids'>
													Anonymous IDs: <code>{identity.AnonymousIds.join(', ')}</code>
												</div>
											)}
										</div>
									</div>
								);
							})
						) : (
							<div className='user-drawer__no-identities'>No identities associated to this user</div>
						)}
					</div>
				</SlTabPanel>
			</SlTabGroup>
		</SlDrawer>
	);
};

export { UserDrawer };
