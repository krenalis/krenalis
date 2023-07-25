import { useEffect, useContext, useState } from 'react';
import './UsersWrapper.css';
import { UsersContext } from '../../../context/UsersContext';
import { AppContext } from '../../../context/providers/AppProvider';
import statuses from '../../../constants/statuses';
import { adminBasePath } from '../../../constants/path';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { Outlet } from 'react-router-dom';

const DEFAULT_USER_LIMIT = 15;

const UsersWrapper = () => {
	const [usersRows, setUsersRows] = useState([]);
	const [usersCount, setUsersCount] = useState(0);
	const [columnDefs, setColumnDefs] = useState([]);
	const [properties, setProperties] = useState([]);
	const [userIDList, setUserIDList] = useState([]);
	const [pagination, setPagination] = useState({});
	const [isLoading, setIsLoading] = useState(false);
	const [limit, setLimit] = useState(0);

	const { api, showError, showStatus, redirect } = useContext(AppContext);

	useEffect(() => {
		fetchUsers(1);
	}, []);

	const fetchUsers = async (page) => {
		setIsLoading(true);

		let lim = DEFAULT_USER_LIMIT;
		const storageLimit = localStorage.getItem('usersLimit');
		if (storageLimit != null) {
			lim = Number(JSON.parse(storageLimit));
		}
		setLimit(lim);

		let properties = [];
		const storageProperties = localStorage.getItem('usersProperties');
		if (storageProperties != null) {
			properties = JSON.parse(storageProperties);
		} else {
			const [schema, err] = await api.workspace.userSchema();
			if (err) {
				setTimeout(() => {
					setIsLoading(false);
				}, 300);
				showError(err);
				return;
			}
			for (const p of schema.properties) {
				properties.push({ name: p.name, isUsed: true, type: p.type.name });
			}
			localStorage.setItem('usersProperties', JSON.stringify(properties));
		}
		setProperties(properties);

		const propertiesNames = [];
		for (const p of properties) {
			if (p.name === 'id') {
				// always fetch the id. it is needed for navigation.
				propertiesNames.push(p.name);
			} else if (p.isUsed) {
				propertiesNames.push(p.name);
			}
		}

		const start = page * lim - lim;
		const [res, err] = await api.users.find(propertiesNames, start, start + lim);
		if (err != null) {
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
			if (err instanceof NotFoundError) {
				redirect(adminBasePath);
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

		const { count, users } = res;

		setUsersCount(count);
		setPagination({ current: page, last: Math.ceil(count / lim) });

		// find the index of the id property. We should use it for the
		// navigation but also remove it from the rows if the user has manually
		// hidden it in the UI.
		let idIndex, isIDHidden;
		for (const [i, p] of properties.entries()) {
			if (p.name === 'id') {
				idIndex = i;
				if (!p.isUsed) isIDHidden = true;
				break;
			}
		}

		const rows = [];
		const idList = [];
		for (const user of users) {
			const id = user[idIndex];
			idList.push(id);
			const row = {
				onClick: () => {
					redirect(`users/${id}`);
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

		const usersColumns = [];
		for (const p of properties) {
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
			<div className='usersWrapper'>
				<Outlet />
			</div>
		</UsersContext.Provider>
	);
};

export default UsersWrapper;
