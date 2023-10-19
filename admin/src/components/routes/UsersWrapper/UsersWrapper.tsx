import React, { useEffect, useContext, useState } from 'react';
import './UsersWrapper.css';
import UsersContext from '../../../context/UsersContext';
import { AppContext } from '../../../context/providers/AppProvider';
import statuses from '../../../constants/statuses';
import { adminBasePath } from '../../../constants/path';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { UserProperty, UserPagination } from '../../../types/internal/user';
import { Outlet } from 'react-router-dom';

const DEFAULT_USER_LIMIT = 15;

const UsersWrapper = () => {
	const [usersRows, setUsersRows] = useState<GridRow[]>([]);
	const [usersCount, setUsersCount] = useState<number>(0);
	const [columnDefs, setColumnDefs] = useState<GridColumn[]>([]);
	const [properties, setProperties] = useState<UserProperty[]>([]);
	const [userIDList, setUserIDList] = useState<number[]>([]);
	const [pagination, setPagination] = useState<UserPagination>({} as UserPagination);
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [limit, setLimit] = useState<number>(0);

	const { api, showError, showStatus, redirect, selectedWorkspace, warehouse } = useContext(AppContext);

	useEffect(() => {
		if (warehouse == null) {
			redirect('settings');
			showError('Please connect to a data warehouse before proceeding');
			return;
		}
		fetchUsers(1);
	}, [selectedWorkspace]);

	const fetchUsers = async (page: number) => {
		setIsLoading(true);

		let lim = DEFAULT_USER_LIMIT;
		const storageLimit = localStorage.getItem('usersLimit');
		if (storageLimit != null) {
			lim = Number(JSON.parse(storageLimit));
		}
		setLimit(lim);

		let properties: UserProperty[] = [];
		const storageProperties = localStorage.getItem('usersProperties');
		if (storageProperties != null) {
			properties = JSON.parse(storageProperties);
		} else {
			let schema;
			try {
				schema = await api.workspaces.userSchema();
			} catch (err) {
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

		const propertiesNames: string[] = [];
		for (const p of properties) {
			if (p.name === 'id') {
				// always fetch the id. it is needed for navigation.
				propertiesNames.push(p.name);
			} else if (p.isUsed) {
				propertiesNames.push(p.name);
			}
		}

		const start = page * lim - lim;
		let res;
		try {
			res = await api.workspaces.users.find(null, propertiesNames, start, start + lim);
		} catch (err) {
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
					case 'PropertyNotExist':
						localStorage.removeItem('usersProperties');
						fetchUsers(page);
						break;
					case 'DataWarehouseFailed':
						showStatus(statuses.dataWarehouseFailed);
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

		const rows: GridRow[] = [];
		const idList: number[] = [];
		for (const user of users) {
			const id = user[idIndex];
			idList.push(id);
			if (isIDHidden) {
				user.splice(idIndex, 1);
			}
			const row: GridRow = {
				onClick: () => {
					redirect(`users/${id}`);
				},
				cells: user,
			};
			rows.push(row);
		}
		setUsersRows(rows);
		setUserIDList(idList);

		const usersColumns: GridColumn[] = [];
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
