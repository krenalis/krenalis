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

interface LatestAlterProfileSchema {
	startTime: string | null;
	endTime: string | null;
	error: string | null;
}

interface ConsentPurposePipeline {
	id: string;
	name: string;
	connection: string;
	connector: string;
}

interface ConsentPurpose {
	id: string;
	name: string;
	code: string;
	pipelines: ConsentPurposePipeline[];
}

export default Workspace;
export type {
	CreateWorkspaceResponse,
	UIPreferences,
	Profile,
	PrimarySources,
	LatestIdentityResolution,
	LatestAlterProfileSchema,
	ConsentPurpose,
	ConsentPurposePipeline,
};
