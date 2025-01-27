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
import toJSDateString from '../../../utils/toJSDateString';
import { Link } from '../../base/Link/Link';
import JSONbig from 'json-bigint';

interface UserDrawerProps {
	selectedUser: string;
	setSelectedUser: React.Dispatch<React.SetStateAction<string>>;
}

const UserDrawer = ({ selectedUser, setSelectedUser }: UserDrawerProps) => {
	const [selectedTab, setSelectedTab] = useState<UserTab>();

	const { connections, workspaces, selectedWorkspace } = useContext(AppContext);
	const { userIDList } = useContext(UsersContext);
	const { isLoading, traits, events, identities } = useUserDrawer(selectedUser, selectedTab);

	const workspace = useMemo(
		() => workspaces.find((w) => w.id === selectedWorkspace),
		[workspaces, selectedWorkspace],
	);

	useEffect(() => {
		let tab: string;
		try {
			tab = localStorage.getItem('meergo_ui_users_tab');
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
			localStorage.setItem('meergo_ui_users_tab', selectedTab);
		} catch (err) {
			console.error(`cannot write the user tab preference on local storage: ${err}`);
			return;
		}
	}, [selectedTab]);

	const onNavigate = async (direction: 'previous' | 'next') => {
		const i = userIDList.findIndex((id) => id === selectedUser);
		let newUserID: string;
		if (direction === 'previous') {
			if (i - 1 < 0) {
				// if the index is overflowing the start of the users list,
				// select the last user.
				newUserID = userIDList[userIDList.length - 1];
			} else {
				// select the previous user.
				newUserID = userIDList[i - 1];
			}
		} else if (direction === 'next') {
			if (i + 1 >= userIDList.length) {
				// if the index is overflowing the end of the users list, select
				// the first user.
				newUserID = userIDList[0];
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
		setSelectedUser('');
	};

	let userImage: string | number | undefined;
	let userFirstName: string | number | undefined;
	let userLastName: string | number | undefined;
	let userExtra: string | number | undefined;
	if (traits && Object.keys(traits).length > 0) {
		function getValueFromPath(path: string): string | number | undefined {
			if (path == '') {
				return undefined;
			}
			let v: any = traits;
			for (const part of path.split('.')) {
				if (typeof v === 'object' && v !== null && part in v) {
					v = v[part];
				}
			}
			if (typeof v != 'string' && typeof v != 'number') {
				return undefined;
			} else {
				return v;
			}
		}
		userImage = getValueFromPath(workspace.uiPreferences.userProfile.image);
		userFirstName = getValueFromPath(workspace.uiPreferences.userProfile.firstName);
		userLastName = getValueFromPath(workspace.uiPreferences.userProfile.lastName);
		userExtra = getValueFromPath(workspace.uiPreferences.userProfile.extra);
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
			open={selectedUser !== ''}
			contained
			style={{ '--size': '600px' } as React.CSSProperties}
			onSlHide={onClose}
		>
			<div className='user-drawer__navigation'>
				<SlIconButton name='chevron-left' onClick={() => onNavigate('previous')} />
				<SlIconButton name='chevron-right' onClick={() => onNavigate('next')} />
			</div>
			<div className='user-drawer__top-section'>
				<SlAvatar className='user-drawer__image' image={String(userImage) || ''} />
				<div className='user-drawer__user-properties'>
					<span className='user-drawer__first-name'>{userFirstName || ''}</span>{' '}
					<span className='user-drawer__last-name'>{userLastName || ''}</span>
					<div className='user-drawer__information'>{userExtra || ''}</div>
					{userImage == null && userFirstName == null && userLastName == null && userExtra == null && (
						<div className='user-drawer__customize'>
							You can customize the properties to display in the{' '}
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
						) : traits && Object.keys(traits).length > 0 ? (
							Object.entries(traits).map(([key, value]) => {
								return (
									<Fragment key={key}>
										<span className='user-drawer__trait-key'>{key}:</span>{' '}
										{typeof value === 'object' ? (
											<span className='user-drawer__trait-object'>
												{JSONbig.stringify(value)}
											</span>
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
								const source = connections.find((c) => c.id === event.connection);
								const logo = getConnectorLogo(source?.connector.icon);
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
								const connection = connections.find((c) => c.id === identity.connection);
								const logo = getConnectorLogo(connection?.connector.icon);
								return (
									<div className='user-drawer__identity' key={identity.lastChangeTime}>
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
													{new Date(toJSDateString(identity.lastChangeTime)).toLocaleString(
														'it-IT',
														{
															timeZone: 'Europe/Rome',
														},
													)}
												</div>
											</div>
											<div className='user-drawer__action'>
												Imported from action: <code>{identity.action}</code>
											</div>
											{identity.id && (
												<div className='user-drawer__identity-id'>
													{connection.connector.getIdentityIDLabel()}:{' '}
													<code>{identity.id}</code>
												</div>
											)}
											{identity.anonymousIds !== null && (
												<div className='user-drawer__identity-anonymous-ids'>
													Anonymous IDs: <code>{identity.anonymousIds.join(', ')}</code>
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
