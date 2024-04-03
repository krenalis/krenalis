import { useEffect, useContext, useState } from 'react';
import AppContext from '../../../context/AppContext';
import { uiBasePath } from '../../../constants/path';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { UserProperty, UserPagination } from '../../../types/internal/user';
import { ObjectType } from '../../../types/external/types';
import { FindUsersResponse } from '../../../types/external/api';

const DEFAULT_USER_LIMIT = 15;

const useUsers = () => {
	const [users, setUsers] = useState<Record<string, any>[]>([]);
	const [usersCount, setUsersCount] = useState<number>(0);
	const [usersProperties, setUsersProperties] = useState<UserProperty[]>([]);
	const [userIDList, setUserIDList] = useState<number[]>([]);
	const [pagination, setPagination] = useState<UserPagination>({} as UserPagination);
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [limit, setLimit] = useState<number>(0);

	const { api, handleError, redirect, selectedWorkspace, warehouse } = useContext(AppContext);

	useEffect(() => {
		if (warehouse == null) {
			// a workspace without a connected data warehouse cannot show
			// warehouse users.
			redirect('settings');
			handleError('Please connect to a data warehouse before proceeding');
			return;
		}
		// on mount, fetch the first page of users.
		fetchUsers(1);
	}, [selectedWorkspace]);

	const fetchUsers = async (page: number): Promise<number[]> => {
		setIsLoading(true);

		// compute the max number of users to show in the users list.
		let lim = DEFAULT_USER_LIMIT;
		const storageLimit = localStorage.getItem('chichi_ui_users_limit');
		if (storageLimit != null) {
			try {
				lim = Number(JSON.parse(storageLimit));
			} catch (err) {
				// the value of the limit in the storage is corrupted.
				localStorage.removeItem('chichi_ui_users_limit');
			}
		}
		setLimit(lim);

		// fetch the user schema.
		let schema: ObjectType;
		try {
			schema = await api.workspaces.userSchema();
		} catch (err) {
			setTimeout(() => {
				setIsLoading(false);
				handleError(err);
			}, 300);
			return;
		}

		// check if previous users properties are already saved in the storage.
		const storageProperties = localStorage.getItem('chichi_ui_users_properties');
		let preferenceProperties: UserProperty[] = [];
		if (storageProperties != null) {
			try {
				preferenceProperties = JSON.parse(storageProperties);
			} catch (err) {
				// the value of the properties in the storage is corrupted.
				localStorage.removeItem('chichi_ui_users_properties');
			}
		}

		// compute the users properties.
		const properties: UserProperty[] = [];
		for (const p of schema.properties) {
			// check if there is a preference for the visualization of the
			// property, othwerwise, show the property by default.
			const property = preferenceProperties.find((prop) => prop.name === p.name);
			properties.push({
				name: p.name,
				isUsed: property ? property.isUsed : true,
				type: p.type.name,
			});
		}
		setUsersProperties(properties);

		// update the value of the properties in the storage.
		localStorage.setItem('chichi_ui_users_properties', JSON.stringify(properties));

		// compute the names of the showed user properties to request only those
		// properties when fetching the users.
		const propertiesNames: string[] = [];
		for (const p of properties) {
			// always request the id as it is needed for navigation.
			if (p.name === 'Id' || p.isUsed) {
				propertiesNames.push(p.name);
			}
		}

		// fetch the users.
		const cursor = page * lim - lim;
		let res: FindUsersResponse;
		try {
			res = await api.workspaces.users.find(propertiesNames, null, cursor, lim);
		} catch (err) {
			setTimeout(() => {
				setIsLoading(false);
				if (err instanceof NotFoundError) {
					redirect(uiBasePath);
					handleError('The workspace does not exist anymore');
					return;
				}
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'PropertyNotExist':
							// one of the properties has been concurrently
							// removed from the user schema. Try again.
							fetchUsers(page);
							return;
						case 'DataWarehouseFailed':
							handleError('An error occurred with the data warehouse');
							return;
					}
				}
				handleError(err);
			}, 300);
			return;
		}
		const users = res.users;
		const count = res.count;

		setUsers(users);
		setUsersCount(count);
		setPagination({ current: page, last: Math.ceil(count / lim) });

		// compute the list of users ids needed for navigating between users.
		const ids: number[] = [];
		for (const user of users) {
			ids.push(user.Id);
		}
		setUserIDList(ids);

		setTimeout(() => {
			setIsLoading(false);
		}, 300);

		return ids;
	};

	return {
		users,
		usersCount,
		limit,
		usersProperties,
		pagination,
		isLoading,
		userIDList,
		fetchUsers,
	};
};

export { useUsers };
