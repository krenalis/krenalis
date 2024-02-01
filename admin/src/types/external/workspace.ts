import { Identifiers } from './identifiers';

type PrivacyRegion = 'Europe' | '';

interface Workspace {
	ID: number;
	Name: string;
	Identifiers: Identifiers;
	PrivacyRegion: PrivacyRegion;
}

interface AddWorkspaceResponse {
	id: number;
}

export default Workspace;
export type { PrivacyRegion, AddWorkspaceResponse };
