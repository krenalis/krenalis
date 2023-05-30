import { useEffect, useContext, useState } from 'react';
import './UsersWrapper.css';
import { NavigationContext } from '../../context/NavigationContext';
import { UsersContext } from '../../context/UsersContext';
import { AppContext } from '../../context/AppContext';
import statuses from '../../constants/statuses';
import { NotFoundError, UnprocessableError } from '../../api/errors';
import { useNavigate } from 'react-router';
import { Outlet } from 'react-router-dom';

const DEFAULT_USER_LIMIT = 15;

const UsersWrapper = () => {
	let [usersRows, setUsersRows] = useState([]);
	let [usersCount, setUsersCount] = useState(0);
	let [columnDefs, setColumnDefs] = useState([]);
	let [properties, setProperties] = useState([]);
	let [userIDList, setUserIDList] = useState([]);
	let [pagination, setPagination] = useState({});
	let [isLoading, setIsLoading] = useState(false);
	let [limit, setLimit] = useState(0);

	let { setCurrentRoute } = useContext(NavigationContext);

	let { API, showError, showStatus, redirect } = useContext(AppContext);

	const navigate = useNavigate();

	useEffect(() => {
		setCurrentRoute('users');
	}, []);

	useEffect(() => {
		fetchUsers(1);
	}, []);

	const fetchUsers = async (page) => {
		setIsLoading(true);

		let lim = DEFAULT_USER_LIMIT;
		let storageLimit = localStorage.getItem('usersLimit');
		if (storageLimit != null) {
			lim = Number(JSON.parse(storageLimit));
		}
		setLimit(lim);

		let properties = [];
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
				properties.push({ name: p.name, isUsed: true, type: p.type.name });
			}
			localStorage.setItem('usersProperties', JSON.stringify(properties));
		}
		setProperties(properties);

		let propertiesNames = [];
		for (let p of properties) {
			if (p.name === 'id') {
				// always fetch the id. it is needed for navigation.
				propertiesNames.push(p.name);
			} else if (p.isUsed) {
				propertiesNames.push(p.name);
			}
		}

		let start = page * lim - lim;
		let [res, err] = await API.users.find(propertiesNames, start, start + lim);
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
						fetchUsers(page);
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
		setPagination({ current: page, last: Math.ceil(count / lim) });

		// find the index of the id property. We should use it for the
		// navigation but also remove it from the rows if the user has manually
		// hidden it in the UI.
		let idIndex, isIDHidden;
		for (let [i, p] of properties.entries()) {
			if (p.name === 'id') {
				idIndex = i;
				if (!p.isUsed) isIDHidden = true;
				break;
			}
		}

		let rows = [];
		let idList = [];
		for (let user of users) {
			let id = user[idIndex];
			idList.push(id);
			let row = {
				onClick: () => {
					navigate(`/admin/users/${id}`);
				},
			};
			if (isIDHidden) {
				user.splice(idIndex, 1);
			}
			row.cells = user;
			rows.push(row);
		}
		setUsersRows(rows);
		setUserIDList(idList);

		let usersColumns = [];
		for (let p of properties) {
			if (p.isUsed) {
				usersColumns.push({
					name: p.name,
					type: p.type,
				});
			}
		}
		setColumnDefs(usersColumns);
		setTimeout(() => {
			setIsLoading(false);
		}, 300);
	};

	return (
		<UsersContext.Provider
			value={{
				usersRows,
				usersCount,
				limit,
				properties,
				pagination,
				columnDefs,
				isLoading,
				userIDList,
				fetchUsers,
			}}
		>
			<div className='UsersWrapper'>
				<Outlet />
			</div>
		</UsersContext.Provider>
	);
};

export default UsersWrapper;
