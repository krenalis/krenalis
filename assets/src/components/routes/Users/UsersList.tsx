import React, { useContext, useState } from 'react';
import * as icons from '../../../constants/icons';
import UsersContext from '../../../context/UsersContext';
import Toolbar from '../../layout/Toolbar/Toolbar';
import Grid from '../../shared/Grid/Grid';
import SlButton from '@shoelace-style/shoelace/dist/react/button/index.js';
import SlDropdown from '@shoelace-style/shoelace/dist/react/dropdown/index.js';
import SlIcon from '@shoelace-style/shoelace/dist/react/icon/index.js';
import SlMenu from '@shoelace-style/shoelace/dist/react/menu/index.js';
import SlOption from '@shoelace-style/shoelace/dist/react/option/index.js';
import SlSwitch from '@shoelace-style/shoelace/dist/react/switch/index.js';
import SlSelect from '@shoelace-style/shoelace/dist/react/select/index.js';
import { UserDrawer } from './UserDrawer';
import { useUsersGrid } from './useUsersGrid';
import { UserProperty } from './Users.types';
import AppContext from '../../../context/AppContext';

const UsersList = () => {
	const [selectedUser, setSelectedUser] = useState<number>(0);
	const [isLoadingIdentityResolution, setIsLoadingIdentityResolution] = useState<boolean>(false);

	const { api, handleError, showStatus } = useContext(AppContext);
	const { users, usersCount, limit, usersProperties, pagination, isLoading, fetchUsers } = useContext(UsersContext);
	const { usersRows, usersColumns } = useUsersGrid(users, usersProperties, selectedUser, (id: number) =>
		setSelectedUser(id),
	);

	const onPageChange = async (page: number) => {
		await fetchUsers(page);
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
		localStorage.setItem('chichi_ui_users_properties', JSON.stringify(updatedProps));
		fetchUsers(pagination.current);
	};

	const onLimitChange = (e) => {
		const value = e.currentTarget.value;
		localStorage.setItem('chichi_ui_users_limit', value);
		fetchUsers(pagination.current);
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
			fetchUsers(pagination.current);
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
						columns={usersColumns}
						rows={usersRows}
						isLoading={isLoading}
						noRowsMessage={'No users to show'}
					/>
					<UserDrawer selectedUser={selectedUser} setSelectedUser={setSelectedUser} />
					<div className='users-list__footer'>
						<div className='users-list__footer-total'>
							<div className='users-list__footer-found'>Found {usersCount} users</div>
							<div className='users-list__footer-limit'>
								<span>Show:</span>
								<SlSelect value={String(limit)} placeholder={String(limit)} onSlChange={onLimitChange}>
									<SlOption value='15'>15</SlOption>
									<SlOption value='30'>30</SlOption>
									<SlOption value='50'>50</SlOption>
									<SlOption value='70'>70</SlOption>
									<SlOption value='100'>100</SlOption>
								</SlSelect>
							</div>
						</div>
						{usersCount > limit && (
							<div className='users-list__pagination'>
								<span
									className='users-list__pagination-first'
									onClick={() => {
										onPageChange(1);
									}}
								>
									<SlIcon slot='suffix' name='chevron-double-left' />
								</span>
								{pagination.current !== 1 && (
									<span
										className='users-list__pagination-previous'
										onClick={() => {
											onPageChange(pagination.current - 1);
										}}
									>
										<SlIcon slot='suffix' name='chevron-left' />
									</span>
								)}
								<div className='users-list__pagination-pages'>
									Page
									<span className='users-list__pagination-current'>{pagination.current}</span>
									of
									<span className='users-list__pagination-last'>{pagination.last}</span>
								</div>
								{pagination.current !== pagination.last && (
									<span
										className='users-list__pagination-next'
										onClick={() => {
											onPageChange(pagination.current + 1);
										}}
									>
										<SlIcon slot='suffix' name='chevron-right' />
									</span>
								)}
								<span
									className='users-list__pagination-last'
									onClick={() => {
										onPageChange(pagination.last);
									}}
								>
									<SlIcon slot='suffix' name='chevron-double-right' />
								</span>
							</div>
						)}
					</div>
				</div>
			</div>
		</div>
	);
};

export { UsersList };
