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
			const traits = userCopy.traits;

			const cells: any[] = [];
			for (const p of usersProperties) {
				if (!p.isUsed) {
					continue;
				}
				const path = p.name;
				const isSubProperty = path.includes('.');
				if (isSubProperty) {
					const parts = path.split('.');
					let v: any = traits;
					for (const part of parts) {
						if (typeof v === 'object' && v !== null) {
							v = v[part];
						}
					}
					cells.push(v);
				} else {
					cells.push(traits[path]);
				}
			}

			const row: GridRow = {
				onClick: () => onUserClick(user.id),
				cells: [userCopy.lastChangeTime, ...cells],
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
			type: 'datetime',
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
