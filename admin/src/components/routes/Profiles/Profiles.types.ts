type ProfileTab = 'attributes' | 'events' | 'identities';

interface ProfileProperty {
	name: string;
	isUsed: boolean;
	type: string;
}

export type { ProfileProperty, ProfileTab };
