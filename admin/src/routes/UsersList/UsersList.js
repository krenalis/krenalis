import { useState, useEffect, useRef } from 'react';
import './UsersList.css';
import Breadcrumbs from '../../components/Breadcrumbs/Breadcrumbs';
import Toast from '../../components/Toast/Toast';
import call from '../../utils/call';
import {
	SlButton,
	SlDropdown,
	SlIcon,
	SlMenu,
	SlMenuItem,
	SlSwitch,
	SlSelect,
	SlSpinner,
} from '@shoelace-style/shoelace/dist/react/index.js';
import { AgGridReact } from 'ag-grid-react';
import 'ag-grid-community/styles/ag-grid.css';
import 'ag-grid-community/styles/ag-theme-alpine.css';

const UsersList = () => {
	let [status, setStatus] = useState(null);
	let [columnDefs, setColumnDefs] = useState([]);
	let [usersRows, setUsersRows] = useState([]);
	let [usersCount, setUsersCount] = useState(0);
	let [properties, setProperties] = useState([]);
	let [pagination, setPagination] = useState({});
	let [isLoading, setIsLoading] = useState(false);
	let [limit, setLimit] = useState(15);

	let toastRef = useRef();
	let gridRef = useRef();

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
				let [userSchema, err] = await call('/admin/user-schema');
				if (err != null) {
					setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
					toastRef.current.toast();
					return;
				}
				for (let p of userSchema.properties) {
					properties[p.name] = { label: p.label, isUsed: true };
				}
				localStorage.setItem('usersProperties', JSON.stringify(properties));
			}
			setProperties(properties);

			let propertiesNames = [];
			for (let name in properties) {
				propertiesNames.push(name);
			}

			setIsLoading(true);
			let [{ count, users }, err] = await call('/api/users', 'POST', {
				properties: propertiesNames,
				start: 0,
				end: lim,
			});
			if (err != null) {
				setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
				setIsLoading(false);
				toastRef.current.toast();
				return;
			}
			setIsLoading(false);

			setUsersCount(count);
			setPagination({ current: 1, last: Math.ceil(count / lim) });

			let usersColumns = [];
			for (let [name, property] of Object.entries(properties)) {
				if (property.isUsed) {
					usersColumns.push({
						field: name,
						headerName: property.label,
						headerTooltip: property.description,
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
		fetchUsers();
	}, [limit]);

	const handlePageChange = async (page) => {
		let propertiesNames = [];
		for (let name in properties) propertiesNames.push(name);
		let start = page * limit - limit;
		setIsLoading(true);
		let [{ count, users }, err] = await call('/api/users', 'POST', {
			properties: propertiesNames,
			start: start,
			end: start + limit,
		});
		if (err != null) {
			setStatus({ variant: 'danger', icon: 'exclamation-octagon', text: err });
			toastRef.current.toast();
			setIsLoading(false);
			return;
		}
		setIsLoading(false);

		setUsersCount(count);
		setPagination({ current: page, last: Math.ceil(count / limit) });

		let usersColumns = [];
		for (let [name, property] of Object.entries(properties)) {
			if (property.isUsed) {
				usersColumns.push({
					field: name,
					headerName: property.label,
					headerTooltip: property.description,
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

	const handleToggleColumn = (name) => {
		let props = { ...properties };
		props[name].isUsed = !props[name].isUsed;
		let usersColumns = [];
		for (let [name, property] of Object.entries(props)) {
			if (property.isUsed) {
				usersColumns.push({
					field: name,
					headerName: property.label,
					headerTooltip: property.description,
				});
			}
		}
		gridRef.current.api.setColumnDefs(usersColumns);
		setProperties(props);
		localStorage.setItem('usersProperties', JSON.stringify(props));
	};

	const handleLimitChange = (e) => {
		let value = e.currentTarget.value;
		localStorage.setItem('usersLimit', value);
		setLimit(value);
	};

	return (
		<div className='UsersList'>
			<Breadcrumbs
				breadcrumbs={[
					{ Name: 'Connections map', Link: '/admin/connections-map' },
					{ Name: `Golden Record users` },
				]}
			/>
			<div className='routeContent'>
				<Toast reactRef={toastRef} status={status} />
				<div className='title'>
					<div className='text'>Golden Record users</div>
				</div>
				<div className='gridContainer'>
					<div className='head'>
						<div className='gridHeading'>
							<div className='gridTitle'>Users list</div>
						</div>
						<div className='gridActions'>
							<SlDropdown stayOpenOnSelect={true} className='toggleColumns'>
								<SlButton slot='trigger' variant='default'>
									<SlIcon slot='suffix' name='layout-three-columns' />
									Toggle columns
								</SlButton>
								<SlMenu>
									{Object.entries(properties).map(([name, property]) => {
										return (
											<SlMenuItem>
												<SlSwitch
													onSlChange={() => handleToggleColumn(name)}
													checked={property.isUsed}
												>
													{property.label}
												</SlSwitch>
											</SlMenuItem>
										);
									})}
								</SlMenu>
							</SlDropdown>
						</div>
					</div>
					<div className='grid ag-theme-alpine' style={{ height: '700px', width: '100%' }}>
						<AgGridReact ref={gridRef} columnDefs={columnDefs} rowData={usersRows}></AgGridReact>
						{isLoading && (
							<div className='loading'>
								<SlSpinner
									style={{
										fontSize: '3rem',
										'--track-width': '6px',
									}}
								/>
							</div>
						)}
					</div>
					<div className='footer'>
						<div className='total'>
							<div className='found'>Found {usersCount} users</div>
							<div className='gridLimit'>
								<span>Show:</span>
								<SlSelect value={limit} placeholder={limit} onSlChange={handleLimitChange}>
									<SlMenuItem value={15}>15</SlMenuItem>
									<SlMenuItem value={30}>30</SlMenuItem>
									<SlMenuItem value={50}>50</SlMenuItem>
									<SlMenuItem value={70}>70</SlMenuItem>
									<SlMenuItem value={100}>100</SlMenuItem>
								</SlSelect>
							</div>
						</div>
						{usersCount > limit && (
							<div className='pagination'>
								<span
									className='firstPage'
									onClick={() => {
										handlePageChange(1);
									}}
								>
									<SlIcon slot='suffix' name='chevron-double-left' />
								</span>
								{pagination.current !== 1 && (
									<span
										className='previousPage'
										onClick={() => {
											handlePageChange(pagination.current - 1);
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
											handlePageChange(pagination.last);
										}}
									>
										{pagination.last}
									</span>
								</div>
								{pagination.current !== pagination.last && (
									<span
										className='nextPage'
										onClick={() => {
											handlePageChange(pagination.current + 1);
										}}
									>
										<SlIcon slot='suffix' name='chevron-right' />
									</span>
								)}
								<span
									className='lastPage'
									onClick={() => {
										handlePageChange(pagination.last);
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
