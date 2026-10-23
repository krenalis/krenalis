import { createElement, useMemo } from 'react';
import { GridColumn, GridRow } from '../../base/Grid/Grid.types';
import { ProfileProperty } from './Profiles.types';
import { ResponseProfile } from '../../../lib/api/types/responses';
import { CountryFormat } from '../../../lib/api/types/types';
import { ProfileRoleAssignments } from '../../../lib/api/types/workspace';
import { ProfileCell } from './ProfileCell';

const useProfilesGrid = (
	profiles: ResponseProfile[],
	profilesProperties: ProfileProperty[],
	profilesProjection: string[],
	assignedRoles: ProfileRoleAssignments | undefined,
	countryFormat: CountryFormat | undefined,
	activeProfileID: string,
	onProfileClick: (kpid: string) => void,
) => {
	const projectedRoots = useMemo(() => new Set(profilesProjection), [profilesProjection]);
	const profilesRows = useMemo(() => {
		// compute the rows for the grid component.
		const rows: GridRow[] = [];
		for (const profile of profiles) {
			const isActive = profile.kpid === activeProfileID;
			const attributes = profile.attributes;

			const cells: any[] = [];
			for (const p of profilesProperties) {
				if (!p.isUsed || !projectedRoots.has(p.name.split('.')[0])) {
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
				onClick: () => onProfileClick(profile.kpid),
				cells: [
					createElement(ProfileCell, {
						assignedRoles,
						attributes,
						countryFormat,
						profileID: profile.kpid,
					}),
					...cells,
				],
				id: profile.kpid,
				key: profile.kpid,
				active: isActive,
			};
			rows.push(row);
		}
		return rows;
	}, [profiles, profilesProperties, projectedRoots, assignedRoles, countryFormat, activeProfileID, onProfileClick]);

	const profileColumns = useMemo(() => {
		// compute the columns for the grid component.
		const profileColumns: GridColumn[] = [];
		profileColumns.push({
			key: 'profile',
			name: 'Profile',
		});
		for (const p of profilesProperties) {
			if (p.isUsed && projectedRoots.has(p.name.split('.')[0])) {
				profileColumns.push({
					key: p.name,
					name: p.label,
					reorderable: true,
					type: p.type,
				});
			}
		}
		return profileColumns;
	}, [profilesProperties, projectedRoots]);

	return { profilesRows, profileColumns };
};

export { useProfilesGrid };
