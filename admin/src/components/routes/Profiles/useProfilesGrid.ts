import { useMemo } from 'react';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { ProfileProperty } from './Profiles.types';
import { ResponseProfile } from '../../../lib/api/types/responses';

const useProfilesGrid = (
	profiles: ResponseProfile[],
	profilesProperties: ProfileProperty[],
	selectedProfile: string,
	onProfileClick: (mpid: string) => void,
) => {
	const profilesRows = useMemo(() => {
		// compute the rows for the grid component.
		const rows: GridRow[] = [];
		for (const profile of profiles) {
			// copy the profile to prevent changes in-place.
			let profileCopy = { ...profile };
			const isSelected = profileCopy.mpid === selectedProfile;
			const attributes = profileCopy.attributes;

			const cells: any[] = [];
			for (const p of profilesProperties) {
				if (!p.isUsed) {
					continue;
				}
				const path = p.name;
				const isSubProperty = path.includes('.');
				if (isSubProperty) {
					const parts = path.split('.');
					let v: any = attributes;
					for (const part of parts) {
						if (typeof v === 'object' && v !== null) {
							v = v[part];
						}
					}
					cells.push(v);
				} else {
					cells.push(attributes[path]);
				}
			}

			const row: GridRow = {
				onClick: () => onProfileClick(profile.mpid),
				cells: [profileCopy.sourcesLastUpdate, ...cells],
				selected: isSelected,
			};
			rows.push(row);
		}
		return rows;
	}, [profiles, profilesProperties, onProfileClick]);

	const profileColumns = useMemo(() => {
		// compute the columns for the grid component.
		const profileColumns: GridColumn[] = [];
		profileColumns.push({
			name: 'Sources last update',
			type: 'datetime',
			explanation: 'The last update time on the sources from which the profile has been imported.',
		});
		for (const p of profilesProperties) {
			if (p.isUsed) {
				profileColumns.push({
					name: p.name,
					type: p.type,
				});
			}
		}
		return profileColumns;
	}, [profilesProperties]);

	return { profilesRows: profilesRows, profileColumns: profileColumns };
};

export { useProfilesGrid };
