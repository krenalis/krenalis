import { Identifiers } from './identifiers';
import { WarehouseMode } from './warehouse';

type PrimarySources = Record<string, string>;

interface Profile {
	image: string;
	firstName: string;
	lastName: string;
	extra: string;
}

interface UIPreferences {
	profile: Profile;
}

interface Workspace {
	id: string;
	name: string;
	resolveIdentitiesOnBatchImport: boolean;
	identifiers: Identifiers;
	warehouseMode: WarehouseMode;
	primarySources: PrimarySources;
	uiPreferences: UIPreferences;
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
	code: string;
	name: string;
}

export default Workspace;
export type {
	CreateWorkspaceResponse,
	UIPreferences,
	Profile,
	PrimarySources,
	LatestIdentityResolution,
	IdentityResolutionRun,
	IdentityResolutionRunsResponse,
	IdentityResolutionRunStatus,
	LatestAlterProfileSchema,
	ConsentPurpose,
};
