import React, { useContext, useState } from 'react';
import * as icons from '../../../constants/icons';
import UsersContext from '../../../context/UsersContext';
import Toolbar from '../../base/Toolbar/Toolbar';
import Grid from '../../base/Grid/Grid';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import { UserDrawer } from './UserDrawer';
import { useUsersGrid } from './useUsersGrid';
import { UserProperty } from './Users.types';
import AppContext from '../../../context/AppContext';

const UsersList = () => {
	const [selectedUser, setSelectedUser] = useState<string>('');
	const [isLoadingIdentityResolution, setIsLoadingIdentityResolution] = useState<boolean>(false);

	const { api, handleError, showStatus } = useContext(AppContext);
	const { users, usersCount, usersProperties, isLoading, fetchUsers } = useContext(UsersContext);
	const { usersRows, userColumns } = useUsersGrid(users, usersProperties, selectedUser, (id: string) =>
		setSelectedUser(id),
	);

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
				<SlButton onClick={onRunIdentityResolution} loading={isLoadingIdentityResolution} variant='primary'>
					Run identity resolution
				</SlButton>
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
