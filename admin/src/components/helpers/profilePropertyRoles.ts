import { ProfilePropertyRoles, ProfilePropertyRoleID } from '../../lib/api/types/workspace';
import Type from '../../lib/api/types/types';

interface ProfilePropertyRoleDefinition {
	description: string;
	id: ProfilePropertyRoleID;
	label: string;
}

interface PropertyRoleTarget {
	type?: Type | null;
}

const PROFILE_PROPERTY_ROLES: readonly ProfilePropertyRoleDefinition[] = [
	{
		id: 'firstName',
		label: 'First name',
		description: "The profile's given name",
	},
	{
		id: 'lastName',
		label: 'Last name',
		description: "The profile's family name",
	},
	{
		id: 'email',
		label: 'Email',
		description: "The profile's email address",
	},
	{
		id: 'country',
		label: 'Country',
		description: "The profile's country address",
	},
	{
		id: 'photo',
		label: 'Photo',
		description: "The person's photo",
	},
];

const copyProfilePropertyRoles = (assignments?: ProfilePropertyRoles): ProfilePropertyRoles => ({
	firstName: assignments?.firstName ?? '',
	lastName: assignments?.lastName ?? '',
	email: assignments?.email ?? '',
	country: assignments?.country ?? '',
	photo: assignments?.photo ?? '',
});

const getPropertyRole = (
	assignments: ProfilePropertyRoles | undefined,
	propertyKey: string,
): ProfilePropertyRoleID | null => {
	if (assignments == null) {
		return null;
	}
	return PROFILE_PROPERTY_ROLES.find((role) => assignments[role.id] === propertyKey)?.id ?? null;
};

const getCompatibleProfilePropertyRoles = (property: PropertyRoleTarget): readonly ProfilePropertyRoleDefinition[] => {
	return PROFILE_PROPERTY_ROLES.filter((role) => isProfilePropertyRoleCompatible(role.id, property));
};

const getProfilePropertyRole = (id: ProfilePropertyRoleID): ProfilePropertyRoleDefinition => {
	return PROFILE_PROPERTY_ROLES.find((role) => role.id === id)!;
};

const isProfilePropertyRoleCompatible = (role: ProfilePropertyRoleID, property: PropertyRoleTarget): boolean => {
	if (role === 'firstName' || role === 'lastName') {
		return property.type?.kind === 'string' && property.type.semantics == null;
	}
	const singleValue = property.type != null && property.type.kind !== 'array' && property.type.kind !== 'map';
	if (!singleValue) {
		return false;
	}
	switch (role) {
		case 'email':
			return property.type?.kind === 'string' && property.type.semantics === 'email';
		case 'country':
			return property.type?.kind === 'string' && property.type.semantics === 'country';
		case 'photo':
			return property.type?.kind === 'string' && property.type.semantics === 'url';
	}
};

export {
	PROFILE_PROPERTY_ROLES,
	copyProfilePropertyRoles,
	getPropertyRole,
	getCompatibleProfilePropertyRoles,
	getProfilePropertyRole,
	isProfilePropertyRoleCompatible,
};
export type { PropertyRoleTarget };
