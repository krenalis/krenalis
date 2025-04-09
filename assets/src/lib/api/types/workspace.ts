import { Identifiers } from './identifiers';
import { WarehouseMode } from './warehouse';

type PrimarySources = Record<string, number>;

interface UserProfile {
	image: string;
	firstName: string;
	lastName: string;
	extra: string;
}

interface UIPreferences {
	userProfile: UserProfile;
}

interface Workspace {
	id: number;
	name: string;
	resolveIdentitiesOnBatchImport: boolean;
	identifiers: Identifiers;
	warehouseMode: WarehouseMode;
	userPrimarySources: PrimarySources;
	uiPreferences: UIPreferences;
}

interface CreateWorkspaceResponse {
	id: number;
}

interface LatestIdentityResolution {
	startTime: string;
	endTime: string;
}

interface LatestUserSchemaUpdate {
	startTime: string;
	endTime: string;
	error: string;
}

export default Workspace;
export type {
	CreateWorkspaceResponse,
	UIPreferences,
	UserProfile,
	PrimarySources,
	LatestIdentityResolution,
	LatestUserSchemaUpdate,
};
