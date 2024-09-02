import { Identifiers } from './identifiers';
import { WarehouseMode } from './warehouse';

type PrivacyRegion = 'Europe' | '';

type PrimarySources = Record<string, number>;

interface DisplayedProperties {
	Image: string;
	FirstName: string;
	LastName: string;
	Information: string;
}

interface Workspace {
	ID: number;
	Name: string;
	RunIdentityResolutionOnBatchImport: boolean;
	Identifiers: Identifiers;
	PrivacyRegion: PrivacyRegion;
	DisplayedProperties: DisplayedProperties;
	WarehouseMode: WarehouseMode;
	UserPrimarySources: PrimarySources;
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
