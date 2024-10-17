import React, { useContext, useState, useEffect } from 'react';
import * as icons from '../../../constants/icons';
import UsersContext from '../../../context/UsersContext';
import Toolbar from '../../base/Toolbar/Toolbar';
import AlertDialog from '../../base/AlertDialog/AlertDialog';
import Grid from '../../base/Grid/Grid';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlRelativeTime from '@shoelace-style/shoelace/dist/react/relative-time/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlSpinner from '@shoelace-style/shoelace/dist/react/spinner/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import { UserDrawer } from './UserDrawer';
import { useUsersGrid } from './useUsersGrid';
import { UserProperty } from './Users.types';
import AppContext from '../../../context/AppContext';
import { IdentityResolutionExecution } from '../../../lib/api/types/workspace';

const UsersList = () => {
	const [selectedUser, setSelectedUser] = useState<string>('');
	const [isLoadingIdentityResolution, setIsLoadingIdentityResolution] = useState<boolean>(false);
	const [askRunIRConfirmation, setAskResolveIdentitiesConfirmation] = useState<boolean>(false);
	const [secondsSinceIRStart, setSecondsSinceIRStart] = useState<number>();
	const [lastIRExecutionEnd, setLastIRExecutionEnd] = useState<string>();

	const { api, handleError, showStatus } = useContext(AppContext);
	const { users, usersCount, usersProperties, isLoading, fetchUsers } = useContext(UsersContext);
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

	const handleIdentityResolutionExecution = async () => {
		let res: IdentityResolutionExecution;
		try {
			res = await api.workspaces.identityResolutionExecution();
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

	const onResolveIdentities = async () => {
		setIsLoadingIdentityResolution(true);
		setAskResolveIdentitiesConfirmation(false);
		setSecondsSinceIRStart(undefined);
		setLastIRExecutionEnd(undefined);
		try {
			await api.workspaces.resolveIdentities();
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsLoadingIdentityResolution(false);
			}, 300);
			return;
		}
		setTimeout(() => {
			showStatus({ variant: 'success', icon: icons.OK, text: 'Identity resolution completed successfully' });
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
								return (
									<SlOption key={p.name}>
										<SlSwitch
											size='small'
											onSlChange={() => onToggleColumn(p.name)}
											checked={p.isUsed}
										>
											{p.name}
										</SlSwitch>
									</SlOption>
								);
							})}
						</SlMenu>
					</SlDropdown>
					<div
						className={`users-list__identity-resolution${!secondsSinceIRStart && !lastIRExecutionEnd && !isLoadingIdentityResolution ? ' users-list__identity-resolution--is-first-execution' : ''}`}
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
							) : lastIRExecutionEnd ? (
								<div className='users-list__identity-resolution-end-time'>
									<span>Last Identity Resolution:</span>
									<SlRelativeTime lang='en-US' date={lastIRExecutionEnd} />
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
								<SlButton variant='primary' onClick={onResolveIdentities}>
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
								<div className='users-list__footer-found'>Found {usersCount} users</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
};

export { UsersList };
