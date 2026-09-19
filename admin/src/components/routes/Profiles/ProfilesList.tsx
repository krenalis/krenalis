import React, { useCallback, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import ProfilesContext from '../../../context/ProfilesContext';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import Grid from '../../base/Grid/Grid';
import { GridRef } from '../../base/Grid/Grid.types';
import { GridKeyboardHints } from '../../base/Grid/GridKeyboardHints';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlCheckbox from '@shoelace-style/shoelace/dist/react/checkbox/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlInput from '@shoelace-style/shoelace/dist/react/input/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import { arrayMove } from '@dnd-kit/sortable';
import { ProfileDrawer } from './ProfileDrawer';
import { useProfilesGrid } from './useProfilesGrid';
import AppContext from '../../../context/AppContext';
import { LatestIdentityResolution } from '../../../lib/api/types/workspace';
import { RelativeTime } from '../../base/RelativeTime/RelativeTime';
import { formatNumber } from '../../../utils/formatNumber';
import { ProfilesPagination } from './ProfilesPagination';
import { ProfilesFilters } from './ProfilesFilters';

const ProfilesList = () => {
	const [isLoadingIdentityResolution, setIsLoadingIdentityResolution] = useState<boolean>(false);
	const [askRunIRConfirmation, setAskResolveIdentitiesConfirmation] = useState<boolean>(false);
	const [secondsSinceIRStart, setSecondsSinceIRStart] = useState<number>();
	const [latestIRExecutionEnd, setLastIRExecutionEnd] = useState<string>();
	const [columnSearch, setColumnSearch] = useState('');
	const [askSchemaResetConfirmation, setAskSchemaResetConfirmation] = useState(false);

	const { api, handleError, workspaces, selectedWorkspace } = useContext(AppContext);
	const {
		appliedFilter,
		previewProfilesFilter,
		showProfilesFilter,
		profiles,
		profilesTotal,
		profilesProperties,
		profileSchema,
		profileSchemaProperties,
		isLoading,
		isBoundaryLoading,
		profilesFirst,
		profilesLimit,
		profilesProjection,
		profileSchemaSessionID,
		resetProfilesSchema,
		hasNextPage,
		activeProfileID,
		isProfileSchemaNotAligned,
		goToProfilesPage,
		navigateProfile,
		refreshProfiles,
		reorderProfilesProperties,
		setActiveProfileID,
		setProfilesPageSize,
		updateProfilesProperties,
	} = useContext(ProfilesContext);
	const assignedRoles = useMemo(
		() => workspaces.find((workspace) => workspace.id === selectedWorkspace)?.assignedRoles,
		[workspaces, selectedWorkspace],
	);
	const countryType = profileSchemaProperties[assignedRoles?.country ?? '']?.type;
	const countryFormat =
		countryType?.kind === 'string' && countryType.semantic === 'country' ? countryType.format : undefined;
	const gridRef = useRef<GridRef>(null);
	const [filterGridActionContainer, setFilterGridActionContainer] = useState<HTMLDivElement | null>(null);
	const { profilesRows, profileColumns } = useProfilesGrid(
		profiles,
		profilesProperties,
		profilesProjection,
		assignedRoles,
		countryFormat,
		activeProfileID,
		setActiveProfileID,
	);
	const isGridKeyboardNavigationEnabled = !isLoading && profilesRows.length > 0;
	const isIdentityResolutionRunning = secondsSinceIRStart != null;
	const gridResultsLabel = `${formatNumber(profilesTotal)} ${profilesTotal === 1 ? 'profile' : 'profiles'}`;

	const inspectProfiles = useCallback(() => {
		requestAnimationFrame(() => gridRef.current?.focus());
	}, []);

	useEffect(() => {
		if (!isGridKeyboardNavigationEnabled) {
			return;
		}
		const animationFrame = requestAnimationFrame(() => {
			if (document.activeElement == null || document.activeElement === document.body) {
				gridRef.current?.focus();
			}
		});
		return () => cancelAnimationFrame(animationFrame);
	}, [isGridKeyboardNavigationEnabled, profilesRows.length]);

	useLayoutEffect(() => {
		if (activeProfileID === '') {
			return;
		}
		const animationFrame = requestAnimationFrame(() => gridRef.current?.scrollRowIntoView(activeProfileID));
		return () => cancelAnimationFrame(animationFrame);
	}, [activeProfileID, profilesFirst]);

	useEffect(() => {
		const intervalID = setInterval(() => {
			handleIdentityResolutionExecution();
		}, 10000);

		handleIdentityResolutionExecution();

		return () => {
			clearInterval(intervalID);
		};
	}, [api, handleError]);

	const usedProperties = useMemo(
		() => profilesProperties.filter((property) => property.isUsed),
		[profilesProperties],
	);
	const normalizedColumnSearch = columnSearch.trim().toLocaleLowerCase();
	const matchingProfileProperties = useMemo(
		() =>
			profilesProperties.filter(
				(property) =>
					normalizedColumnSearch === '' ||
					property.label.toLocaleLowerCase().includes(normalizedColumnSearch) ||
					property.name.toLocaleLowerCase().includes(normalizedColumnSearch),
			),
		[normalizedColumnSearch, profilesProperties],
	);
	const handleIdentityResolutionExecution = async () => {
		let res: LatestIdentityResolution;
		try {
			res = await api.workspaces.latestIdentityResolution();
		} catch (err) {
			handleError(err);
			return;
		}
		const startTime = res.startTime;
		const endTime = res.endTime;

		let sinceStart: number | undefined;
		let end: string | undefined;
		if (startTime != null && endTime == null) {
			const st = new Date(startTime);
			const now = new Date();
			sinceStart = Math.ceil((now.getTime() - st.getTime()) / 1000);
		} else if (startTime != null && endTime !== null) {
			end = endTime;
		}

		setSecondsSinceIRStart(sinceStart);
		setLastIRExecutionEnd(end);
	};

	const onToggleColumn = (name: string) => {
		const isLastUsed = usedProperties.length === 1 && usedProperties[0].name === name;
		if (isLastUsed) {
			// Prevent the user from hiding all the columns.
			return;
		}
		const updatedProperties = profilesProperties.map((property) => ({
			...property,
			isUsed: property.name === name ? !property.isUsed : property.isUsed,
		}));
		updateProfilesProperties(updatedProperties);
	};

	const onSortColumn = (overColumnKey: string, movedColumnKey: string) => {
		const overColumnIndex = profilesProperties.findIndex((property) => property.name === overColumnKey);
		const movedColumnIndex = profilesProperties.findIndex((property) => property.name === movedColumnKey);
		if (overColumnIndex === -1 || movedColumnIndex === -1 || overColumnIndex === movedColumnIndex) {
			return;
		}
		reorderProfilesProperties(arrayMove(profilesProperties, movedColumnIndex, overColumnIndex));
	};

	const onStartIdentityResolution = async () => {
		setIsLoadingIdentityResolution(true);
		setAskResolveIdentitiesConfirmation(false);
		setSecondsSinceIRStart(undefined);
		setLastIRExecutionEnd(undefined);
		try {
			await api.workspaces.startIdentityResolution();
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsLoadingIdentityResolution(false);
			}, 300);
			return;
		}
		setTimeout(() => {
			setIsLoadingIdentityResolution(false);
			handleIdentityResolutionExecution();
		}, 300);
	};

	return (
		<div className='profiles-list'>
			{isProfileSchemaNotAligned && (
				<div className='profiles-list__stale-notice' role='status'>
					<span>The profile schema is no longer compatible with this view.</span>
					<SlButton size='small' onClick={() => setAskSchemaResetConfirmation(true)}>
						Reset view
					</SlButton>
				</div>
			)}
			<div className='profiles-list__page-header'>
				<div>
					<h1>Profiles</h1>
				</div>
				<div className='profiles-list__identity-resolution'>
					<SlButton
						onClick={() => setAskResolveIdentitiesConfirmation(true)}
						variant='primary'
						disabled={isLoadingIdentityResolution || isIdentityResolutionRunning}
						className='profiles-list__identity-resolution-button'
					>
						{isLoadingIdentityResolution || isIdentityResolutionRunning ? (
							<SlSpinner className='profiles-list__identity-resolution-spinner' slot='prefix' />
						) : (
							<SlIcon slot='prefix' name='play' />
						)}
						{isIdentityResolutionRunning ? 'Identity Resolution' : 'Run Profile Unification'}
					</SlButton>
					{(isIdentityResolutionRunning || latestIRExecutionEnd) && (
						<div className='profiles-list__identity-resolution-progress'>
							{isIdentityResolutionRunning ? (
								<div className='profiles-list__identity-resolution-since-start'>{`Progress: ${String(secondsSinceIRStart)}s`}</div>
							) : (
								<div className='profiles-list__identity-resolution-end-time'>
									<span>Latest Identity Resolution:</span>
									<RelativeTime date={latestIRExecutionEnd} />
								</div>
							)}
						</div>
					)}
				</div>
			</div>
			<div className='profiles-list__content'>
				{profileSchema != null && (
					<ProfilesFilters
						key={`${selectedWorkspace}:${profileSchemaSessionID}`}
						appliedFilter={appliedFilter}
						gridActionContainer={filterGridActionContainer}
						onInspectProfiles={inspectProfiles}
						onPreview={previewProfilesFilter}
						onShow={showProfilesFilter}
						profilesTotal={isLoading ? undefined : profilesTotal}
						schema={profileSchema}
						schemaProperties={profileSchemaProperties}
					/>
				)}
				<div className='grid-keyboard-hints-layout profiles-list__layout'>
					<div className='profiles-list__card grid-keyboard-hints-layout__grid'>
						<div className='profiles-list__toolbar'>
							<div className='profiles-list__grid-summary-group'>
								<div className='profiles-list__grid-summary'>
									<span className='profiles-list__grid-summary-content'>
										<SlIcon name='people' aria-hidden='true' />
										<span>{gridResultsLabel}</span>
									</span>
									<span className='profiles-list__grid-summary-sweep' aria-hidden='true'>
										<SlIcon name='people' />
										<span>{gridResultsLabel}</span>
									</span>
								</div>
								<div ref={setFilterGridActionContainer} className='profiles-list__filter-grid-action' />
							</div>
							<div className='profiles-list__toolbar-actions'>
								<SlButton size='small' onClick={refreshProfiles} disabled={isLoading}>
									<SlIcon slot='prefix' name='arrow-clockwise' />
									Refresh
								</SlButton>
								<SlDropdown
									stayOpenOnSelect={true}
									className='profiles-list__toggle-columns'
									placement='bottom-end'
									distance={8}
								>
									<SlButton className='profiles-list__toolbar-button' slot='trigger' size='small'>
										<SlIcon slot='prefix' name='layout-three-columns' />
										Columns
									</SlButton>
									<div className='profiles-list__column-chooser'>
										<div className='profiles-list__column-chooser-header'>
											<SlInput
												className='profiles-list__column-chooser-search'
												size='small'
												label='Search columns'
												placeholder='Search columns...'
												clearable
												value={columnSearch}
												onSlInput={(event: any) => setColumnSearch(event.target.value)}
											>
												<SlIcon slot='prefix' name='search' />
												<SlIcon slot='clear-icon' name='backspace' />
											</SlInput>
										</div>
										<div
											className='profiles-list__column-chooser-list'
											role='group'
											aria-label='Profile columns'
										>
											{matchingProfileProperties.map((property) => {
												const isLastUsed =
													usedProperties.length === 1 &&
													usedProperties[0].name === property.name;
												return (
													<SlCheckbox
														className='profiles-list__column-chooser-option'
														key={property.name}
														onSlChange={() => onToggleColumn(property.name)}
														checked={property.isUsed}
														disabled={isLastUsed}
													>
														{property.label}
													</SlCheckbox>
												);
											})}
											{matchingProfileProperties.length === 0 && (
												<div className='profiles-list__column-chooser-empty' role='status'>
													No columns found
												</div>
											)}
										</div>
									</div>
								</SlDropdown>
							</div>
						</div>
						<Grid
							ref={gridRef}
							className='profiles-list__grid'
							gridID='profiles-grid'
							ariaLabel='Profiles'
							activeRowID={activeProfileID}
							columns={profileColumns}
							onSortColumn={onSortColumn}
							rows={profilesRows}
							isLoading={isLoading}
							onNavigateActiveRow={navigateProfile}
							loadingText='Loading profiles'
							noRowsIcon='people'
							noRowsMessage='No profiles to show'
						/>
						<ProfileDrawer />
					</div>
					{(profilesRows.length > 0 || profilesFirst > 0) && (
						<footer className='profiles-list__footer'>
							<ProfilesPagination
								first={profilesFirst}
								hasNext={hasNextPage}
								isLoading={isLoading}
								limit={profilesLimit}
								onFirstChange={goToProfilesPage}
								onLimitChange={setProfilesPageSize}
								profileCount={profiles.length}
								total={profilesTotal}
							>
								<GridKeyboardHints
									canExpand={false}
									disabled={!isGridKeyboardNavigationEnabled || isBoundaryLoading}
									navigationAriaLabel='Use Up and Down arrow keys to navigate profiles.'
									navigationLabel='Navigate profiles'
								/>
							</ProfilesPagination>
						</footer>
					)}
				</div>
			</div>
			{askSchemaResetConfirmation && (
				<AlertDialog
					isOpen={askSchemaResetConfirmation}
					onClose={() => setAskSchemaResetConfirmation(false)}
					title='Reset profile view?'
					actions={
						<>
							<SlButton onClick={() => setAskSchemaResetConfirmation(false)}>Cancel</SlButton>
							<SlButton
								variant='primary'
								onClick={() => {
									setAskSchemaResetConfirmation(false);
									resetProfilesSchema();
								}}
							>
								Reset view
							</SlButton>
						</>
					}
				>
					<p>
						This reloads the profile schema and clears the applied filter, draft, filter history, and
						selection.
					</p>
				</AlertDialog>
			)}
			<AlertDialog
				isOpen={askRunIRConfirmation}
				onClose={() => setAskResolveIdentitiesConfirmation(false)}
				title='Processing time notice'
				actions={
					<>
						<SlButton onClick={() => setAskResolveIdentitiesConfirmation(false)}>Cancel</SlButton>
						<SlButton variant='primary' onClick={onStartIdentityResolution}>
							Run Profile Unification
						</SlButton>
					</>
				}
			>
				<p>
					The time it takes to resolve the identities can vary significantly, from seconds to hours, depending
					on the size of user data.
				</p>
			</AlertDialog>
		</div>
	);
};

export { ProfilesList };
