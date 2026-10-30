import React from 'react';
import SlAvatar from '@shoelace-style/shoelace/dist/react/avatar/index.js';
import { getProfileAvatarColors, getProfileInitials } from './Profiles.helpers';

interface ProfileThumbnailProps {
	className?: string;
	firstName?: string;
	lastName?: string;
	photo?: string;
	profileID: string;
}

const ProfileThumbnail = ({ className, firstName, lastName, photo, profileID }: ProfileThumbnailProps) => {
	const initials = getProfileInitials(firstName, lastName);
	if (photo == null && initials === '') {
		return null;
	}

	const name = [firstName, lastName].filter((value) => value != null).join(' ');
	const colors = getProfileAvatarColors(profileID);
	const style = {
		'--profile-avatar-background-color': colors.backgroundColor,
		'--profile-avatar-color': colors.color,
	} as React.CSSProperties;
	const kind = photo == null ? 'avatar' : 'photo';

	return (
		<SlAvatar
			className={`profile-thumbnail profile-thumbnail--${kind}${className == null ? '' : ` ${className}`}`}
			image={photo ?? ''}
			initials={initials}
			label={name === '' ? 'Profile photo' : name}
			style={style}
		/>
	);
};

export { ProfileThumbnail };
