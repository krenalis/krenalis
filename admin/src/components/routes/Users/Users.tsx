import React, { useContext, useLayoutEffect } from 'react';
import './Users.css';
import AppContext from '../../../context/AppContext';
import UsersContext from '../../../context/UsersContext';
import { UsersList } from './UsersList';

import { useUsers } from './useUsers';

const Users = () => {
	const { setTitle } = useContext(AppContext);

	const { users, usersTotal, usersProperties, isLoading, userIDList, fetchUsers } = useUsers();

	useLayoutEffect(() => {
		setTitle('User Profiles');
	}, []);

	return (
		<UsersContext.Provider
			value={{
				users,
				usersTotal,
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
