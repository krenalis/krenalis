import React, { useState, useEffect, useContext, useMemo, useRef } from 'react';
import { useProfileDrawer } from './useProfileDrawer';
import { ProfileTab } from './Profiles.types';
import AppContext from '../../../context/AppContext';
import ProfilesContext from '../../../context/ProfilesContext';
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
import { PROFILES_EXPANDED_ATTRIBUTES_KEY, PROFILES_TAB_KEY } from '../../../constants/storage';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

interface ProfileDrawerProps {
	selectedProfile: string;
	setSelectedProfile: React.Dispatch<React.SetStateAction<string>>;
}

const ProfileDrawer = ({ selectedProfile, setSelectedProfile }: ProfileDrawerProps) => {
	const [selectedTab, setSelectedTab] = useState<ProfileTab>();

	const { connections, workspaces, selectedWorkspace } = useContext(AppContext);
	const { profileIDList } = useContext(ProfilesContext);
	const { isLoading, attributes, events, identities } = useProfileDrawer(selectedProfile, selectedTab);

	const workspace = useMemo(
		() => workspaces.find((w) => w.id === selectedWorkspace),
		[workspaces, selectedWorkspace],
	);

	useEffect(() => {
		let tab: string;
		try {
			tab = localStorage.getItem(PROFILES_TAB_KEY);
		} catch (err) {
			setSelectedTab('attributes');
			return;
		}
		if (tab === 'attributes' || tab === 'events' || tab === 'identities') {
			setSelectedTab(tab);
			return;
		}
		setSelectedTab('attributes');
	}, [selectedProfile]);

	useEffect(() => {
		try {
			localStorage.setItem(PROFILES_TAB_KEY, selectedTab);
		} catch (err) {
			console.error(`cannot write the profile tab preference on local storage: ${err}`);
			return;
		}
	}, [selectedTab]);

	const onNavigate = async (direction: 'previous' | 'next') => {
		const i = profileIDList.findIndex((id) => id === selectedProfile);
		let newProfileID: string;
		if (direction === 'previous') {
			if (i - 1 < 0) {
				// if the index is overflowing the start of the profiles list,
				// select the last profile.
				newProfileID = profileIDList[profileIDList.length - 1];
			} else {
				// select the previous profile.
				newProfileID = profileIDList[i - 1];
			}
		} else if (direction === 'next') {
			if (i + 1 >= profileIDList.length) {
				// if the index is overflowing the end of the profiles list, select
				// the first profile.
				newProfileID = profileIDList[0];
			} else {
				// select the next profile.
				newProfileID = profileIDList[i + 1];
			}
		}
		setSelectedProfile(newProfileID);
	};

	const onSelectTab = (e: any) => {
		setSelectedTab(e.detail.name);
	};

	const onClose = (e: any) => {
		if (
			e.target.classList.contains('drawer-attributes__value-copy') ||
			e.target.classList.contains('profile-drawer__action')
		) {
			e.stopPropagation();
			return;
		}
		setSelectedProfile('');
	};

	let profileImage: string | number | undefined;
	let profileFirstName: string | number | undefined;
	let profileLastName: string | number | undefined;
	let profileExtra: string | number | undefined;
	if (attributes && Object.keys(attributes).length > 0) {
		function getValueFromPath(path: string): string | number | undefined {
			if (path == '') {
				return undefined;
			}
			let v: any = attributes;
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
		profileImage = getValueFromPath(workspace.uiPreferences.profile.image);
		profileFirstName = getValueFromPath(workspace.uiPreferences.profile.firstName);
		profileLastName = getValueFromPath(workspace.uiPreferences.profile.lastName);
		profileExtra = getValueFromPath(workspace.uiPreferences.profile.extra);
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
			className={`profile-drawer${isLoading ? ' profile-drawer--loading' : ''}`}
			open={selectedProfile !== ''}
			contained
			style={{ '--size': '600px' } as React.CSSProperties}
			onSlHide={onClose}
		>
			<div className='profile-drawer__navigation'>
				<SlIconButton name='chevron-left' onClick={() => onNavigate('previous')} />
				<SlIconButton name='chevron-right' onClick={() => onNavigate('next')} />
			</div>
			<div className='profile-drawer__top-section'>
				<SlAvatar className='profile-drawer__image' image={String(profileImage) || ''} />
				<div className='profile-drawer__profile-properties'>
					<span className='profile-drawer__first-name'>{profileFirstName || ''}</span>{' '}
					<span className='profile-drawer__last-name'>{profileLastName || ''}</span>
					<div className='profile-drawer__information'>{profileExtra || ''}</div>
					{profileImage == null &&
						profileFirstName == null &&
						profileLastName == null &&
						profileExtra == null && (
							<div className='profile-drawer__customize'>
								You can customize the properties to display in the{' '}
								<Link path='settings/general'>
									<span className='profile-drawer__customize-link'>settings</span>
								</Link>
							</div>
						)}
					<span className='profile-drawer__mpid'>
						<SlTooltip
							content='Meergo profile ID'
							onSlHide={(e) => {
								// Prevent the event from bubbling up and
								// causing the drawer to close.
								e.stopPropagation();
							}}
						>
							<SlIcon name='info-circle-fill' />
						</SlTooltip>
						MPID: <span className='profile-drawer__mpid-value'>{selectedProfile}</span>
					</span>
				</div>
			</div>
			<SlTabGroup onSlTabShow={onSelectTab}>
				<SlTab slot='nav' panel='attributes' active={selectedTab === 'attributes'}>
					Attributes
				</SlTab>
				<SlTab slot='nav' panel='events' active={selectedTab === 'events'}>
					Events
				</SlTab>
				<SlTab slot='nav' panel='identities' active={selectedTab === 'identities'}>
					Identities
				</SlTab>
				<SlTabPanel name='attributes'>
					<div className='profile-drawer__attributes'>
						{isLoading ? (
							spinner
						) : attributes && Object.keys(attributes).length > 0 ? (
							Object.entries(attributes).map(([name, value]) => {
								if (typeof value === 'object') {
									return <DrawerNestedAttributes name={name} value={value} indentation={1} />;
								} else {
									return (
										<DrawerAttribute
											name={name}
											value={value}
											isParent={false}
											isIndented={false}
										/>
									);
								}
							})
						) : (
							<div className='profile-drawer__no-attributes'>
								No attributes associated to this profile
							</div>
						)}
					</div>
				</SlTabPanel>
				<SlTabPanel name='events'>
					<div
						className={`profile-drawer__events${selectedTab === 'events' ? ' profile-drawer__events--selected' : ''}`}
					>
						{isLoading ? (
							spinner
						) : events && events.length > 0 ? (
							events.map((event) => {
								const source = connections.find((c) => c.id === event.connectionId);
								const logo = <LittleLogo code={source?.connector.code} path={CONNECTORS_ASSETS_PATH} />;
								return (
									<div className='profile-drawer__event' key={event.sentAt}>
										<div className='profile-drawer__event-head'>
											<Link path={`connections/${source.id}/actions`}>
												<div className='profile-drawer__event-logo'>{logo}</div>
											</Link>
											<div className='profile-drawer__event-type'>{event.type}</div>
										</div>
										<div className='profile-drawer__event-sent-at'>
											{toJSDate(event.sentAt).toLocaleString()}
										</div>
									</div>
								);
							})
						) : (
							<div className='profile-drawer__no-events'>No events associated to this profile</div>
						)}
					</div>
				</SlTabPanel>
				<SlTabPanel name='identities'>
					<div
						className={`profile-drawer__identities${selectedTab === 'identities' ? ' profile-drawer__identities--selected' : ''}`}
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
									<div className='profile-drawer__identity' key={identity.lastChangeTime}>
										<div className='profile-drawer__identity-head'>
											<SlTooltip className='profile-drawer__action' placement='left' hoist>
												<div slot='content'>
													Imported from action{' '}
													<span className='profile-drawer__identity-action-link'>
														<Link
															path={`connections/${connection.id}/actions/edit/${identity.action}`}
														>
															{identity.action}
														</Link>
													</span>
												</div>
												<Link
													path={`connections/${connection.id}/actions`}
													className='profile-drawer__identity-connection'
												>
													<div className='profile-drawer__identity-connection-logo'>
														{logo}
													</div>
													<div className='profile-drawer__identity-connection-name'>
														{connection.name}
													</div>
												</Link>
											</SlTooltip>
											<div className='profile-drawer__identity-date'>
												{toJSDate(identity.lastChangeTime).toLocaleString()}
											</div>
										</div>
										<div className='profile-drawer__identity-info'>
											{identity.id && (
												<div className='profile-drawer__identity-id'>
													{connection.connector.getIdentityIDLabel()}:{' '}
													<code>{identity.id}</code>
												</div>
											)}
											{identity.anonymousIds !== null && (
												<div className='profile-drawer__identity-anonymous-ids'>
													Anonymous IDs: <code>{identity.anonymousIds.join(', ')}</code>
												</div>
											)}
										</div>
									</div>
								);
							})
						) : (
							<div className='profile-drawer__no-identities'>
								No identities associated to this profile
							</div>
						)}
					</div>
				</SlTabPanel>
			</SlTabGroup>
		</SlDrawer>
	);
};

interface DrawerAttributeProps {
	name: string;
	value: any;
	isParent: boolean;
	isIndented: boolean;
	isExpanded?: boolean;
	setIsExpanded?: React.Dispatch<React.SetStateAction<boolean>>;
}

const DrawerAttribute = ({ name, value, isParent, isIndented, isExpanded, setIsExpanded }: DrawerAttributeProps) => {
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
			className={`drawer-attributes${isParent ? ' drawer-attributes--parent' : ''}`}
			onClick={() => {
				if (isParent) {
					setIsExpanded(!isExpanded);
				}
			}}
		>
			<span className='drawer-attribute__property-padding'>
				{isParent && <SlIcon className='drawer-attribute__property-caret' name='caret-right-fill' />}
			</span>
			<span className='profile-drawer__attribute-key'>
				{isIndented && <span className='profile-drawer__indentation-icon' />}
				{name}
				{!isParent && ':'}
			</span>
			{isParent ? (
				<span className='drawer-attribute__preview'>
					<span className='drawer-attribute__preview-overlay' />
					{preview}
				</span>
			) : (
				<span className='drawer-attribute__value'>
					{value}
					<SlCopyButton
						className='drawer-attributes__value-copy'
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

interface DrawerNestedAttributesProps {
	name: string;
	value: Record<string, any>;
	indentation: number;
}

const DrawerNestedAttributes = ({ name, value, indentation }: DrawerNestedAttributesProps) => {
	const [isExpanded, setIsExpanded] = useState<boolean>(false);

	const isFirstLoad = useRef<boolean>(true);

	useEffect(() => {
		try {
			const v = localStorage.getItem(PROFILES_EXPANDED_ATTRIBUTES_KEY);
			if (v == null) {
				isFirstLoad.current = false;
				return;
			}
			let preferences = JSON.parse(v);
			if (preferences.includes(name)) {
				setIsExpanded(true);
			}
		} catch (err) {
			console.error(`cannot read the profile attribute preference from local storage: ${err}`);
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
			let v = localStorage.getItem(PROFILES_EXPANDED_ATTRIBUTES_KEY);

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

			localStorage.setItem(PROFILES_EXPANDED_ATTRIBUTES_KEY, JSON.stringify(p));
		} catch (err) {
			console.error(`cannot write the profile attribute preference on local storage: ${err}`);
			return;
		}
	}, [isExpanded]);

	return (
		<div className={`drawer-nested-attributes${isExpanded ? ' drawer-nested-attributes--expand' : ''}`}>
			<DrawerAttribute
				name={name}
				value={value}
				isParent={true}
				isIndented={indentation > 1}
				isExpanded={isExpanded}
				setIsExpanded={setIsExpanded}
			/>
			<div
				className='drawer-nested-attributes__sub-properties'
				style={{ '--property-indentation': `${indentation * 20}px` } as React.CSSProperties}
			>
				{isExpanded &&
					Object.entries(value).map(([name, value]) => {
						if (typeof value === 'object') {
							return <DrawerNestedAttributes name={name} value={value} indentation={indentation + 1} />;
						} else {
							return <DrawerAttribute name={name} value={value} isParent={false} isIndented={true} />;
						}
					})}
			</div>
		</div>
	);
};

export { ProfileDrawer };
