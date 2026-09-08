import { createContext } from 'react';
import { ProfileProperty, ProfileSchemaProperties } from '../components/routes/Profiles/Profiles.types';
import { CountProfilesResponse, ResponseProfile } from '../lib/api/types/responses';
import { Filter } from '../lib/api/types/pipeline';
import { ObjectType } from '../lib/api/types/types';

interface ProfilesContext {
	appliedFilter: Filter | null;
	previewProfilesFilter: (filter: Filter | null, signal?: AbortSignal) => Promise<CountProfilesResponse>;
	showProfilesFilter: (filter: Filter | null) => Promise<boolean>;
	profiles: ResponseProfile[];
	profilesTotal: number;
	profilesProperties: ProfileProperty[];
	profileSchema?: ObjectType;
	profileSchemaProperties: ProfileSchemaProperties;
	isLoading: boolean;
	isBoundaryLoading: boolean;
	profilesFirst: number;
	profilesLimit: number;
	profilesProjection: string[];
	profilesExecutionID: number;
	profileSchemaSessionID: number;
	resetProfilesSchema: () => void;
	profilesQueryKey: string;
	hasNextPage: boolean;
	activeProfileID: string;
	canNavigateNextProfile: boolean;
	canNavigatePreviousProfile: boolean;
	isProfileSchemaNotAligned: boolean;
	goToProfilesPage: (first: number) => void;
	markProfileSchemaNotAligned: () => void;
	navigateProfile: (direction: 'previous' | 'next') => void;
	refreshProfiles: () => void;
	reorderProfilesProperties: (properties: ProfileProperty[]) => void;
	setActiveProfileID: (profileID: string) => void;
	setProfilesPageSize: (limit: number) => void;
	updateProfilesProperties: (properties: ProfileProperty[]) => void;
}

const profilesContext = createContext<ProfilesContext>({} as ProfilesContext);

export default profilesContext;
