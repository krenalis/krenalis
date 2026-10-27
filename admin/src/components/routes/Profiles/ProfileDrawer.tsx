import React, { useState, useEffect, useLayoutEffect, useContext, useId, useMemo, useRef } from 'react';
import { useProfileDrawer } from './useProfileDrawer';
import { ProfileTab } from './Profiles.types';
import AppContext from '../../../context/AppContext';
import ProfilesContext from '../../../context/ProfilesContext';
import WaTab from '@awesome.me/webawesome/dist/react/tab/index.js';
import WaAvatar from '@awesome.me/webawesome/dist/react/avatar/index.js';
import WaIcon from '@awesome.me/webawesome/dist/react/icon/index.js';
import WaTabGroup from '@awesome.me/webawesome/dist/react/tab-group/index.js';
import WaCopyButton from '@awesome.me/webawesome/dist/react/copy-button/index.js';
import WaTabPanel from '@awesome.me/webawesome/dist/react/tab-panel/index.js';
import WaButton from '@awesome.me/webawesome/dist/react/button/index.js';
import WaSpinner from '@awesome.me/webawesome/dist/react/spinner/index.js';
import WaTooltip from '@awesome.me/webawesome/dist/react/tooltip/index.js';
import toJSDate from '../../../utils/toJSDate';
import { Link } from '../../base/Link/Link';
import { PROFILES_EXPANDED_ATTRIBUTES_KEY, PROFILES_TAB_KEY } from '../../../constants/storage';
import LittleLogo from '../../base/LittleLogo/LittleLogo';
import { CONNECTORS_ASSETS_PATH } from '../../../constants/paths';

// The panel slides in from the end of the profiles grid.
const showKeyframes = [
	{ opacity: 0, translate: '100%' },
	{ opacity: 1, translate: '0' },
];
const animationOptions = { duration: 250, easing: 'ease' };

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

	const isOpen = selectedProfile !== '';
	const drawerRef = useRef<HTMLDivElement>(null);
	const panelRef = useRef<HTMLDivElement>(null);
	const triggerRef = useRef<Element>(null);
	const isFirstLayout = useRef(true);
	const drawerID = useId();

	// The drawer is contained in the profiles grid: it has no overlay and does
	// not close on Escape, and the page stays interactive while it is open, so
	// that another profile can be selected. It is shown and hidden here, rather
	// than by React, so that it stays visible while it slides out.
	useLayoutEffect(() => {
		const drawer = drawerRef.current;
		const panel = panelRef.current;
		if (isFirstLayout.current) {
			isFirstLayout.current = false;
			drawer.hidden = !isOpen;
			return;
		}
		panel.getAnimations().forEach((animation) => animation.cancel());
		if (isOpen) {
			triggerRef.current = document.activeElement;
			drawer.hidden = false;
			requestAnimationFrame(() => panel.focus({ preventScroll: true }));
			panel.animate(showKeyframes, animationOptions);
			return;
		}
		if (panel.contains(document.activeElement)) {
			(document.activeElement as HTMLElement).blur();
		}
		panel
			.animate([...showKeyframes].reverse(), animationOptions)
			.finished.then(() => {
				drawer.hidden = true;
				const trigger = triggerRef.current;
				if (trigger instanceof HTMLElement) {
					setTimeout(() => trigger.focus());
				}
			})
			.catch(() => {
				// The animation has been canceled because the drawer opens again.
			});
	}, [isOpen]);

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

	const onClose = () => {
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
		<WaSpinner
			style={
				{
					fontSize: '3rem',
					'--track-width': '6px',
				} as React.CSSProperties
			}
		></WaSpinner>
	);

	return (
		<div className={`profile-drawer${isLoading ? ' profile-drawer--loading' : ''}`} ref={drawerRef}>
			<div className='profile-drawer__panel' ref={panelRef} role='dialog' aria-label='Profile' tabIndex={0}>
				<div className='profile-drawer__header'>
					<WaButton className='profile-drawer__close icon-button' appearance='plain' onClick={onClose}>
						<WaIcon name='x-lg' label='Close' />
					</WaButton>
				</div>
				<div className='profile-drawer__body'>
					<div className='profile-drawer__navigation'>
						<WaButton className='icon-button' appearance='plain' onClick={() => onNavigate('previous')}>
							<WaIcon name='chevron-left' label='Previous profile' />
						</WaButton>
						<WaButton className='icon-button' appearance='plain' onClick={() => onNavigate('next')}>
							<WaIcon name='chevron-right' label='Next profile' />
						</WaButton>
					</div>
					<div className='profile-drawer__top-section'>
						<WaAvatar
							className='profile-drawer__image'
							image={profileImage != null ? String(profileImage) : ''}
						>
							<WaIcon slot='icon' name='person-fill' />
						</WaAvatar>
						<div className='profile-drawer__profile-properties'>
							<span className='profile-drawer__first-name'>
								{profileFirstName != null ? profileFirstName : ''}
							</span>{' '}
							<span className='profile-drawer__last-name'>
								{profileLastName != null ? profileLastName : ''}
							</span>
							<div className='profile-drawer__information'>
								{profileExtra != null ? profileExtra : ''}
							</div>
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
							<span className='profile-drawer__kpid'>
								<WaTooltip for={`${drawerID}-kpid`}>Krenalis Profile ID</WaTooltip>
								<WaIcon id={`${drawerID}-kpid`} name='info-circle-fill' />
								KPID: <span className='profile-drawer__kpid-value'>{selectedProfile}</span>
							</span>
						</div>
					</div>
					<WaTabGroup active={selectedTab} onWaTabShow={onSelectTab}>
						<WaTab slot='nav' panel='attributes'>
							Attributes
						</WaTab>
						<WaTab slot='nav' panel='events'>
							Events
						</WaTab>
						<WaTab slot='nav' panel='identities'>
							Identities
						</WaTab>
						<WaTabPanel name='attributes'>
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
						</WaTabPanel>
						<WaTabPanel name='events'>
							<div
								className={`profile-drawer__events${selectedTab === 'events' ? ' profile-drawer__events--selected' : ''}`}
							>
								{isLoading ? (
									spinner
								) : events && events.length > 0 ? (
									events.map((event) => {
										const source = connections.find((c) => c.id === event.connectionId);
										const logo = (
											<LittleLogo code={source?.connector.code} path={CONNECTORS_ASSETS_PATH} />
										);
										return (
											<div className='profile-drawer__event' key={event.sentAt}>
												<div className='profile-drawer__event-head'>
													<Link path={`connections/${source.id}/pipelines`}>
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
									<div className='profile-drawer__no-events'>
										No events associated to this profile
									</div>
								)}
							</div>
						</WaTabPanel>
						<WaTabPanel name='identities'>
							<div
								className={`profile-drawer__identities${selectedTab === 'identities' ? ' profile-drawer__identities--selected' : ''}`}
							>
								{isLoading ? (
									spinner
								) : identities && identities.length > 0 ? (
									identities.map((identity) => {
										const connection = connections.find((c) => c.id === identity.connection);
										const logo = (
											<LittleLogo
												code={connection?.connector.code}
												path={CONNECTORS_ASSETS_PATH}
											/>
										);
										return (
											<div className='profile-drawer__identity' key={identity.updatedAt}>
												<div className='profile-drawer__identity-head'>
													<WaTooltip
														className='profile-drawer__action'
														for={`${drawerID}-${identity.updatedAt}`}
														placement='left'
													>
														Imported from pipeline{' '}
														<span className='profile-drawer__identity-pipeline-link'>
															<Link
																path={`connections/${connection.id}/pipelines/edit/${identity.pipeline}`}
															>
																{identity.pipeline}
															</Link>
														</span>
													</WaTooltip>
													<Link
														id={`${drawerID}-${identity.updatedAt}`}
														path={`connections/${connection.id}/pipelines`}
														className='profile-drawer__identity-connection'
													>
														<div className='profile-drawer__identity-connection-logo'>
															{logo}
														</div>
														<div className='profile-drawer__identity-connection-name'>
															{connection.name}
														</div>
													</Link>
													<div className='profile-drawer__identity-updated-at'>
														{toJSDate(identity.updatedAt).toLocaleString()}
													</div>
												</div>
												<div className='profile-drawer__identity-info'>
													{'userId' in identity && (
														<div className='profile-drawer__user-id'>
															{connection.connector.terms.userID}:{' '}
															<code>{identity.userId}</code>
														</div>
													)}
													{'anonymousIds' in identity && (
														<div className='profile-drawer__identity-anonymous-ids'>
															Anonymous IDs:{' '}
															<code>{identity.anonymousIds.join(', ')}</code>
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
						</WaTabPanel>
					</WaTabGroup>
				</div>
			</div>
		</div>
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
				{isParent && <WaIcon className='drawer-attribute__property-caret' name='caret-right-fill' />}
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
					<WaCopyButton
						className='drawer-attribute__value-copy'
						value={value}
						copyLabel='Click to copy'
						successLabel='✓ Copied'
						errorLabel='Copying to clipboard is not supported by your browser'
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
