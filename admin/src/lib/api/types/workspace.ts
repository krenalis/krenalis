import { Identifiers } from './identifiers';
import { WarehouseMode } from './warehouse';

type PrimarySources = Record<string, string>;

type ProfileRoleID = 'firstName' | 'lastName' | 'email' | 'country' | 'photo';

interface ProfileRoleAssignments {
	firstName: string;
	lastName: string;
	email: string;
	country: string;
	photo: string;
}

interface Workspace {
	assignedRoles: ProfileRoleAssignments;
	id: string;
	name: string;
	resolveIdentitiesOnBatchImport: boolean;
	identifiers: Identifiers;
	warehouseMode: WarehouseMode;
	primarySources: PrimarySources;
}

interface CreateWorkspaceResponse {
	id: string;
}

interface LatestIdentityResolution {
	startTime: string | null;
	endTime: string | null;
}

interface LatestAlterProfileSchema {
	startTime: string | null;
	endTime: string | null;
	error: string | null;
}

interface ConsentPurpose {
	code: string;
	name: string;
}

export default Workspace;
export type {
	CreateWorkspaceResponse,
	PrimarySources,
	LatestIdentityResolution,
	LatestAlterProfileSchema,
	ConsentPurpose,
	ProfileRoleAssignments,
	ProfileRoleID,
};
