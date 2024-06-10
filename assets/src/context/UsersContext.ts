import { createContext } from 'react';
import { UserProperty, UserPagination } from '../components/routes/Users/Users.types';
import { ResponseUser } from '../lib/api/types/responses';

interface UsersContext {
	users: ResponseUser[];
	usersCount: number;
	limit: number;
	usersProperties: UserProperty[];
	pagination: UserPagination;
	isLoading: boolean;
	userIDList: string[];
	fetchUsers: (page: number) => Promise<string[]>;
}

const usersContext = createContext<UsersContext>({} as UsersContext);

export default usersContext;
