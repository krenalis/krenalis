import React, { useContext } from 'react';
import './UsersList.css';
import Toolbar from '../../layout/Toolbar/Toolbar';
import Grid from '../../shared/Grid/Grid';
import { AppContext } from '../../../context/providers/AppProvider';
import UsersContext from '../../../context/UsersContext';
import {
	SlButton,
	SlDropdown,
	SlIcon,
	SlMenu,
	SlOption,
	SlSwitch,
	SlSelect,
} from '@shoelace-style/shoelace/dist/react/index.js';
import { GridColumn } from '../../../types/componentTypes/Grid.types';

const UsersList = () => {
	const { setTitle } = useContext(AppContext);
	setTitle('Users');

	const { usersRows, usersCount, limit, properties, pagination, columnDefs, isLoading, fetchUsers } =
		useContext(UsersContext);

	const onPageChange = async (page: number) => {
		fetchUsers(page);
	};

	const onToggleColumn = (name: string) => {
		const props = [...properties];
		for (const p of props) {
			if (p.name === name) p.isUsed = !p.isUsed;
		}
		const columnDefs: GridColumn[] = [];
		for (const p of props) {
			if (p.isUsed) {
				columnDefs.push({
					name: p.name,
					type: p.type,
				});
			}
		}
		localStorage.setItem('usersProperties', JSON.stringify(props));
		fetchUsers(pagination.current);
	};

	const onLimitChange = (e) => {
		const value = e.currentTarget.value;
		localStorage.setItem('usersLimit', value);
		// setLimit(value);
		fetchUsers(pagination.current);
	};

	return (
		<div className='usersList'>
			<Toolbar>
				<SlDropdown stayOpenOnSelect={true} className='toggleColumns'>
					<SlButton slot='trigger' variant='default'>
						<SlIcon slot='prefix' name='layout-three-columns' />
						Toggle columns
					</SlButton>
					<SlMenu>
						{properties.map((p) => {
							return (
								<SlOption>
									<SlSwitch size='small' onSlChange={() => onToggleColumn(p.name)} checked={p.isUsed}>
										{p.name}
									</SlSwitch>
								</SlOption>
							);
						})}
					</SlMenu>
				</SlDropdown>
			</Toolbar>
			<div className='routeContent'>
				<div className='gridContainer'>
					<Grid
						columns={columnDefs}
						rows={usersRows}
						isLoading={isLoading}
						noRowsMessage={'No users to show'}
					/>
					<div className='footer'>
						<div className='total'>
							<div className='found'>Found {usersCount} users</div>
							<div className='gridLimit'>
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
							<div className='pagination'>
								<span
									className='firstPage'
									onClick={() => {
										onPageChange(1);
									}}
								>
									<SlIcon slot='suffix' name='chevron-double-left' />
								</span>
								{pagination.current !== 1 && (
									<span
										className='previousPage'
										onClick={() => {
											onPageChange(pagination.current - 1);
										}}
									>
										<SlIcon slot='suffix' name='chevron-left' />
									</span>
								)}
								<div className='pages'>
									Page
									<span className='current'>{pagination.current}</span>
									of
									<span className='last'>{pagination.last}</span>
								</div>
								{pagination.current !== pagination.last && (
									<span
										className='nextPage'
										onClick={() => {
											onPageChange(pagination.current + 1);
										}}
									>
										<SlIcon slot='suffix' name='chevron-right' />
									</span>
								)}
								<span
									className='lastPage'
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

export default UsersList;
