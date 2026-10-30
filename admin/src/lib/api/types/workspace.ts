import { Identifiers } from './identifiers';
import { ObjectType } from './types';
import { WarehouseMode } from './warehouse';

type PrimarySources = Record<string, string>;

type ProfilePropertyRoleID = 'firstName' | 'lastName' | 'email' | 'country' | 'photo';

interface ProfilePropertyRoles {
	firstName: string;
	lastName: string;
	email: string;
	country: string;
	photo: string;
}

interface Workspace {
	profileSchema: ProfileSchema;
	id: string;
	name: string;
	resolveIdentitiesOnBatchImport: boolean;
	identifiers: Identifiers;
	warehouseMode: WarehouseMode;
}

interface ProfileSchema {
	schema: ObjectType;
	propertyRoles: ProfilePropertyRoles;
	primarySources: PrimarySources;
}

interface CreateWorkspaceResponse {
	id: string;
}

interface LatestIdentityResolution {
	startTime: string | null;
	endTime: string | null;
}

type IdentityResolutionRunStatus = 'running' | 'successful' | 'failed';

interface IdentityResolutionRun {
	id: string;
	status: IdentityResolutionRunStatus;
	startTime: string;
	endTime: string | null;
	error: string | null;
}

interface IdentityResolutionRunsResponse {
	runs: IdentityResolutionRun[];
}

interface LatestAlterProfileSchema {
	startTime: string | null;
	endTime: string | null;
	error: string | null;
}

interface ConsentPurpose {
	id: string;
	name: string;
	eventConsentLocations: EventConsentLocation[];
	profileConsentLocation: ProfileConsentLocation | null;
}

interface EventConsentLocation {
	purposeCode: string;
}

interface ProfileConsentLocation {
	property: string;
	jsonKey?: string;
}

export default Workspace;
export type {
	CreateWorkspaceResponse,
	PrimarySources,
	LatestIdentityResolution,
	IdentityResolutionRun,
	IdentityResolutionRunsResponse,
	IdentityResolutionRunStatus,
	LatestAlterProfileSchema,
	ConsentPurpose,
	ProfilePropertyRoles,
	ProfilePropertyRoleID,
	ProfileSchema,
	EventConsentLocation,
	ProfileConsentLocation,
};
