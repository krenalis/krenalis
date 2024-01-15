import React, { useEffect, useContext, useState } from 'react';
import './UsersWrapper.css';
import UsersContext from '../../../context/UsersContext';
import AppContext from '../../../context/AppContext';
import statuses from '../../../constants/statuses';
import { adminBasePath } from '../../../constants/path';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { UserProperty, UserPagination } from '../../../types/internal/user';
import { Outlet } from 'react-router-dom';
import { ObjectType } from '../../../types/external/types';
import { FindUsersResponse } from '../../../types/external/api';

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

	const { api, handleError, showStatus, redirect, selectedWorkspace, warehouse } = useContext(AppContext);

	useEffect(() => {
		if (warehouse == null) {
			redirect('settings');
			handleError('Please connect to a data warehouse before proceeding');
			return;
		}
		fetchUsers(1);
	}, [selectedWorkspace]);

	const fetchUsers = async (page: number) => {
		setIsLoading(true);

		let limit = DEFAULT_USER_LIMIT;
		const storageLimit = localStorage.getItem('usersLimit');
		if (storageLimit != null) {
			limit = Number(JSON.parse(storageLimit));
		}
		setLimit(limit);

		let schema: ObjectType;
		try {
			schema = await api.workspaces.userSchema();
		} catch (err) {
			setTimeout(() => {
				setIsLoading(false);
			}, 300);
			handleError(err);
			return;
		}

		let properties: UserProperty[] = [];
		const storageProperties = localStorage.getItem('usersProperties');
		if (storageProperties != null) {
			const decoded = JSON.parse(storageProperties);
			for (const p of schema.properties) {
				const storageProperty = decoded.find((column) => column.name === p.name);
				if (storageProperty != null) {
					properties.push({ name: p.name, isUsed: storageProperty.isUsed, type: p.type.name });
				} else {
					properties.push({ name: p.name, isUsed: true, type: p.type.name });
				}
			}
		} else {
			for (const p of schema.properties) {
				properties.push({ name: p.name, isUsed: true, type: p.type.name });
			}
		}
		localStorage.setItem('usersProperties', JSON.stringify(properties));
		setProperties(properties);

		const propertiesNames: string[] = [];
		for (const p of properties) {
			if (p.name === 'Id') {
				// always fetch the id. it is needed for navigation.
				propertiesNames.push(p.name);
			} else if (p.isUsed) {
				propertiesNames.push(p.name);
			}
		}

		const first = page * limit - limit;
		let res: FindUsersResponse;
		try {
			res = await api.workspaces.users.find(propertiesNames, null, first, limit);
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
			handleError(err);
			return;
		}

		const users = res.users;
		const count = res.count;

		setUsersCount(count);
		setPagination({ current: page, last: Math.ceil(count / limit) });

		// we need the id for navigation but we must remove it from the rows of
		// the grid if the user has manually hidden it in the UI.
		const isIDUsed = properties.find((property) => property.name === 'Id').isUsed;

		const rows: GridRow[] = [];
		const idList: number[] = [];
		for (const user of users) {
			const id = user.Id;
			idList.push(id);
			if (!isIDUsed) {
				delete user.Id;
			}
			const row: GridRow = {
				onClick: () => {
					redirect(`users/${id}`);
				},
				cells: Object.values(user),
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
