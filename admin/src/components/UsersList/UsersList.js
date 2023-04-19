import { useState, useEffect, useContext } from 'react';
import './UsersList.css';
import statuses from '../../constants/statuses';
import Toolbar from '../Toolbar/Toolbar';
import StyledGrid from '../StyledGrid/StyledGrid';
import { AppContext } from '../../context/AppContext';
import { NavigationContext } from '../../context/NavigationContext';
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
	let [columnDefs, setColumnDefs] = useState([]);
	let [usersRows, setUsersRows] = useState([]);
	let [usersCount, setUsersCount] = useState(0);
	let [properties, setProperties] = useState([]);
	let [pagination, setPagination] = useState({});
	let [isLoading, setIsLoading] = useState(false);
	let [refetch, setRefetch] = useState(false);
	let [limit, setLimit] = useState(15);

	let { API, showError, showStatus, redirect } = useContext(AppContext);

	let { setCurrentTitle } = useContext(NavigationContext);
	setCurrentTitle('Golden Record users');

	useEffect(() => {
		const fetchUsers = async () => {
			let lim;
			let storageLimit = localStorage.getItem('usersLimit');
			if (storageLimit != null) {
				lim = Number(JSON.parse(storageLimit));
				setLimit(lim);
			} else {
				lim = limit;
			}

			let properties = {};
			let storageProperties = localStorage.getItem('usersProperties');
			if (storageProperties != null) {
				properties = JSON.parse(storageProperties);
			} else {
				let [schema, err] = await API.workspace.userSchema();
				if (err) {
					showError(err);
					return;
				}
				for (let p of schema.properties) {
					properties[p.name] = { isUsed: true };
				}
				localStorage.setItem('usersProperties', JSON.stringify(properties));
			}
			setProperties(properties);

			let propertiesNames = [];
			for (let name in properties) {
				if (properties[name].isUsed) {
					propertiesNames.push(name);
				}
			}

			setIsLoading(true);
			let [res, err] = await API.users.find(propertiesNames, 0, lim);
			if (err != null) {
				if (err instanceof NotFoundError) {
					redirect('/admin');
					showStatus(statuses.workspaceDoesNotExistAnymore);
					return;
				}
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'PropertyNotExists':
							localStorage.removeItem('usersProperties');
							setRefetch(true);
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
			setPagination({ current: 1, last: Math.ceil(count / lim) });
			setUsersRows(users);

			let usersColumns = [];
			for (let [name, property] of Object.entries(properties)) {
				if (property.isUsed) {
					usersColumns.push({
						Name: name,
					});
				}
			}
			setColumnDefs(usersColumns);
		};
		if (refetch) {
			setRefetch(false);
			return;
		}
		fetchUsers();
	}, [refetch]);

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
					field: name,
				});
			}
		}
		setColumnDefs(usersColumns);

		let usersRows = [];
		for (let u of users) {
			if (u == null) continue;
			let userRow = {};
			for (let [i, p] of Object.keys(properties).entries()) {
				userRow[p] = u[i];
			}
			usersRows.push(userRow);
		}
		setUsersRows(usersRows);
	};

	const onToggleColumn = (name) => {
		let props = { ...properties };
		props[name].isUsed = !props[name].isUsed;
		let columnDefs = [];
		for (let [name, property] of Object.entries(props)) {
			if (property.isUsed) {
				columnDefs.push({
					Name: name,
				});
			}
		}
		localStorage.setItem('usersProperties', JSON.stringify(props));
		setColumnDefs(columnDefs);
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
