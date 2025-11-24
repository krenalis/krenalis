import { createContext } from 'react';
import { ProfileProperty } from '../components/routes/Profiles/Profiles.types';
import { ResponseProfile } from '../lib/api/types/responses';

interface ProfilesContext {
	profiles: ResponseProfile[];
	profilesTotal: number;
	profilesProperties: ProfileProperty[];
	isLoading: boolean;
	profileIDList: string[];
	fetchProfiles: () => Promise<string[]>;
}

const profilesContext = createContext<ProfilesContext>({} as ProfilesContext);

export default profilesContext;
