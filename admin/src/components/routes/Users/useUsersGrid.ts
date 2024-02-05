import { useMemo } from 'react';
import { GridColumn, GridRow } from '../../../types/componentTypes/Grid.types';
import { UserProperty } from '../../../types/internal/user';

const useUsersGrid = (
	users: Record<string, any>[],
	usersProperties: UserProperty[],
	selectedUser: number,
	onUserClick: (id: number) => void,
) => {
	const usersRows = useMemo(() => {
		const isIDUsed = usersProperties.find((property) => property.name === 'Id')?.isUsed;
		// compute the rows for the grid component.
		const rows: GridRow[] = [];
		for (const user of users) {
			// copy the user to prevent changes in-place.
			let userCopy = { ...user };
			const isSelected = userCopy.Id === selectedUser;
			if (!isIDUsed) {
				// do not show the id in the grid if this is the preference.
				delete userCopy.Id;
			}
			const row: GridRow = {
				onClick: () => onUserClick(user.Id),
				cells: Object.values(userCopy),
				selected: isSelected,
			};
			rows.push(row);
		}
		return rows;
	}, [users, usersProperties, onUserClick]);

	const usersColumns = useMemo(() => {
		// compute the columns for the grid component.
		const usersColumns: GridColumn[] = [];
		for (const p of usersProperties) {
			if (p.isUsed) {
				usersColumns.push({
					name: p.name,
					type: p.type,
				});
			}
		}
		return usersColumns;
	}, [usersProperties]);

	return { usersRows, usersColumns };
};

export { useUsersGrid };
