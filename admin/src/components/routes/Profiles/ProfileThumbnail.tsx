import React from 'react';
import WaAvatar from '@awesome.me/webawesome/dist/react/avatar/index.js';
import { getProfileAvatarColors, getProfileInitials, getProfilePhotoSource } from './Profiles.helpers';

interface ProfileThumbnailProps {
	className?: string;
	firstName?: string;
	lastName?: string;
	photo?: string;
	profileID: string;
}

const ProfileThumbnail = ({ className, firstName, lastName, photo, profileID }: ProfileThumbnailProps) => {
	const initials = getProfileInitials(firstName, lastName);
	const name = [firstName, lastName].filter((value) => value != null).join(' ');
	const colors = getProfileAvatarColors(profileID);
	const style = {
		'--profile-avatar-background-color': colors.backgroundColor,
		'--profile-avatar-color': colors.color,
	} as React.CSSProperties;
	const kind = photo == null ? 'avatar' : 'photo';

	return (
		<WaAvatar
			className={`profile-thumbnail profile-thumbnail--${kind}${className == null ? '' : ` ${className}`}`}
			image={getProfilePhotoSource(photo)}
			initials={initials}
			label={name === '' ? 'Profile photo' : name}
			style={style}
		/>
	);
};

export { ProfileThumbnail };
