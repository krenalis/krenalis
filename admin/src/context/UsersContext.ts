import { createContext } from 'react';
import { UserPagination, UserProperty } from '../types/internal/user';
import { GridColumn, GridRow } from '../types/componentTypes/Grid.types';

interface UsersContext {
	usersRows: GridRow[];
	usersCount: number;
	limit: number;
	properties: UserProperty[];
	pagination: UserPagination;
	columnDefs: GridColumn[];
	isLoading: boolean;
	userIDList: number[];
	fetchUsers: (page: number) => void;
}

const usersContext = createContext<UsersContext>({} as UsersContext);

export default usersContext;
