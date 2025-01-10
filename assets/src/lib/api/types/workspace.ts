import { Identifiers } from './identifiers';
import { WarehouseMode } from './warehouse';

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
	displayedProperties: DisplayedProperties;
	warehouseMode: WarehouseMode;
	userPrimarySources: PrimarySources;
}

interface CreateWorkspaceResponse {
	id: number;
}

interface LastIdentityResolution {
	startTime: string;
	endTime: string;
}

export default Workspace;
export type { CreateWorkspaceResponse, DisplayedProperties, PrimarySources, LastIdentityResolution };
