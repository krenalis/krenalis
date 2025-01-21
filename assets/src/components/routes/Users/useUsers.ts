import { useEffect, useContext, useState } from 'react';
import AppContext from '../../../context/AppContext';
import { UI_BASE_PATH } from '../../../constants/paths';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { UserProperty } from './Users.types';
import { ObjectType } from '../../../lib/api/types/types';
import { FindUsersResponse, ResponseUser } from '../../../lib/api/types/responses';

const DEFAULT_USER_LIMIT = 1000;

const useUsers = () => {
	const [users, setUsers] = useState<ResponseUser[]>([]);
	const [usersTotal, setUsersTotal] = useState<number>(0);
	const [usersProperties, setUsersProperties] = useState<UserProperty[]>([]);
	const [userIDList, setUserIDList] = useState<string[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(false);

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
		fetchUsers();
	}, [selectedWorkspace]);

	const fetchUsers = async (): Promise<string[]> => {
		setIsLoading(true);

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
		const storageProperties = localStorage.getItem('meergo_ui_users_properties');
		let preferenceProperties: UserProperty[] = [];
		if (storageProperties != null) {
			try {
				preferenceProperties = JSON.parse(storageProperties);
			} catch (err) {
				// the value of the properties in the storage is corrupted.
				localStorage.removeItem('meergo_ui_users_properties');
			}
		}

		// compute the users properties.
		const properties: UserProperty[] = [];
		for (const p of schema.properties) {
			// check if there is a preference for the visualization of the
			// property, otherwise, show the property by default.
			const property = preferenceProperties.find((prop) => prop.name === p.name);
			properties.push({
				name: p.name,
				isUsed: property ? property.isUsed : true,
				type: p.type.name,
			});
		}
		setUsersProperties(properties);

		// update the value of the properties in the storage.
		localStorage.setItem('meergo_ui_users_properties', JSON.stringify(properties));

		// compute the names of the showed user properties to request
		// only those properties when fetching the users.
		const propertiesNames: string[] = [];
		for (const p of properties) {
			if (p.isUsed) {
				propertiesNames.push(p.name);
			}
		}

		// fetch the users.
		let res: FindUsersResponse;
		try {
			res = await api.workspaces.users.find(propertiesNames, null, '', true, 0, DEFAULT_USER_LIMIT);
		} catch (err) {
			setTimeout(() => {
				setIsLoading(false);
				if (err instanceof NotFoundError) {
					redirect(UI_BASE_PATH);
					handleError('The workspace does not exist anymore');
					return;
				}
				if (err instanceof UnprocessableError) {
					switch (err.code) {
						case 'PropertyNotExist':
							// one of the properties has been concurrently
							// removed from the user schema. Try again.
							fetchUsers();
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
		const total = res.total;

		for (const user of users) {
			const traits: Record<string, any> = {};
			for (const name of propertiesNames) {
				const value = user.traits[name];
				traits[name] = value ? value : undefined;
			}
			user.traits = traits;
		}

		setUsers(users);
		setUsersTotal(total);

		// compute the list of users ids needed for navigating between users.
		const ids: string[] = [];
		for (const user of users) {
			ids.push(user.id);
		}
		setUserIDList(ids);

		setTimeout(() => {
			setIsLoading(false);
		}, 300);

		return ids;
	};

	return {
		users,
		usersTotal,
		usersProperties,
		isLoading,
		userIDList,
		fetchUsers,
	};
};

export { useUsers };
