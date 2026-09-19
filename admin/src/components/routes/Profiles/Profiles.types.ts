import { Property } from '../../../lib/api/types/types';

type ProfileTab = 'attributes' | 'events' | 'identities';

interface ProfileProperty {
	label: string;
	name: string;
	isUsed: boolean;
	type: string;
}

type ProfileSchemaProperties = Record<string, Property>;

export type { ProfileProperty, ProfileSchemaProperties, ProfileTab };
