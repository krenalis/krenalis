import { useMemo } from 'react';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { UserProperty } from './Users.types';

const useUsersGrid = (
	users: Record<string, any>[],
	usersProperties: UserProperty[],
	selectedUser: number,
	onUserClick: (id: number) => void,
) => {
	const usersRows = useMemo(() => {
		const isIDUsed = usersProperties.find((property) => property.name === '__id__')?.isUsed;
		// compute the rows for the grid component.
		const rows: GridRow[] = [];
		for (const user of users) {
			// copy the user to prevent changes in-place.
			let userCopy = { ...user };
			const isSelected = userCopy.__id__ === selectedUser;
			if (!isIDUsed) {
				// do not show the id in the grid if this is the preference.
				delete userCopy.__id__;
			}
			const row: GridRow = {
				onClick: () => onUserClick(user.__id__),
				cells: Object.values(userCopy),
				selected: isSelected,
			};
			rows.push(row);
		}
		return rows;
	}, [users, usersProperties, onUserClick]);

	const userColumns = useMemo(() => {
		// compute the columns for the grid component.
		const userColumns: GridColumn[] = [];
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
