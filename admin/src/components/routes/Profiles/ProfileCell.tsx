import React from 'react';
import { getCountryName } from '../../helpers/countries';
import { CountryFormat } from '../../../lib/api/types/types';
import { ProfileRoleAssignments } from '../../../lib/api/types/workspace';
import { getProfileAttributeString } from './Profiles.helpers';
import { ProfileThumbnail } from './ProfileThumbnail';

interface ProfileCellProps {
	assignedRoles?: ProfileRoleAssignments;
	attributes: Record<string, unknown>;
	countryFormat?: CountryFormat;
	profileID: string;
}

const ProfileCell = ({ assignedRoles, attributes, countryFormat, profileID }: ProfileCellProps) => {
	const firstName = getProfileAttributeString(attributes, assignedRoles?.firstName ?? '');
	const lastName = getProfileAttributeString(attributes, assignedRoles?.lastName ?? '');
	const countryCode = getProfileAttributeString(attributes, assignedRoles?.country ?? '');
	const country = getCountryName(countryCode, countryFormat);
	const photo = getProfileAttributeString(attributes, assignedRoles?.photo ?? '');
	const name = [firstName, lastName].filter((value) => value != null).join(' ');

	return (
		<div className='profiles-list__profile-cell'>
			<ProfileThumbnail
				className='profiles-list__profile-thumbnail'
				firstName={firstName}
				lastName={lastName}
				photo={photo}
				profileID={profileID}
			/>
			{(name !== '' || country != null) && (
				<div className='profiles-list__profile-details'>
					{name !== '' && <div className='profiles-list__profile-name'>{name}</div>}
					{country != null && <div className='profiles-list__profile-country'>{country}</div>}
				</div>
			)}
		</div>
	);
};

export { ProfileCell };
