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

const getProfilePropertyLabel = (path: string, properties: ProfileSchemaProperties): string => {
	const property = properties[path];
	if (property == null) {
		const fragments = path.split('.');
		return fragments[fragments.length - 1];
	}

	return property.displayName?.trim() || property.name;
};

const getProfilePropertyPathLabel = (path: string, properties: ProfileSchemaProperties): string => {
	let propertyPath = '';
	const labels: string[] = [];
	for (const fragment of path.split('.')) {
		propertyPath = propertyPath === '' ? fragment : `${propertyPath}.${fragment}`;
		labels.push(getProfilePropertyLabel(propertyPath, properties));
	}

	return labels.join(' › ');
};

export {
	getProfileAttributeString,
	getProfileAvatarColors,
	getProfileInitials,
	getProfilePropertyLabel,
	getProfilePropertyPathLabel,
};
