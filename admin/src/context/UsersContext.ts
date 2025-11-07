import { createContext } from 'react';
import { UserProperty } from '../components/routes/Users/Users.types';
import { ResponseUser } from '../lib/api/types/responses';

interface UsersContext {
	users: ResponseUser[];
	usersTotal: number;
	usersProperties: UserProperty[];
	isLoading: boolean;
	userIDList: string[];
	fetchUsers: () => Promise<string[]>;
}

const usersContext = createContext<UsersContext>({} as UsersContext);

export default usersContext;
