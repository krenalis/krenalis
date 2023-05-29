import { useEffect, useContext, useState } from 'react';
import './UsersWrapper.css';
import { NavigationContext } from '../../context/NavigationContext';
import { UsersContext } from '../../context/UsersContext';
import { AppContext } from '../../context/AppContext';
import statuses from '../../constants/statuses';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import { useNavigate } from 'react-router';
import { Outlet } from 'react-router-dom';

const UsersWrapper = () => {
	let [usersRows, setUsersRows] = useState([]);
	let [usersCount, setUsersCount] = useState(0);
	let [columnDefs, setColumnDefs] = useState([]);
	let [properties, setProperties] = useState([]);
	let [pagination, setPagination] = useState({});
	let [isLoading, setIsLoading] = useState(false);
	let [limit, setLimit] = useState(15);
	let [refetch, setRefetch] = useState(false);

	let { setCurrentRoute } = useContext(NavigationContext);

	let { API, showError, showStatus, redirect } = useContext(AppContext);

	const navigate = useNavigate();

	useEffect(() => {
		setCurrentRoute('users');
	}, []);

	useEffect(() => {
		const fetchUsers = async () => {
			setIsLoading(true);
			let lim;
			let storageLimit = localStorage.getItem('usersLimit');
			if (storageLimit != null) {
				lim = Number(JSON.parse(storageLimit));
				setLimit(lim);
			} else {
				lim = 15;
			}

			let properties = {};
			let storageProperties = localStorage.getItem('usersProperties');
			if (storageProperties != null) {
				properties = JSON.parse(storageProperties);
			} else {
				let [schema, err] = await API.workspace.userSchema();
				if (err) {
					setTimeout(() => {
						setIsLoading(false);
					}, 300);
					showError(err);

					return;
				}
				for (let p of schema.properties) {
					properties[p.name] = { isUsed: true, type: p.type.name };
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

			let [res, err] = await API.users.find(propertiesNames, 0, lim);
			if (err != null) {
				setTimeout(() => {
					setIsLoading(false);
				}, 300);
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
				return;
			}

			let { count, users } = res;

			setUsersCount(count);
			setPagination({ current: 1, last: Math.ceil(count / lim) });

			let rows = [];
			for (let user of users) {
				let id = user[0];
				rows.push({
					cells: user,
					onClick: () => navigate(`/admin/users/${id}`),
				});
			}
			setUsersRows(rows);

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
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
		};
		if (refetch) {
			setRefetch(false);
			return;
		}
		fetchUsers();
	}, [refetch]);

	return (
		<UsersContext.Provider
			value={{
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
				columnDefs,
				setColumnDefs,
				isLoading,
				setIsLoading,
				setRefetch,
			}}
		>
			<div className='UsersWrapper'>
				<Outlet />
			</div>
		</UsersContext.Provider>
	);
};

export default UsersWrapper;
