import { createContext } from 'react';

const defaultUsersContext = {
	usersRows: [],
	usersCount: 0,
	setUsersRows: () => {},
	setUsersCount: () => {},
};

export const UsersContext = createContext(defaultUsersContext);
