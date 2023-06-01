import { useContext } from 'react';
import './UsersList.css';
import Toolbar from '../Toolbar/Toolbar';
import StyledGrid from '../StyledGrid/StyledGrid';
import { NavigationContext } from '../../context/NavigationContext';
import { UsersContext } from '../../context/UsersContext';
import {
	SlButton,
	SlDropdown,
	SlIcon,
	SlMenu,
	SlOption,
	SlSwitch,
	SlSelect,
} from '@shoelace-style/shoelace/dist/react/index.js';

const UsersList = () => {
	let { setCurrentTitle } = useContext(NavigationContext);
	setCurrentTitle('Golden Record users');

	let { usersRows, usersCount, limit, properties, pagination, columnDefs, isLoading, fetchUsers } =
		useContext(UsersContext);

	const onPageChange = async (page) => {
		fetchUsers(page);
	};

	const onToggleColumn = (name) => {
		let props = [...properties];
		for (let p of props) {
			if (p.name === name) p.isUsed = !p.isUsed;
		}
		let columnDefs = [];
		for (let p of props) {
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
		let value = e.currentTarget.value;
		localStorage.setItem('usersLimit', value);
		// setLimit(value);
		fetchUsers(pagination.current);
	};

	return (
		<div className='UsersList'>
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
					<StyledGrid
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
								<SlSelect value={limit} placeholder={limit} onSlChange={onLimitChange}>
									<SlOption value={15}>15</SlOption>
									<SlOption value={30}>30</SlOption>
									<SlOption value={50}>50</SlOption>
									<SlOption value={70}>70</SlOption>
									<SlOption value={100}>100</SlOption>
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
