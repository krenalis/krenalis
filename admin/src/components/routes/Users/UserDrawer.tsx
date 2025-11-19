import React, { useState, useEffect, useContext, useMemo, useRef } from 'react';
import { useUserDrawer } from './useUserDrawer';
import { UserTab } from './Users.types';
import AppContext from '../../../context/AppContext';
import UsersContext from '../../../context/UsersContext';
import SlDrawer from '@shoelace-style/shoelace/dist/react/drawer/index.js';
import SlTab from '@shoelace-style/shoelace/dist/react/tab/index.js';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import SlTabGroup from '@shoelace-style/shoelace/dist/react/tab-group/index.js';
import SlCopyButton from '@shoelace-style/shoelace/dist/react/copy-button/index.js';
import SlTabPanel from '@shoelace-style/shoelace/dist/react/tab-panel/index.js';
import SlIconButton from '@shoelace-style/shoelace/dist/react/icon-button/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlTooltip from '@shoelace-style/shoelace/dist/react/tooltip/index.js';
import toJSDate from '../../../utils/toJSDate';
import { Link } from '../../base/Link/Link';
import { USERS_EXPANDED_TRAITS_KEY, USERS_TAB_KEY } from '../../../constants/storage';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

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
			tab = localStorage.getItem(USERS_TAB_KEY);
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
			localStorage.setItem(USERS_TAB_KEY, selectedTab);
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

	const onSelectTab = (e: any) => {
		setSelectedTab(e.detail.name);
	};

	const onClose = (e: any) => {
		if (
			e.target.classList.contains('drawer-trait__value-copy') ||
			e.target.classList.contains('user-drawer__action')
		) {
			e.stopPropagation();
			return;
		}
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
			className={`user-drawer${isLoading ? ' user-drawer--loading' : ''}`}
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
					<span className='user-drawer__muid'>
						<SlTooltip
							content='Meergo User ID'
							onSlHide={(e) => {
								// Prevent the event from bubbling up and
								// causing the drawer to close.
								e.stopPropagation();
							}}
						>
							<SlIcon name='info-circle-fill' />
						</SlTooltip>
						MUID: <span className='user-drawer__muid-value'>{selectedUser}</span>
					</span>
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
							Object.entries(traits).map(([name, value]) => {
								if (typeof value === 'object') {
									return <DrawerNestedTraits name={name} value={value} indentation={1} />;
								} else {
									return (
										<DrawerTrait name={name} value={value} isParent={false} isIndented={false} />
									);
								}
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
								const source = connections.find((c) => c.id === event.connectionId);
								const logo = <LittleLogo code={source?.connector.code} path={CONNECTORS_ASSETS_PATH} />;
								return (
									<div className='user-drawer__event' key={event.sentAt}>
										<div className='user-drawer__event-head'>
											<Link path={`connections/${source.id}/actions`}>
												<div className='user-drawer__event-logo'>{logo}</div>
											</Link>
											<div className='user-drawer__event-type'>{event.type}</div>
										</div>
										<div className='user-drawer__event-sent-at'>
											{toJSDate(event.sentAt).toLocaleString()}
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
								const logo = (
									<LittleLogo code={connection?.connector.code} path={CONNECTORS_ASSETS_PATH} />
								);
								return (
									<div className='user-drawer__identity' key={identity.lastChangeTime}>
										<div className='user-drawer__identity-head'>
											<SlTooltip className='user-drawer__action' placement='left' hoist>
												<div slot='content'>
													Imported from action{' '}
													<span className='user-drawer__identity-action-link'>
														<Link
															path={`connections/${connection.id}/actions/edit/${identity.action}`}
														>
															{identity.action}
														</Link>
													</span>
												</div>
												<Link
													path={`connections/${connection.id}/actions`}
													className='user-drawer__identity-connection'
												>
													<div className='user-drawer__identity-connection-logo'>{logo}</div>
													<div className='user-drawer__identity-connection-name'>
														{connection.name}
													</div>
												</Link>
											</SlTooltip>
											<div className='user-drawer__identity-date'>
												{toJSDate(identity.lastChangeTime).toLocaleString()}
											</div>
										</div>
										<div className='user-drawer__identity-info'>
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

interface DrawerTraitProps {
	name: string;
	value: any;
	isParent: boolean;
	isIndented: boolean;
	isExpanded?: boolean;
	setIsExpanded?: React.Dispatch<React.SetStateAction<boolean>>;
}

const DrawerTrait = ({ name, value, isParent, isIndented, isExpanded, setIsExpanded }: DrawerTraitProps) => {
	const preview = useMemo(() => {
		if (!isParent) {
			return '';
		}
		let p: string = '';
		const values = Object.values(value);
		for (let i = 0; i < values.length; i++) {
			const v = values[i];
			if (typeof v === 'object') {
				p += '...';
			} else {
				p += String(v);
			}
			const isLastValue = i === values.length - 1;
			if (!isLastValue) {
				p += ', ';
			}
		}
		return p;
	}, [isParent, value]);

	return (
		<div
			className={`drawer-trait${isParent ? ' drawer-trait--parent' : ''}`}
			onClick={() => {
				if (isParent) {
					setIsExpanded(!isExpanded);
				}
			}}
		>
			<span className='drawer-trait__property-padding'>
				{isParent && <SlIcon className='drawer-trait__property-caret' name='caret-right-fill' />}
			</span>
			<span className='user-drawer__trait-key'>
				{isIndented && <span className='user-drawer__indentation-icon' />}
				{name}
				{!isParent && ':'}
			</span>
			{isParent ? (
				<span className='drawer-trait__preview'>
					<span className='drawer-trait__preview-overlay' />
					{preview}
				</span>
			) : (
				<span className='drawer-trait__value'>
					{value}
					<SlCopyButton
						className='drawer-trait__value-copy'
						value={value}
						copyLabel='Click to copy'
						successLabel='✓ Copied'
						errorLabel='Copying to clipboard is not supported by your browser'
						hoist={true}
					/>
				</span>
			)}
		</div>
	);
};

interface DrawerNestedTraitsProps {
	name: string;
	value: Record<string, any>;
	indentation: number;
}

const DrawerNestedTraits = ({ name, value, indentation }: DrawerNestedTraitsProps) => {
	const [isExpanded, setIsExpanded] = useState<boolean>(false);

	const isFirstLoad = useRef<boolean>(true);

	useEffect(() => {
		try {
			const v = localStorage.getItem(USERS_EXPANDED_TRAITS_KEY);
			if (v == null) {
				isFirstLoad.current = false;
				return;
			}
			let preferences = JSON.parse(v);
			if (preferences.includes(name)) {
				setIsExpanded(true);
			}
		} catch (err) {
			console.error(`cannot read the user trait preference from local storage: ${err}`);
			isFirstLoad.current = false;
			return;
		}
		isFirstLoad.current = false;
	}, []);

	useEffect(() => {
		if (isFirstLoad.current) {
			return;
		}

		try {
			let v = localStorage.getItem(USERS_EXPANDED_TRAITS_KEY);

			let p: string[] = [];
			if (v != null) {
				p = JSON.parse(v) as Array<string>;
				const isIncluded = p.includes(name);
				if (isExpanded) {
					if (isIncluded) {
						return;
					}
					p = [...p, name];
				} else {
					if (isIncluded) {
						const i = p.findIndex((p) => p === name);
						p = [...p.slice(0, i), ...p.slice(i + 1, p.length)];
					}
				}
			} else {
				if (isExpanded) {
					p = [name];
				}
			}

			localStorage.setItem(USERS_EXPANDED_TRAITS_KEY, JSON.stringify(p));
		} catch (err) {
			console.error(`cannot write the user trait preference on local storage: ${err}`);
			return;
		}
	}, [isExpanded]);

	return (
		<div className={`drawer-nested-traits${isExpanded ? ' drawer-nested-traits--expand' : ''}`}>
			<DrawerTrait
				name={name}
				value={value}
				isParent={true}
				isIndented={indentation > 1}
				isExpanded={isExpanded}
				setIsExpanded={setIsExpanded}
			/>
			<div
				className='drawer-nested-traits__sub-properties'
				style={{ '--property-indentation': `${indentation * 20}px` } as React.CSSProperties}
			>
				{isExpanded &&
					Object.entries(value).map(([name, value]) => {
						if (typeof value === 'object') {
							return <DrawerNestedTraits name={name} value={value} indentation={indentation + 1} />;
						} else {
							return <DrawerTrait name={name} value={value} isParent={false} isIndented={true} />;
						}
					})}
			</div>
		</div>
	);
};

export { UserDrawer };
