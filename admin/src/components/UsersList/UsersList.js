import { useContext } from 'react';
import './UsersList.css';
import statuses from '../../constants/statuses';
import Toolbar from '../Toolbar/Toolbar';
import StyledGrid from '../StyledGrid/StyledGrid';
import { AppContext } from '../../context/AppContext';
import { NavigationContext } from '../../context/NavigationContext';
import { UsersContext } from '../../context/UsersContext';
import { NotFoundError, UnprocessableError } from '../../api/errors';
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
	let { API, showError, showStatus, redirect } = useContext(AppContext);

	let { setCurrentTitle } = useContext(NavigationContext);
	setCurrentTitle('Golden Record users');

	let {
		usersRows,
		setUsersRows,
		usersCount,
		setUsersCount,
		limit,
		setLimit,
		properties,
		setProperties,
		pagination,
		setPagination,
		setRefetch,
		columnDefs,
		setColumnDefs,
		isLoading,
		setIsLoading,
	} = useContext(UsersContext);

	const onPageChange = async (page) => {
		let propertiesNames = [];
		for (let name in properties) propertiesNames.push(name);
		let start = page * limit - limit;
		setIsLoading(true);
		let [res, err] = await API.users.find(propertiesNames, start, start + limit);
		if (err != null) {
			if (err instanceof NotFoundError) {
				redirect('/admin');
				showStatus(statuses.workspaceDoesNotExistAnymore);
				return;
			}
			if (err instanceof UnprocessableError) {
				switch (err.code) {
					case 'PropertyNotExists':
						showStatus(statuses.propertyNotExist);
						break;
					case 'WarehouseFailed':
						showStatus(statuses.warehouseConnectionFailed);
						break;
					default:
						break;
				}
				return;
			}
			showError(err);
			setTimeout(() => setIsLoading(false), 500);
			return;
		}
		setTimeout(() => setIsLoading(false), 500);

		let { count, users } = res;

		setUsersCount(count);
		setPagination({ current: page, last: Math.ceil(count / limit) });

		let usersColumns = [];
		for (let [name, property] of Object.entries(properties)) {
			if (property.isUsed) {
				usersColumns.push({
					name: name,
					type: property.type,
				});
			}
		}
		setColumnDefs(usersColumns);

		let rows = [];
		for (let user of users) {
			rows.push({ cells: user });
		}
		setUsersRows(rows);
		setUsersRows(usersRows);
	};

	const onToggleColumn = (name) => {
		let props = { ...properties };
		props[name].isUsed = !props[name].isUsed;
		let columnDefs = [];
		for (let [name, property] of Object.entries(props)) {
			if (property.isUsed) {
				columnDefs.push({
					name: name,
					type: property.type,
				});
			}
		}
		localStorage.setItem('usersProperties', JSON.stringify(props));
		setProperties(props);
		setRefetch(true);
	};

	const onLimitChange = (e) => {
		let value = e.currentTarget.value;
		localStorage.setItem('usersLimit', value);
		setLimit(value);
		setRefetch(true);
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
						{Object.entries(properties).map(([name, property]) => {
							return (
								<SlOption>
									<SlSwitch
										size='small'
										onSlChange={() => onToggleColumn(name)}
										checked={property.isUsed}
									>
										{name}
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
									<span
										className='last'
										onClick={() => {
											onPageChange(pagination.last);
										}}
									>
										{pagination.last}
									</span>
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
