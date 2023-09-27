import { AnonymousIdentifiers } from './identifiers';

type PrivacyRegion = 'Europe' | '';

interface Workspace {
	ID: number;
	Name: string;
	AnonymousIdentifiers: AnonymousIdentifiers;
	PrivacyRegion: PrivacyRegion;
}

interface AddWorkspaceResponse {
	id: number;
}

export default Workspace;
export type { PrivacyRegion, AddWorkspaceResponse };
