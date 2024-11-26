import { Identifiers } from './identifiers';
import { WarehouseMode } from './warehouse';

type PrivacyRegion = 'Europe' | '';

type PrimarySources = Record<string, number>;

interface DisplayedProperties {
	image: string;
	firstName: string;
	lastName: string;
	information: string;
}

interface Workspace {
	id: number;
	name: string;
	resolveIdentitiesOnBatchImport: boolean;
	identifiers: Identifiers;
	privacyRegion: PrivacyRegion;
	displayedProperties: DisplayedProperties;
	warehouseMode: WarehouseMode;
	userPrimarySources: PrimarySources;
}

interface AddWorkspaceResponse {
	id: number;
}

interface IdentityResolutionExecution {
	startTime: string;
	endTime: string;
}

export default Workspace;
export type { PrivacyRegion, AddWorkspaceResponse, DisplayedProperties, PrimarySources, IdentityResolutionExecution };
