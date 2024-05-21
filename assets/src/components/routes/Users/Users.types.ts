type UserTab = 'traits' | 'events' | 'identities';

interface UserProperty {
	name: string;
	isUsed: boolean;
	type: string;
}

interface UserPagination {
	current: number;
	last: number;
}

export type { UserProperty, UserPagination, UserTab };
