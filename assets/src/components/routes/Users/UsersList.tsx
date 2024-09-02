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
	const [askRunIRConfirmation, setAskRunIRConfirmation] = useState<boolean>(false);
	const [secondsSinceIRExecutionStart, setSecondsSinceIRExecutionStart] = useState<number>();
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

		if (secondsSinceIRExecutionStart != null && sinceStart == null) {
			// identity resolution is concluded. Reload the users list.
			fetchUsers();
		}

		setSecondsSinceIRExecutionStart(sinceStart);
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

	const onRunIdentityResolution = async () => {
		setIsLoadingIdentityResolution(true);
		setAskRunIRConfirmation(false);
		setSecondsSinceIRExecutionStart(undefined);
		setLastIRExecutionEnd(undefined);
		try {
			await api.workspaces.runIdentityResolution();
		} catch (err) {
			setTimeout(() => {
				handleError(err);
				setIsLoadingIdentityResolution(false);
			}, 300);
			return;
		}
		setTimeout(() => {
			showStatus({ variant: 'success', icon: icons.OK, text: 'Identity resolution completed succesfully' });
			setIsLoadingIdentityResolution(false);
			handleIdentityResolutionExecution();
			fetchUsers();
		}, 300);
	};

	return (
		<div className='users-list'>
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
									<SlSwitch size='small' onSlChange={() => onToggleColumn(p.name)} checked={p.isUsed}>
										{p.name}
									</SlSwitch>
								</SlOption>
							);
						})}
					</SlMenu>
				</SlDropdown>
				<div className='users-list__identity-resolution'>
					<SlButton
						onClick={() => setAskRunIRConfirmation(true)}
						variant='primary'
						disabled={isLoadingIdentityResolution || secondsSinceIRExecutionStart != null}
						size='small'
						className='users-list__identity-resolution-button'
					>
						{isLoadingIdentityResolution || secondsSinceIRExecutionStart ? (
							<SlSpinner className='users-list__identity-resolution-spinner' slot='prefix' />
						) : (
							<SlIcon slot='prefix' name='play' />
						)}
						{secondsSinceIRExecutionStart ? 'Identity Resolution' : 'Run Identity Resolution'}
					</SlButton>
					<span className='users-list__identity-resolution-progress'>
						{secondsSinceIRExecutionStart ? (
							<div className='users-list__identity-resolution-since-start'>{`Progress: ${String(secondsSinceIRExecutionStart)}s`}</div>
						) : lastIRExecutionEnd ? (
							<div className='users-list__identity-resolution-end-time'>
								<span>Last execution:</span>
								<SlRelativeTime lang='en-US' date={lastIRExecutionEnd} />
							</div>
						) : (
							''
						)}
					</span>
				</div>
				<AlertDialog
					isOpen={askRunIRConfirmation}
					onClose={() => setAskRunIRConfirmation(false)}
					title='Are you sure?'
					actions={
						<>
							<SlButton onClick={() => setAskRunIRConfirmation(false)}>Cancel</SlButton>
							<SlButton variant='primary' onClick={onRunIdentityResolution}>
								Run
							</SlButton>
						</>
					}
				>
					<p>
						The time it takes to perform the Identity Resolution can vary significantly, from seconds to
						hours, depending on the size of user data.
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
	);
};

export { UsersList };
