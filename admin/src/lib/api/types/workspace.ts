import { Identifiers } from './identifiers';
import { WarehouseMode } from './warehouse';

type PrimarySources = Record<string, number>;

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
	id: number;
	name: string;
	resolveIdentitiesOnBatchImport: boolean;
	identifiers: Identifiers;
	warehouseMode: WarehouseMode;
	primarySources: PrimarySources;
	uiPreferences: UIPreferences;
}

interface CreateWorkspaceResponse {
	id: number;
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

export default Workspace;
export type {
	CreateWorkspaceResponse,
	UIPreferences,
	Profile,
	PrimarySources,
	LatestIdentityResolution,
	LatestAlterProfileSchema,
};
