import React, { useContext, useState, useEffect, useMemo } from 'react';
import UsersContext from '../../../context/UsersContext';
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
import { UserDrawer } from './UserDrawer';
import { useUsersGrid } from './useUsersGrid';
import { UserProperty } from './Users.types';
import AppContext from '../../../context/AppContext';
import { LatestIdentityResolution } from '../../../lib/api/types/workspace';
import { RelativeTime } from '../../base/RelativeTime/RelativeTime';
import { formatNumber } from '../../../utils/formatNumber';

const UsersList = () => {
	const [selectedUser, setSelectedUser] = useState<string>('');
	const [isLoadingIdentityResolution, setIsLoadingIdentityResolution] = useState<boolean>(false);
	const [askRunIRConfirmation, setAskResolveIdentitiesConfirmation] = useState<boolean>(false);
	const [secondsSinceIRStart, setSecondsSinceIRStart] = useState<number>();
	const [latestIRExecutionEnd, setLastIRExecutionEnd] = useState<string>();

	const { api, handleError } = useContext(AppContext);
	const { users, usersTotal, usersProperties, isLoading, fetchUsers } = useContext(UsersContext);
	const { usersRows, userColumns } = useUsersGrid(users, usersProperties, selectedUser, (id: string) =>
		setSelectedUser(id),
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
		const used: UserProperty[] = [];
		for (const p of usersProperties) {
			if (p.isUsed) {
				used.push(p);
			}
		}
		return used;
	}, [usersProperties]);

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

		if (secondsSinceIRStart != null && sinceStart == null) {
			// identity resolution is concluded. Reload the users list.
			fetchUsers();
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
		const updatedProps: UserProperty[] = [];
		for (const p of usersProperties) {
			const cp = { ...p };
			if (cp.name === name) {
				cp.isUsed = !cp.isUsed;
			}
			updatedProps.push(cp);
		}
		localStorage.setItem('meergo_ui_users_properties', JSON.stringify(updatedProps));
		fetchUsers();
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
			fetchUsers();
		}, 300);
	};

	return (
		<div className='users-list'>
			<div className='route-content'>
				<Toolbar>
					<SlDropdown stayOpenOnSelect={true} className='users-list__toggle-columns'>
						<SlButton slot='trigger' variant='default'>
							<SlIcon slot='prefix' name='layout-three-columns' />
							Toggle columns
						</SlButton>
						<SlMenu>
							{usersProperties.map((p) => {
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
						className={`users-list__identity-resolution${!secondsSinceIRStart && !latestIRExecutionEnd && !isLoadingIdentityResolution ? ' users-list__identity-resolution--is-first-execution' : ''}`}
					>
						<SlButton
							onClick={() => setAskResolveIdentitiesConfirmation(true)}
							variant='primary'
							disabled={isLoadingIdentityResolution || secondsSinceIRStart != null}
							className='users-list__identity-resolution-button'
						>
							{isLoadingIdentityResolution || secondsSinceIRStart ? (
								<SlSpinner className='users-list__identity-resolution-spinner' slot='prefix' />
							) : (
								<SlIcon slot='prefix' name='play' />
							)}
							{secondsSinceIRStart ? 'Identity Resolution' : 'Resolve identities'}
						</SlButton>
						<span className='users-list__identity-resolution-progress'>
							{secondsSinceIRStart ? (
								<div className='users-list__identity-resolution-since-start'>{`Progress: ${String(secondsSinceIRStart)}s`}</div>
							) : latestIRExecutionEnd ? (
								<div className='users-list__identity-resolution-end-time'>
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
				<div className='users-list__content'>
					<div className='users-list__grid-container'>
						<Grid
							columns={userColumns}
							rows={usersRows}
							isLoading={isLoading}
							noRowsMessage={'No users to show'}
						/>
						<UserDrawer selectedUser={selectedUser} setSelectedUser={setSelectedUser} />
						<div className='users-list__footer'>
							<div className='users-list__footer-total'>
								<div className='users-list__footer-found'>Found {formatNumber(usersTotal)} users</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
};

export { UsersList };
