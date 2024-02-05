import { createContext } from 'react';
import { UserPagination, UserProperty } from '../types/internal/user';

interface UsersContext {
	users: Record<string, any>[];
	usersCount: number;
	limit: number;
	usersProperties: UserProperty[];
	pagination: UserPagination;
	isLoading: boolean;
	userIDList: number[];
	fetchUsers: (page: number) => Promise<number[]>;
}

const usersContext = createContext<UsersContext>({} as UsersContext);

export default usersContext;
