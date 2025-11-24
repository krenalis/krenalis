import React, { useContext, useState, useEffect, useMemo } from 'react';
import ProfilesContext from '../../../context/ProfilesContext';
import Toolbar from '../../base/Toolbar/Toolbar';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import Grid from '../../base/Grid/Grid';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import { ProfileDrawer } from './ProfileDrawer';
import { useProfilesGrid } from './useProfilesGrid';
import { ProfileProperty } from './Profiles.types';
import AppContext from '../../../context/AppContext';
import { LatestIdentityResolution } from '../../../lib/api/types/workspace';
import { RelativeTime } from '../../base/RelativeTime/RelativeTime';
import { formatNumber } from '../../../utils/formatNumber';
import { PROFILES_PROPERTIES_KEY } from '../../../constants/storage';

const ProfilesList = () => {
	const [selectedProfile, setSelectedProfile] = useState<string>('');
	const [isLoadingIdentityResolution, setIsLoadingIdentityResolution] = useState<boolean>(false);
	const [askRunIRConfirmation, setAskResolveIdentitiesConfirmation] = useState<boolean>(false);
	const [secondsSinceIRStart, setSecondsSinceIRStart] = useState<number>();
	const [latestIRExecutionEnd, setLastIRExecutionEnd] = useState<string>();

	const { api, handleError } = useContext(AppContext);
	const { profiles, profilesTotal, profilesProperties, isLoading, fetchProfiles } = useContext(ProfilesContext);
	const { profilesRows, profileColumns } = useProfilesGrid(
		profiles,
		profilesProperties,
		selectedProfile,
		(id: string) => setSelectedProfile(id),
	);

	useEffect(() => {
		const intervalID = setInterval(() => {
			handleIdentityResolutionExecution();
		}, 10000);

		handleIdentityResolutionExecution();

		return () => {
			clearInterval(intervalID);
		};
	}, []);

	const usedProperties = useMemo(() => {
		const used: ProfileProperty[] = [];
		for (const p of profilesProperties) {
			if (p.isUsed) {
				used.push(p);
			}
		}
		return used;
	}, [profilesProperties]);

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

		let sinceStart: number | null;
		let end: string | null;
		if (startTime != null && endTime == null) {
			const st = new Date(startTime);
			const now = new Date();
			sinceStart = Math.ceil((now.getTime() - st.getTime()) / 1000);
		} else if (startTime != null && endTime !== null) {
			end = endTime;
		}

		if (secondsSinceIRStart != null && sinceStart == null) {
			// identity resolution is concluded. Reload the profiles list.
			fetchProfiles();
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
		const updatedProps: ProfileProperty[] = [];
		for (const p of profilesProperties) {
			const cp = { ...p };
			if (cp.name === name) {
				cp.isUsed = !cp.isUsed;
			}
			updatedProps.push(cp);
		}
		localStorage.setItem(PROFILES_PROPERTIES_KEY, JSON.stringify(updatedProps));
		fetchProfiles();
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
			fetchProfiles();
		}, 300);
	};

	return (
		<div className='profiles-list'>
			<div className='route-content'>
				<Toolbar>
					<SlDropdown stayOpenOnSelect={true} className='profiles-list__toggle-columns'>
						<SlButton slot='trigger' variant='default'>
							<SlIcon slot='prefix' name='layout-three-columns' />
							Toggle columns
						</SlButton>
						<SlMenu>
							{profilesProperties.map((p) => {
								const isLastUsed = usedProperties.length === 1 && usedProperties[0].name === p.name;
								return (
									<SlOption key={p.name}>
										<SlSwitch
											size='small'
											onSlChange={() => onToggleColumn(p.name)}
											checked={p.isUsed}
											disabled={isLastUsed}
										>
											{p.name}
										</SlSwitch>
									</SlOption>
								);
							})}
						</SlMenu>
					</SlDropdown>
					<div
						className={`profiles-list__identity-resolution${!secondsSinceIRStart && !latestIRExecutionEnd && !isLoadingIdentityResolution ? ' profiles-list__identity-resolution--is-first-execution' : ''}`}
					>
						<SlButton
							onClick={() => setAskResolveIdentitiesConfirmation(true)}
							variant='primary'
							disabled={isLoadingIdentityResolution || secondsSinceIRStart != null}
							className='profiles-list__identity-resolution-button'
						>
							{isLoadingIdentityResolution || secondsSinceIRStart ? (
								<SlSpinner className='profiles-list__identity-resolution-spinner' slot='prefix' />
							) : (
								<SlIcon slot='prefix' name='play' />
							)}
							{secondsSinceIRStart ? 'Identity Resolution' : 'Resolve identities'}
						</SlButton>
						<span className='profiles-list__identity-resolution-progress'>
							{secondsSinceIRStart ? (
								<div className='profiles-list__identity-resolution-since-start'>{`Progress: ${String(secondsSinceIRStart)}s`}</div>
							) : latestIRExecutionEnd ? (
								<div className='profiles-list__identity-resolution-end-time'>
									<span>Latest Identity Resolution:</span>
									<RelativeTime date={latestIRExecutionEnd} />
								</div>
							) : (
								''
							)}
						</span>
					</div>
					<AlertDialog
						isOpen={askRunIRConfirmation}
						onClose={() => setAskResolveIdentitiesConfirmation(false)}
						title='Processing time notice'
						actions={
							<>
								<SlButton onClick={() => setAskResolveIdentitiesConfirmation(false)}>Cancel</SlButton>
								<SlButton variant='primary' onClick={onStartIdentityResolution}>
									Run identity resolution
								</SlButton>
							</>
						}
					>
						<p>
							The time it takes to resolve the identities can vary significantly, from seconds to hours,
							depending on the size of user data.
						</p>
					</AlertDialog>
				</Toolbar>
				<div className='profiles-list__content'>
					<div className='profiles-list__grid-container'>
						<Grid
							columns={profileColumns}
							rows={profilesRows}
							isLoading={isLoading}
							noRowsMessage={'No profiles to show'}
						/>
						<ProfileDrawer selectedProfile={selectedProfile} setSelectedProfile={setSelectedProfile} />
						<div className='profiles-list__footer'>
							<div className='profiles-list__footer-total'>
								<div className='profiles-list__footer-found'>
									Found {formatNumber(profilesTotal)} profiles
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
};

export { ProfilesList };
