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
	Identifiers: Identifiers;
	PrivacyRegion: PrivacyRegion;
	DisplayedProperties: DisplayedProperties;
	WarehouseMode: WarehouseMode;
	UserPrimarySources: PrimarySources;
}

interface AddWorkspaceResponse {
	id: number;
}

export default Workspace;
export type { PrivacyRegion, AddWorkspaceResponse, DisplayedProperties, PrimarySources };
