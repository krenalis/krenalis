type AccessKeyType = 'API' | 'MCP';

interface AccessKey {
	id: number;
	workspace: number | null;
	name: string;
	type: AccessKeyType;
	token: string;
	createdAt: string;
}

export { AccessKey, AccessKeyType };
