import { AnonymousIdentifiers } from './identifiers';

interface Workspace {
	ID: number;
	Name: string;
	AnonymousIdentifiers: AnonymousIdentifiers;
	PrivacyRegion: '' | 'Europe';
}

export default Workspace;
