import React, { useContext, useLayoutEffect } from 'react';
import './Users.css';
import AppContext from '../../../context/AppContext';
import UsersContext from '../../../context/UsersContext';
import { UsersList } from './UsersList';

import { useUsers } from './useUsers';

const Users = () => {
	const { setTitle } = useContext(AppContext);

	const { users, usersCount, usersProperties, isLoading, userIDList, fetchUsers } = useUsers();

	useLayoutEffect(() => {
		setTitle('Users');
	}, []);

	return (
		<UsersContext.Provider
			value={{
				users,
				usersCount,
				usersProperties,
				isLoading,
				userIDList,
				fetchUsers,
			}}
		>
			<UsersList />
		</UsersContext.Provider>
	);
};

export { Users };
