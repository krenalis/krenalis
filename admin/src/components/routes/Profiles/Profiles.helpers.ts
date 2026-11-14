import { ProfileSchemaProperties } from './Profiles.types';

interface ProfileAvatarColors {
	backgroundColor: string;
	color: string;
}

const getProfileAvatarColors = (profileID: string): ProfileAvatarColors => {
	let hash = 2166136261;
	for (let index = 0; index < profileID.length; index++) {
		hash ^= profileID.charCodeAt(index);
		hash = Math.imul(hash, 16777619);
	}

	const hue = (hash >>> 0) % 360;
	return {
		backgroundColor: `hsl(${hue} 65% 90%)`,
		color: `hsl(${hue} 55% 28%)`,
	};
};

const getProfileAttributeString = (
	attributes: Record<string, unknown> | undefined,
	path: string,
): string | undefined => {
	if (attributes == null || path === '') {
		return undefined;
	}

	let value: unknown = attributes;
	for (const fragment of path.split('.')) {
		if (typeof value !== 'object' || value === null || !Object.prototype.hasOwnProperty.call(value, fragment)) {
			return undefined;
		}
		value = (value as Record<string, unknown>)[fragment];
	}
	if (typeof value !== 'string') {
		return undefined;
	}

	const normalizedValue = value.trim();
	return normalizedValue === '' ? undefined : normalizedValue;
};

const getProfileInitials = (firstName?: string, lastName?: string): string => {
	const firstNameInitial = firstName == null ? '' : (Array.from(firstName)[0]?.toLocaleUpperCase() ?? '');
	const lastNameInitial = lastName == null ? '' : (Array.from(lastName)[0]?.toLocaleUpperCase() ?? '');

	return `${firstNameInitial}${lastNameInitial}`;
};

const getProfilePhotoSource = (photo?: string): string => {
	if (photo == null) return '';
	try {
		const url = new URL(photo);
		if (url.origin === window.location.origin && url.pathname.startsWith('/photos/'))
			return `${url.pathname}${url.search}${url.hash}`;
	} catch {
		// Preserve values that are not absolute URLs.
	}
	return photo;
};

const getProfilePropertyLabel = (path: string, properties: ProfileSchemaProperties): string => {
	const property = properties[path];
	if (property == null) {
		const fragments = path.split('.');
		return fragments[fragments.length - 1];
	}

	const displayName = property.displayName?.trim() ?? '';
	return displayName !== '' ? displayName : property.name;
};

export {
	getProfileAttributeString,
	getProfileAvatarColors,
	getProfileInitials,
	getProfilePhotoSource,
	getProfilePropertyLabel,
};
