type UserTab = 'traits' | 'events' | 'identities';

interface UserProperty {
	name: string;
	isUsed: boolean;
	type: string;
}

export type { UserProperty, UserTab };
