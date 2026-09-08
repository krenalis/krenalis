import React, { useState, useEffect, useContext, useMemo, useRef } from 'react';
import { useProfileDrawer } from './useProfileDrawer';
import { ProfileTab } from './Profiles.types';
import AppContext from '../../../context/AppContext';
import ProfilesContext from '../../../context/ProfilesContext';
import SlTab from '@shoelace-style/shoelace/dist/react/tab/index.js';
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
import { ProfileSchemaProperties } from './Profiles.types';
import { getProfileAttributeString, getProfilePropertyLabel } from './Profiles.helpers';
import { ProfileThumbnail } from './ProfileThumbnail';

const ProfileDrawer = () => {
	const [selectedTab, setSelectedTab] = useState<ProfileTab>();

	const { connections, workspaces, selectedWorkspace } = useContext(AppContext);
	const {
		activeProfileID,
		canNavigateNextProfile,
		canNavigatePreviousProfile,
		profileSchema,
		profilesExecutionID,
		profilesQueryKey,
		markProfileSchemaNotAligned,
		navigateProfile,
		profileSchemaProperties,
		profiles,
		setActiveProfileID,
	} = useContext(ProfilesContext);
	const {
		attributes,
		events,
		identities,
		isAttributesLoading,
		isEventsLoading,
		isIdentitiesLoading,
		isProfileMissing,
	} = useProfileDrawer(
		activeProfileID,
		selectedTab,
		profileSchema,
		`${profilesExecutionID}:${profilesQueryKey}`,
		markProfileSchemaNotAligned,
	);

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
	}, [activeProfileID]);

	useEffect(() => {
		try {
			localStorage.setItem(PROFILES_TAB_KEY, selectedTab);
		} catch (err) {
			console.error(`cannot write the profile tab preference on local storage: ${err}`);
			return;
		}
	}, [selectedTab]);

	const onSelectTab = (e: any) => {
		setSelectedTab(e.detail.name);
	};

	const summary = profiles.find((profile) => profile.kpid === activeProfileID);
	const displayedAttributes = isProfileMissing ? undefined : (attributes ?? summary?.attributes);
	const profilePhoto = getProfileAttributeString(displayedAttributes, workspace?.assignedRoles.photo ?? '');
	const profileFirstName = getProfileAttributeString(displayedAttributes, workspace?.assignedRoles.firstName ?? '');
	const profileLastName = getProfileAttributeString(displayedAttributes, workspace?.assignedRoles.lastName ?? '');
	const profileEmail = getProfileAttributeString(displayedAttributes, workspace?.assignedRoles.email ?? '');

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

	if (activeProfileID === '') {
		return null;
	}

	return (
		<aside className='profile-drawer' aria-label='Profile details'>
			<div className='profile-drawer__navigation'>
				<SlIconButton
					name='chevron-left'
					label='Previous profile'
					disabled={!canNavigatePreviousProfile}
					onClick={() => navigateProfile('previous')}
				/>
				<SlIconButton
					name='chevron-right'
					label='Next profile'
					disabled={!canNavigateNextProfile}
					onClick={() => navigateProfile('next')}
				/>
				<SlIconButton
					className='profile-drawer__close'
					name='x-lg'
					label='Close profile details'
					onClick={() => setActiveProfileID('')}
				/>
			</div>
			<div className='profile-drawer__top-section'>
				<ProfileThumbnail
					className='profile-drawer__profile-thumbnail'
					firstName={profileFirstName}
					lastName={profileLastName}
					photo={profilePhoto}
					profileID={activeProfileID}
				/>
				<div className='profile-drawer__profile-properties'>
					<span className='profile-drawer__first-name'>{profileFirstName ?? ''}</span>{' '}
					<span className='profile-drawer__last-name'>{profileLastName ?? ''}</span>
					<div className='profile-drawer__email'>{profileEmail ?? ''}</div>
					<span className='profile-drawer__kpid'>
						<SlTooltip
							content='Krenalis Profile ID'
							onSlHide={(e) => {
								// Prevent the event from bubbling up and
								// causing the drawer to close.
								e.stopPropagation();
							}}
						>
							<SlIcon name='info-circle-fill' />
						</SlTooltip>
						KPID: <span className='profile-drawer__kpid-value'>{activeProfileID}</span>
					</span>
				</div>
			</div>
			{isProfileMissing && (
				<p role='status'>This profile no longer exists. You can continue browsing or refresh the list.</p>
			)}
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
						{isAttributesLoading && attributes == null ? (
							spinner
						) : displayedAttributes && Object.keys(displayedAttributes).length > 0 ? (
							Object.entries(displayedAttributes).map(([name, value]) => {
								const path = name;
								if (typeof value === 'object') {
									return (
										<DrawerNestedAttributes
											key={path}
											name={name}
											path={path}
											profileSchemaProperties={profileSchemaProperties}
											value={value}
											indentation={1}
										/>
									);
								} else {
									return (
										<DrawerAttribute
											key={path}
											label={getProfilePropertyLabel(path, profileSchemaProperties)}
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
						{isEventsLoading ? (
							spinner
						) : events && events.length > 0 ? (
							events.map((event) => {
								const source = connections.find((c) => c.id === event.connectionId);
								const logo = <LittleLogo code={source?.connector.code} path={CONNECTORS_ASSETS_PATH} />;
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
							<div className='profile-drawer__no-events'>No events associated to this profile</div>
						)}
					</div>
				</SlTabPanel>
				<SlTabPanel name='identities'>
					<div
						className={`profile-drawer__identities${selectedTab === 'identities' ? ' profile-drawer__identities--selected' : ''}`}
					>
						{isIdentitiesLoading ? (
							spinner
						) : identities && identities.length > 0 ? (
							identities.map((identity) => {
								const connection = connections.find((c) => c.id === identity.connection);
								const logo = (
									<LittleLogo code={connection?.connector.code} path={CONNECTORS_ASSETS_PATH} />
								);
								return (
									<div className='profile-drawer__identity' key={identity.updatedAt}>
										<div className='profile-drawer__identity-head'>
											<SlTooltip className='profile-drawer__action' placement='left' hoist>
												<div slot='content'>
													Imported from pipeline{' '}
													<span className='profile-drawer__identity-pipeline-link'>
														<Link
															path={`connections/${connection.id}/pipelines/edit/${identity.pipeline}`}
														>
															{identity.pipeline}
														</Link>
													</span>
												</div>
												<Link
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
											</SlTooltip>
											<div className='profile-drawer__identity-updated-at'>
												{toJSDate(identity.updatedAt).toLocaleString()}
											</div>
										</div>
										<div className='profile-drawer__identity-info'>
											{'userId' in identity && (
												<div className='profile-drawer__user-id'>
													{connection.connector.terms.userID}: <code>{identity.userId}</code>
												</div>
											)}
											{'anonymousIds' in identity && (
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
		</aside>
	);
};

interface DrawerAttributeProps {
	label: string;
	value: any;
	isParent: boolean;
	isIndented: boolean;
	isExpanded?: boolean;
	setIsExpanded?: React.Dispatch<React.SetStateAction<boolean>>;
}

const DrawerAttribute = ({ label, value, isParent, isIndented, isExpanded, setIsExpanded }: DrawerAttributeProps) => {
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
				{label}
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
						className='drawer-attribute__value-copy'
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
	path: string;
	profileSchemaProperties: ProfileSchemaProperties;
	value: Record<string, any>;
	indentation: number;
}

const DrawerNestedAttributes = ({
	name,
	path,
	profileSchemaProperties,
	value,
	indentation,
}: DrawerNestedAttributesProps) => {
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
				label={getProfilePropertyLabel(path, profileSchemaProperties)}
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
						const childPath = `${path}.${name}`;
						if (typeof value === 'object') {
							return (
								<DrawerNestedAttributes
									key={childPath}
									name={name}
									path={childPath}
									profileSchemaProperties={profileSchemaProperties}
									value={value}
									indentation={indentation + 1}
								/>
							);
						} else {
							return (
								<DrawerAttribute
									key={childPath}
									label={getProfilePropertyLabel(childPath, profileSchemaProperties)}
									value={value}
									isParent={false}
									isIndented={true}
								/>
							);
						}
					})}
			</div>
		</div>
	);
};

export { ProfileDrawer };
