import { useMemo } from 'react';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { UserProperty } from './Users.types';
import { ResponseUser } from '../../../lib/api/types/responses';

const useUsersGrid = (
	users: ResponseUser[],
	usersProperties: UserProperty[],
	selectedUser: string,
	onUserClick: (id: string) => void,
) => {
	const usersRows = useMemo(() => {
		// compute the rows for the grid component.
		const rows: GridRow[] = [];
		for (const user of users) {
			// copy the user to prevent changes in-place.
			let userCopy = { ...user };
			const isSelected = userCopy.id === selectedUser;
			const row: GridRow = {
				onClick: () => onUserClick(user.id),
				cells: [userCopy.lastChangeTime, ...Object.values(userCopy.traits)],
				selected: isSelected,
			};
			rows.push(row);
		}
		return rows;
	}, [users, usersProperties, onUserClick]);

	const userColumns = useMemo(() => {
		// compute the columns for the grid component.
		const userColumns: GridColumn[] = [];
		userColumns.push({
			name: 'Last Change Time',
			type: 'DateTime',
		});
		for (const p of usersProperties) {
			if (p.isUsed) {
				userColumns.push({
					name: p.name,
					type: p.type,
				});
			}
		}
		return userColumns;
	}, [usersProperties]);

	return { usersRows, userColumns };
};

export { useUsersGrid };
