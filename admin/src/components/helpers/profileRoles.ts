import { ProfileRoleAssignments, ProfileRoleID } from '../../lib/api/types/workspace';
import Type, { Semantic } from '../../lib/api/types/types';

interface ProfileRoleDefinition {
	description: string;
	id: ProfileRoleID;
	label: string;
}

interface ProfileRoleProperty {
	semantic?: Semantic;
	type?: Type | null;
}

const PROFILE_ROLES: readonly ProfileRoleDefinition[] = [
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

const copyProfileRoleAssignments = (assignments?: ProfileRoleAssignments): ProfileRoleAssignments => ({
	firstName: assignments?.firstName ?? '',
	lastName: assignments?.lastName ?? '',
	email: assignments?.email ?? '',
	country: assignments?.country ?? '',
	photo: assignments?.photo ?? '',
});

const getAssignedProfileRole = (
	assignments: ProfileRoleAssignments | undefined,
	propertyKey: string,
): ProfileRoleID | null => {
	if (assignments == null) {
		return null;
	}
	return PROFILE_ROLES.find((role) => assignments[role.id] === propertyKey)?.id ?? null;
};

const getCompatibleProfileRoles = (property: ProfileRoleProperty): readonly ProfileRoleDefinition[] => {
	return PROFILE_ROLES.filter((role) => isProfileRoleCompatible(role.id, property));
};

const getProfileRole = (id: ProfileRoleID): ProfileRoleDefinition => {
	return PROFILE_ROLES.find((role) => role.id === id)!;
};

const isProfileRoleCompatible = (role: ProfileRoleID, property: ProfileRoleProperty): boolean => {
	if (role === 'firstName' || role === 'lastName') {
		return property.type?.kind === 'string' && property.semantic == null;
	}
	const singleValue = property.type != null && property.type.kind !== 'array' && property.type.kind !== 'map';
	if (!singleValue) {
		return false;
	}
	switch (role) {
		case 'email':
			return property.semantic?.kind === 'email';
		case 'country':
			return property.semantic?.kind === 'country';
		case 'photo':
			return property.semantic?.kind === 'url';
	}
};

export {
	PROFILE_ROLES,
	copyProfileRoleAssignments,
	getAssignedProfileRole,
	getCompatibleProfileRoles,
	getProfileRole,
	isProfileRoleCompatible,
};
export type { ProfileRoleDefinition, ProfileRoleProperty };
