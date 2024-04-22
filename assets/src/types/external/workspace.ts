import { Identifiers } from './identifiers';

type PrivacyRegion = 'Europe' | '';

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
}

interface AddWorkspaceResponse {
	id: number;
}

export default Workspace;
export type { PrivacyRegion, AddWorkspaceResponse, DisplayedProperties };
