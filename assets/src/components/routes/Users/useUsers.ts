import { useEffect, useContext, useState } from 'react';
import AppContext from '../../../context/AppContext';
import { UI_BASE_PATH } from '../../../constants/paths';
import { NotFoundError, UnprocessableError } from '../../../lib/api/errors';
import { UserProperty } from './Users.types';
import { ObjectType } from '../../../lib/api/types/types';
import { FindUsersResponse, ResponseUser } from '../../../lib/api/types/responses';
import { flattenSchema } from '../../../lib/core/action';

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
		let preferences: UserProperty[] = [];
		if (storageProperties != null) {
			try {
				preferences = JSON.parse(storageProperties);
			} catch (err) {
				// the value of the properties in the storage is corrupted.
				localStorage.removeItem('meergo_ui_users_properties');
			}
		}

		const flatSchema = flattenSchema(schema);
		const paths = Object.keys(flatSchema);

		// compute the properties to show in the table columns and in
		// the “Toggle columns” menu, and those that should be requested
		// to the server.
		const toShow: UserProperty[] = [];
		const toFetch: string[] = [];
		for (const path of paths) {
			const isFirstLevel = !path.includes('.');
			if (isFirstLevel) {
				// fetch all the users properties by passing all the
				// first level properties to the server.
				toFetch.push(path);
			}

			let isParent = false;
			const depth = path.split('.').length;
			for (const p of paths) {
				const isSameProperty = p === path;
				if (isSameProperty) {
					continue;
				}
				const isChildren = p.includes('.');
				if (isChildren) {
					const parts = p.split('.');
					const prefix = parts.slice(0, depth).join('.');
					if (prefix === path) {
						isParent = true;
						continue;
					}
				}
			}

			if (isParent) {
				// show only flattened subproperties instead of full
				// parent properties (e.g. `obj.prop.prop2` instead of
				// `obj`).
				continue;
			}

			// check if there is a preference for the property.
			const preference = preferences.find((prop) => prop.name === path);

			let isTypeChanged = false;
			if (preference != null) {
				isTypeChanged = preference.type !== flatSchema[path].type;
			}

			toShow.push({
				name: path,
				isUsed: preference != null && !isTypeChanged ? preference.isUsed : true,
				type: flatSchema[path].type,
			});
		}
		setUsersProperties(toShow);

		// update the value of the properties in the storage.
		localStorage.setItem('meergo_ui_users_properties', JSON.stringify(toShow));

		// fetch the users.
		let res: FindUsersResponse;
		try {
			res = await api.workspaces.users.find(toFetch, null, '', true, 0, DEFAULT_USER_LIMIT);
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
