interface UserProperty {
	name: string;
	isUsed: boolean;
	type: string;
}

interface UserPagination {
	current: number;
	last: number;
}

export { UserProperty, UserPagination };
